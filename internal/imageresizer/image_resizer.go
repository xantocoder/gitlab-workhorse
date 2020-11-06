// Copyright 2020 GitLab Inc. All rights reserved.
// Copyright 2009 The Go Authors. All rights reserved.

package imageresizer

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/config"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/labkit/tracing"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/senddata"
)

type Resizer struct {
	config.Config
	senddata.Prefix
	numScalerProcs processCounter
}

type resizeParams struct {
	Location    string
	ContentType string
	Width       uint
}

type processCounter struct {
	n int32
}

type resizeStatus = string

type imageFile struct {
	reader        io.ReadCloser
	contentLength int64
	lastModified  time.Time
}

const (
	statusSuccess        = "success"        // a rescaled image was served
	statusScalingFailure = "scaling-failed" // scaling failed but the original image was served
	statusRequestFailure = "request-failed" // no image was served
	statusUnknown        = "unknown"        // indicates an unhandled status case
)

var envInjector = tracing.NewEnvInjector()

// Images might be located remotely in object storage, in which case we need to stream
// it via http(s)
var httpTransport = tracing.NewRoundTripper(correlation.NewInstrumentedRoundTripper(&http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 10 * time.Second,
	}).DialContext,
	MaxIdleConns:          2,
	IdleConnTimeout:       30 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 10 * time.Second,
	ResponseHeaderTimeout: 30 * time.Second,
}))

var httpClient = &http.Client{
	Transport: httpTransport,
}

const (
	namespace = "gitlab_workhorse"
	subsystem = "image_resize"
)

var (
	imageResizeConcurrencyLimitExceeds = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "concurrency_limit_exceeds_total",
			Help:      "Amount of image resizing requests that exceeded the maximum allowed scaler processes",
		},
	)
	imageResizeProcesses = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "processes",
			Help:      "Amount of image scaler processes working now",
		},
	)
	imageResizeMaxProcesses = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "max_processes",
			Help:      "The maximum amount of image scaler processes allowed to run concurrently",
		},
	)
	imageResizeRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "requests_total",
			Help:      "Image resizing operations requested",
		},
		[]string{"status"},
	)
	imageResizeDurations = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "duration_seconds",
			Help:      "Breakdown of total time spent serving successful image resizing requests (incl. data transfer)",
			Buckets: []float64{
				0.025, /* 25ms */
				0.050, /* 50ms */
				0.1,   /* 100ms */
				0.2,   /* 200ms */
				0.4,   /* 400ms */
				0.8,   /* 800ms */
			},
		},
		[]string{"content_type", "width"},
	)
)

func init() {
	prometheus.MustRegister(imageResizeConcurrencyLimitExceeds)
	prometheus.MustRegister(imageResizeProcesses)
	prometheus.MustRegister(imageResizeMaxProcesses)
	prometheus.MustRegister(imageResizeRequests)
	prometheus.MustRegister(imageResizeDurations)
}

func NewResizer(cfg config.Config) *Resizer {
	imageResizeMaxProcesses.Set(float64(cfg.ImageResizerConfig.MaxScalerProcs))

	return &Resizer{Config: cfg, Prefix: "send-scaled-img:"}
}

