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
	"github.com/sirupsen/logrus"

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
	logPrefix = "imgresizer."
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

// This Injecter forks into a dedicated scaler process to resize an image identified by path or URL
// and streams the resized image back to the client
func (r *Resizer) Inject(w http.ResponseWriter, req *http.Request, paramsData string) {
	var status resizeStatus = statusUnknown
	defer func() {
		imageResizeRequests.WithLabelValues(status).Inc()
	}()

	start := time.Now()
	logger := log.ContextLogger(req.Context())
	logInfo := logInfoFn(logger, w, req, start)
	logError := logErrorFn(w, req, start)
	render500 := render500Fn(w, req, start)

	params, err := r.unpackParameters(paramsData)
	if err != nil {
		// This means the response header coming from Rails was malformed; there is no way
		// to sensibly recover from this other than failing fast
		status = statusRequestFailure
		render500(fmt.Errorf("read image resize params: %v", err), 0, params, 0)
		return
	}

	sourceImageReader, fileSize, err := openSourceImage(params.Location)
	if err != nil {
		// This means we cannot even read the input image; fail fast.
		status = statusRequestFailure
		render500(fmt.Errorf("open image data stream: %v", err), 0, params, fileSize)
		return
	}
	defer sourceImageReader.Close()

	// We first attempt to rescale the image; if this should fail for any reason, imageReader
	// will point to the original image, i.e. we render it unchanged.
	imageReader, resizeCmd, err := r.tryResizeImage(req, sourceImageReader, logger.Writer(), params, fileSize, r.Config.ImageResizerConfig)
	if err != nil {
		// Something failed, but we can still write out the original image, so don't return early.
		logError(err, 0, params, fileSize)
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
			logError(err, bytesWritten, params, fileSize)
		}
		return
	}

	// This means we served the original image because rescaling failed; this is a soft failure
	if resizeCmd == nil {
		status = statusScalingFailure
		logInfo("served original", bytesWritten, params, fileSize)
		return
	}

	widthLabelVal := strconv.Itoa(int(params.Width))
	imageResizeDurations.WithLabelValues(params.ContentType, widthLabelVal).Observe(time.Since(start).Seconds())

	logInfo("success", bytesWritten, params, fileSize)

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
		return nil, fmt.Errorf("'Location' not set")
	}

	if params.ContentType == "" {
		return nil, fmt.Errorf("'ContentType' must be set")
	}

	return &params, nil
}

// Attempts to rescale the given image data, or in case of errors, falls back to the original image.
func (r *Resizer) tryResizeImage(req *http.Request, reader io.Reader, errorWriter io.Writer, params *resizeParams, fileSize int64, cfg config.ImageResizerConfig) (io.Reader, *exec.Cmd, error) {
	if fileSize > int64(cfg.MaxFilesize) {
		return reader, nil, fmt.Errorf("%d bytes exceeds maximum file size of %d bytes", fileSize, cfg.MaxFilesize)
	}

	if !r.numScalerProcs.tryIncrement(int32(cfg.MaxScalerProcs)) {
		return reader, nil, fmt.Errorf("too many running scaler processes (%d / %d)", r.numScalerProcs.n, cfg.MaxScalerProcs)
	}

	ctx := req.Context()
	go func() {
		<-ctx.Done()
		r.numScalerProcs.decrement()
	}()

	resizeCmd, resizedImageReader, err := startResizeImageCommand(ctx, reader, errorWriter, params)
	if err != nil {
		return reader, nil, fmt.Errorf("fork into scaler process: %w", err)
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

func openSourceImage(location string) (io.ReadCloser, int64, error) {
	if isURL(location) {
		return openFromURL(location)
	}

	return openFromFile(location)
}

func openFromURL(location string) (io.ReadCloser, int64, error) {
	res, err := httpClient.Get(location)
	if err != nil {
		return nil, 0, err
	}

	if res.StatusCode != http.StatusOK {
		res.Body.Close()

		return nil, 0, fmt.Errorf("cannot read data from %q: %d %s",
			location, res.StatusCode, res.Status)
	}

	return res.Body, res.ContentLength, nil
}

func openFromFile(location string) (io.ReadCloser, int64, error) {
	file, err := os.Open(location)

	if err != nil {
		return file, 0, err
	}

	fi, err := file.Stat()
	if err != nil {
		return file, 0, err
	}

	return file, fi.Size(), nil
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

func logFields(startTime time.Time, params *resizeParams, bytesWritten int64, fileSize int64) *log.Fields {
	var targetWidth, contentType string
	if params != nil {
		targetWidth = fmt.Sprint(params.Width)
		contentType = fmt.Sprint(params.ContentType)
	}
	return &log.Fields{
		"subsystem":                     "imageresizer",
		"written_bytes":                 bytesWritten,
		"duration_s":                    time.Since(startTime).Seconds(),
		logPrefix + "target_width":      targetWidth,
		logPrefix + "content_type":      contentType,
		logPrefix + "original_filesize": fileSize,
	}
}

func logInfoFn(logger *logrus.Entry, w http.ResponseWriter, req *http.Request, startTime time.Time) func(msg string, bytesWritten int64, p *resizeParams, fileSize int64) {
	return func(msg string, bytesWritten int64, params *resizeParams, fileSize int64) {
		logger.WithFields(*logFields(startTime, params, bytesWritten, fileSize)).Printf(msg)
	}
}

func logErrorFn(w http.ResponseWriter, req *http.Request, startTime time.Time) func(err error, bytesWritten int64, p *resizeParams, fileSize int64) {
	return func(err error, bytesWritten int64, params *resizeParams, fileSize int64) {
		helper.LogErrorWithFields(req, err, *logFields(startTime, params, bytesWritten, fileSize))
	}
}

func render500Fn(w http.ResponseWriter, req *http.Request, startTime time.Time) func(err error, bytesWritten int64, p *resizeParams, fileSize int64) {
	return func(err error, bytesWritten int64, params *resizeParams, fileSize int64) {
		helper.Fail500WithFields(w, req, err, *logFields(startTime, params, bytesWritten, fileSize))
	}
}
