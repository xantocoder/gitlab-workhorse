package imageresizer

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/labkit/tracing"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/senddata"
)

type resizer struct{ senddata.Prefix }

var SendScaledImage = &resizer{"send-scaled-img:"}

type resizeParams struct {
	Location string
	Width    uint
}

const maxImageScalerProcs = 100

var numScalerProcs int32 = 0

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

// This Injecter forks into graphicsmagick to resize an image identified by path or URL
// and streams the resized image back to the client
func (r *resizer) Inject(w http.ResponseWriter, req *http.Request, paramsData string) {
	logger := log.ContextLogger(req.Context())
	params, err := r.unpackParameters(paramsData)
	if err != nil {
		// This means the response header coming from Rails was malformed; there is no way
		// to sensibly recover from this other than failing fast
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Failed reading image resize params: %v", err))
		return
	}

	sourceImageReader, err := readSourceImageData(params.Location)
	if err != nil {
		// This means we cannot even read the input image; fail fast.
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Failed opening image data stream: %v", err))
		return
	}

	// Past this point we attempt to rescale the image; if this should fail for any reason, we
	// simply fail over to rendering out the original image unchanged.
	defer atomic.AddInt32(&numScalerProcs, -1)
	imageReader, resizeCmd := tryResizeImage(sourceImageReader, params.Width, logger)
	defer helper.CleanUpProcessGroup(resizeCmd)

	bytesWritten, err := writeTargetImageData(w, imageReader)
	if bytesWritten == 0 {
		// we can only write out a full 500 if we haven't already  tried to serve the image
		helper.Fail500(w, req, err)
		return
	}

	logger.Infof("ImageResizer: bytes written: %d", bytesWritten)
}

func (r *resizer) unpackParameters(paramsData string) (*resizeParams, error) {
	var params resizeParams
	if err := r.Unpack(&params, paramsData); err != nil {
		return nil, err
	}

	if params.Location == "" {
		return nil, fmt.Errorf("ImageResizer: Location is empty")
	}

	return &params, nil
}

// Attempts to rescale the given image data, or in case of errors, falls back to the original image.
func tryResizeImage(r io.Reader, width uint, logger *logrus.Entry) (io.Reader, *exec.Cmd) {
	// Only allow more scaling requests if we haven't yet reached the maximum allows number
	// of concurrent graphicsmagick processes
	if numScalerProcs := atomic.AddInt32(&numScalerProcs, 1); numScalerProcs > maxImageScalerProcs {
		logger.Errorf("ImageResizer: too many image resize requests (cur: %d, max %d)", numScalerProcs, maxImageScalerProcs)
		return r, nil
	}

	resizeCmd, resizedImageReader, err := startResizeImageCommand(r, logger.Writer(), width)
	if err != nil {
		logger.Errorf("ImageResizer: failed forking into graphicsmagick: %v", err)
		return r, nil
	}
	return resizedImageReader, resizeCmd
}

func startResizeImageCommand(imageReader io.Reader, errorWriter io.Writer, width uint) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.Command("gm", "convert", "-resize", fmt.Sprintf("%dx", width), "-", "-")
	cmd.Stdin = imageReader
	cmd.Stderr = errorWriter
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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

func readSourceImageData(location string) (io.Reader, error) {
	if !isURL(location) {
		return os.Open(location)
	}

	res, err := httpClient.Get(location)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusOK {
		return res.Body, nil
	}

	return nil, fmt.Errorf("ImageResizer: cannot read data from %q: %d %s",
		location, res.StatusCode, res.Status)
}

func writeTargetImageData(w http.ResponseWriter, r io.Reader) (int64, error) {
	// TODO: Double-check it. I noticed we do it in some injectors.
	// Without it, I fail with "http: wrote more than the declared Content-Length" in the `io.Copy`.
	// I still receive "Content-Length" header with this change (checked locally).
	w.Header().Del("Content-Length")

	bytesWritten, err := io.Copy(w, r)

	if err != nil {
		// Is there a better way to recover from this, since we will abort mid-stream?
		return 0, fmt.Errorf("failed serving image data to client after %d bytes: %v", bytesWritten, err)
	}

	return bytesWritten, nil
}
