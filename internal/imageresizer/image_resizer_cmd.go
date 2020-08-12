package imageresizer

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync/atomic"
	"syscall"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/senddata"
)

type resizer struct{ senddata.Prefix }

var ImageResizerCmd = &resizer{"send-scaled-img:"}

type resizeParams struct {
	Location string
	Width    uint
}

const maxImageScalerProcs = 100

var numScalerProcs int32 = 0

func (r *resizer) Inject(w http.ResponseWriter, req *http.Request, paramsData string) {
	if atomic.AddInt32(&numScalerProcs, 1) > maxImageScalerProcs {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: too many image resize requests (max %d)", maxImageScalerProcs))
		return
	}
	defer atomic.AddInt32(&numScalerProcs, -1)

	logger := log.ContextLogger(req.Context())

	params, err := r.unpackParameters(paramsData)
	if err != nil {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Failed reading image resize params: %v", err))
		return		
	}

	inImageReader, err := ReadAllData(params.Location)
	if err != nil {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Failed opening image data stream: %v", err))
		return
	}

	resizeCmd, outImageReader, err := r.startResizeImageCommand(inImageReader, logger.Writer(), params.Width)
	if err != nil {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Failed forking into graphicsmagick: %v", err))
		return
	}

	defer helper.CleanUpProcessGroup(resizeCmd)

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

func (r * resizer) startResizeImageCommand(imageReader io.Reader, errorWriter io.Writer, width uint) (*exec.Cmd, io.ReadCloser, error) {
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