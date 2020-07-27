package imageresizer

import (
	"fmt"
	"net/http"
	"os"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/senddata"
)

type entry struct{ senddata.Prefix }

type entryParams struct {
	Path  string
	Width uint
}

var ImageResizer = &entry{"image-resizer:"}

func (e *entry) Inject(w http.ResponseWriter, r *http.Request, paramsData string) {
	var params entryParams

	// Handle / sanitize parameters
	fmt.Println("Image params:", paramsData)

	if err := e.Unpack(&params, paramsData); err != nil {
		helper.Fail500(w, r, fmt.Errorf("ImageResizer: unpack paramsData: %v", err))
		return
	}

	if params.Path == "" {
		helper.Fail500(w, r, fmt.Errorf("ImageResizer: Path is empty"))
		return
	}

	// TODO: maybe we shouldn't even do anything if the image has the desired size alredy?

	// Read image data
	data, err := readAllData(params.Path)
	if err != nil {
		helper.Fail500(w, r, fmt.Errorf("ImageResizer: cannot read data: %v", err))
		return
	}

	// Resize it
	resizeImplementation := os.Getenv("GITLAB_IMAGE_RESIZER")

	resizedImg, format, err := resizeImage(data, params.Width, resizeImplementation)
	if err != nil {
		helper.LogError(r, err)
		w.WriteHeader(http.StatusOK)
		w.Write(data) // unsafe, as we don't check result
		return
	}

	fmt.Println("Image resized, format:", format)

	w.WriteHeader(http.StatusOK)
	w.Write(resizedImg)
}
