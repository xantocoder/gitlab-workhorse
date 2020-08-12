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
	// Only allow more scaling requests if we haven't yet reached the maximum allows number
	// of concurrent graphicsmagick processes
	numScalerProcs := atomic.AddInt32(&numScalerProcs, 1)
	defer atomic.AddInt32(&numScalerProcs, -1)
	if numScalerProcs > maxImageScalerProcs {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: too many image resize requests (max %d)", maxImageScalerProcs))
		return
	}

	logger := log.ContextLogger(req.Context())

	params, err := r.unpackParameters(paramsData)
	if err != nil {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Failed reading image resize params: %v", err))
		return
	}

	inImageReader, err := readSourceImageData(params.Location)
	if err != nil {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Failed opening image data stream: %v", err))
		return
	}

	resizeCmd, outImageReader, err := startResizeImageCommand(inImageReader, logger.Writer(), params.Width)
	if err != nil {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Failed forking into graphicsmagick: %v", err))
		return
	}

	defer helper.CleanUpProcessGroup(resizeCmd)

	// TODO: Double-check it. I noticed we do it some injectors.
	// Without it, I fail with "http: wrote more than the declared Content-Length"
	//w.Header().Del("Content-Length")

	bytesWritten, err := io.Copy(w, outImageReader)

	if bytesWritten == 0 {
		// we can only write out a full 500 if we haven't already  tried to serve the image
		helper.Fail500(w, req, err)
		return
	}

	if err != nil {
		// Is there a better way to recover from this, since we will abort mid-stream?
		logger.Errorf("Failed serving image data to client after %d bytes: %v", bytesWritten, err)
		return
	}

	logger.Infof("Served send-scaled-img request (bytes written: %d)", bytesWritten)
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
