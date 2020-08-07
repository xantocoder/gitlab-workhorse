package imageresizer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
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

const maxImageScalerProcs = 30

var numScalerProcs int32 = 0

func (r *resizer) Inject(w http.ResponseWriter, req *http.Request, paramsData string) {
	if atomic.AddInt32(&numScalerProcs, 1) > maxImageScalerProcs {
		helper.Fail500(w, req, fmt.Errorf("Too many image resize requests (max %d)", maxImageScalerProcs))
		return
	}
	var params resizeParams

	if err := r.Unpack(&params, paramsData); err != nil {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: unpack paramsData: %v", err))
		return
	}

	if params.Location == "" {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Location is empty"))
		return
	}

	// Set up environment, run `cmd/resize-image`
	resizeCmd := exec.Command("gitlab-resize-image")
	resizeCmd.Env = append(os.Environ(),
		"WH_RESIZE_IMAGE_LOCATION="+params.Location,
		"WH_RESIZE_IMAGE_WIDTH="+strconv.Itoa(int(params.Width)),
	)
	logger := log.ContextLogger(req.Context())
	resizeCmd.Stderr = logger.Writer()
	resizeCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := resizeCmd.StdoutPipe()
	if err != nil {
		helper.Fail500(w, req, fmt.Errorf("create gitlab-resize-image stdout pipe: %v", err))
		return
	}

	if err := resizeCmd.Start(); err != nil {
		helper.Fail500(w, req, fmt.Errorf("start %v: %v", resizeCmd.Args, err))
		return
	}
	defer helper.CleanUpProcessGroup(resizeCmd)

	bytesWritten, err := io.Copy(w, stdout)

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

	atomic.AddInt32(&numScalerProcs, -1)

	logger.Infof("Served send-scaled-img request (bytes written: %d)", bytesWritten)
}