// Inject forks into a dedicated scaler process to resize an image identified by path or URL
// and streams the resized image back to the client
func (r *Resizer) Inject(w http.ResponseWriter, req *http.Request, paramsData string) {
	var status resizeStatus = statusUnknown
	defer func() {
		imageResizeRequests.WithLabelValues(status).Inc()
	}()

	start := time.Now()
	logger := log.ContextLogger(req.Context())
	params, err := r.unpackParameters(paramsData)
	if err != nil {
		// This means the response header coming from Rails was malformed; there is no way
		// to sensibly recover from this other than failing fast
		status = statusRequestFailure
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Failed reading image resize params: %v", err))
		return
	}

	imageFile, err := openSourceImage(req, params.Location)
	if err != nil {
		// This means we cannot even read the input image; fail fast.
		status = statusRequestFailure
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Failed opening image data stream: %v", err))
		return
	}
	defer imageFile.reader.Close()

	logFields := func(bytesWritten int64) *log.Fields {
		return &log.Fields{
			"bytes_written":     bytesWritten,
			"duration_s":        time.Since(start).Seconds(),
			"target_width":      params.Width,
			"content_type":      params.ContentType,
			"original_filesize": imageFile.contentLength,
		}
	}

	setLastModified(w, imageFile.lastModified)
	// If the original file has not changed, then any cached resized versions have not changed either.
	if checkNotModified(req, imageFile.lastModified) {
		logger.WithFields(*logFields(0)).Printf("ImageResizer: Use cached image")
		writeNotModified(w)
		return
	}

	// We first attempt to rescale the image; if this should fail for any reason, imageReader
	// will point to the original image, i.e. we render it unchanged.
	imageReader, resizeCmd, err := r.tryResizeImage(req, imageFile, logger.Writer(), params, r.Config.ImageResizerConfig)
	if err != nil {
		// Something failed, but we can still write out the original image, so don't return early.
		helper.LogErrorWithFields(req, err, *logFields(0))
	}
	defer helper.CleanUpProcessGroup(resizeCmd)

	w.Header().Del("Content-Length")
	bytesWritten, err := serveImage(imageReader, w, resizeCmd)

	// We failed serving image data; this is a hard failure.
	if err != nil {
		status = statusRequestFailure
		if bytesWritten <= 0 {
			helper.Fail500(w, req, err)
		} else {
			helper.LogErrorWithFields(req, err, *logFields(bytesWritten))
		}
		return
	}

	// This means we served the original image because rescaling failed; this is a soft failure
	if resizeCmd == nil {
		status = statusScalingFailure
		logger.WithFields(*logFields(bytesWritten)).Printf("ImageResizer: Served original")
		return
	}

	widthLabelVal := strconv.Itoa(int(params.Width))
	imageResizeDurations.WithLabelValues(params.ContentType, widthLabelVal).Observe(time.Since(start).Seconds())

	logger.WithFields(*logFields(bytesWritten)).Printf("ImageResizer: Success")

	status = statusSuccess
}

// Streams image data from the given reader to the given writer and returns the number of bytes written.
// Errors are either served to the caller or merely logged, depending on whether any image data had
// already been transmitted or not.
func serveImage(r io.Reader, w io.Writer, resizeCmd *exec.Cmd) (int64, error) {
	bytesWritten, err := io.Copy(w, r)
	if err != nil {
		return bytesWritten, err
	}

	if resizeCmd != nil {
		return bytesWritten, resizeCmd.Wait()
	}

	return bytesWritten, nil
}

func (r *Resizer) unpackParameters(paramsData string) (*resizeParams, error) {
	var params resizeParams
	if err := r.Unpack(&params, paramsData); err != nil {
		return nil, err
	}

	if params.Location == "" {
		return nil, fmt.Errorf("ImageResizer: Location is empty")
	}

	if params.ContentType == "" {
		return nil, fmt.Errorf("ImageResizer: ContentType must be set")
	}

	return &params, nil
}

// Attempts to rescale the given image data, or in case of errors, falls back to the original image.
func (r *Resizer) tryResizeImage(req *http.Request, f *imageFile, errorWriter io.Writer, params *resizeParams, cfg config.ImageResizerConfig) (io.Reader, *exec.Cmd, error) {
	if f.contentLength > int64(cfg.MaxFilesize) {
		return f.reader, nil, fmt.Errorf("ImageResizer: %db exceeds maximum file size of %db", f.contentLength, cfg.MaxFilesize)
	}

	if !r.numScalerProcs.tryIncrement(int32(cfg.MaxScalerProcs)) {
		return f.reader, nil, fmt.Errorf("ImageResizer: too many running scaler processes (%d / %d)", r.numScalerProcs.n, cfg.MaxScalerProcs)
	}

	ctx := req.Context()
	go func() {
		<-ctx.Done()
		r.numScalerProcs.decrement()
	}()

	resizeCmd, resizedImageReader, err := startResizeImageCommand(ctx, f.reader, errorWriter, params)
	if err != nil {
		return f.reader, nil, fmt.Errorf("ImageResizer: failed forking into scaler process: %w", err)
	}
	return resizedImageReader, resizeCmd, nil
}

