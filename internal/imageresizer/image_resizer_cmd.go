package imageresizer

import (
	"os"
	"os/exec"
	"fmt"
	"net/http"
	"strconv"
	"syscall"
	"io"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/senddata"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
)

type resizer struct{ senddata.Prefix }

var ImageResizerCmd = &resizer{"image-resizer:"}

type resizeParams struct {
	Path  string
	Width uint
}

func (r *resizer) Inject(w http.ResponseWriter, req *http.Request, paramsData string) {
	var params resizeParams
	fmt.Println("Image params:", paramsData)

	if err := r.Unpack(&params, paramsData); err != nil {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: unpack paramsData: %v", err))
		return
	}

	if params.Path == "" {
		helper.Fail500(w, req, fmt.Errorf("ImageResizer: Path is empty"))
		return
	}

	// Set up environment, run `cmd/resize-image`
	resizeCmd := exec.Command("gitlab-resize-image")
	resizeCmd.Env = append(os.Environ(),
		"WH_RESIZE_IMAGE_URL=" + params.Path,
		"WH_RESIZE_IMAGE_WIDTH=" + strconv.Itoa(int(params.Width)),
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

	if err != nil {
		if (bytesWritten == 0) {
			// we can only write out a full 500 if we haven't already  tried to serve the image
			helper.Fail500(w, req, err)
			return
		} else {
			// Is there a better way to recover from this, since we will abort mid-stream?
			logger.Errorf("Failed serving image data to client after %d bytes: %v", bytesWritten, err)
			return
		}
	}

	logger.Debugf("Served resized image (bytes written: %d)", bytesWritten)
}
