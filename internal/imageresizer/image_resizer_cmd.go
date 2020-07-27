package imageresizer

import (
	"os"
	"os/exec"
	"fmt"
	"net/http"
	"strconv"

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
	resizeCmd.Stderr = log.ContextLogger(req.Context()).Writer()
	resizedImg, err := resizeCmd.Output()
	if err != nil {
		helper.Fail500(w, req, fmt.Errorf("start %v: %v", resizeCmd.Args, err))
		return
	}

	fmt.Println("Image resized; bytes received:", len(resizedImg))
	//TODO: do we need this?
	defer helper.CleanUpProcessGroup(resizeCmd)

	// Serve resized image
	w.WriteHeader(http.StatusOK)
	w.Write(resizedImg)
}