func startResizeImageCommand(ctx context.Context, imageReader io.Reader, errorWriter io.Writer, params *resizeParams) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, "gitlab-resize-image")
	cmd.Stdin = imageReader
	cmd.Stderr = errorWriter
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = []string{
		"GL_RESIZE_IMAGE_WIDTH=" + strconv.Itoa(int(params.Width)),
		"GL_RESIZE_IMAGE_CONTENT_TYPE=" + params.ContentType,
	}
	cmd.Env = envInjector(ctx, cmd.Env)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	return cmd, stdout, nil
}

func isURL(location string) bool {
	return strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://")
}

func openSourceImage(req *http.Request, location string) (*imageFile, error) {
	if isURL(location) {
		return openFromURL(location)
	}

	return openFromFile(req, location)
}

func openFromURL(location string) (*imageFile, error) {
	res, err := httpClient.Get(location)
	if err != nil {
		return nil, err
	}

	switch res.StatusCode {
	case http.StatusOK, http.StatusNotModified:
		// Extract headers for conditional GETs from response.
		lastModified, err := http.ParseTime(res.Header.Get("Last-Modified"))
		if err != nil {
			// This is unlikely to happen, coming from an object storage provider.
			lastModified = time.Now().UTC()
		}
		return &imageFile{res.Body, res.ContentLength, lastModified}, nil
	default:
		res.Body.Close()
		return nil, fmt.Errorf("unexpected upstream response for %q: %d %s",
			location, res.StatusCode, res.Status)
	}
}

func openFromFile(req *http.Request, location string) (*imageFile, error) {
	file, err := os.Open(location)
	if err != nil {
		return nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}

	return &imageFile{file, fi.Size(), fi.ModTime()}, nil
}

// Only allow more scaling requests if we haven't yet reached the maximum
// allowed number of concurrent scaler processes
func (c *processCounter) tryIncrement(maxScalerProcs int32) bool {
	if p := atomic.AddInt32(&c.n, 1); p > maxScalerProcs {
		c.decrement()
		imageResizeConcurrencyLimitExceeds.Inc()

		return false
	}

	imageResizeProcesses.Set(float64(c.n))
	return true
}

func (c *processCounter) decrement() {
	atomic.AddInt32(&c.n, -1)
	imageResizeProcesses.Set(float64(c.n))
}

func checkNotModified(r *http.Request, modtime time.Time) bool {
	ims := r.Header.Get("If-Modified-Since")
	if ims == "" || isZeroTime(modtime) {
		// Treat bogus times as if there was no such header at all
		return false
	}
	t, err := http.ParseTime(ims)
	if err != nil {
		return false
	}
	// The Last-Modified header truncates sub-second precision so
	// the modtime needs to be truncated too.
	return !modtime.Truncate(time.Second).After(t)
}

// isZeroTime reports whether t is obviously unspecified (either zero or Unix epoch time).
func isZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(time.Unix(0, 0))
}

func setLastModified(w http.ResponseWriter, modtime time.Time) {
	if !isZeroTime(modtime) {
		w.Header().Set("Last-Modified", modtime.UTC().Format(http.TimeFormat))
	}
}

func writeNotModified(w http.ResponseWriter) {
	h := w.Header()
	delete(h, "Content-Type")
	delete(h, "Content-Length")
	w.WriteHeader(http.StatusNotModified)
}
