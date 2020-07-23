package imageresizer

import (
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/senddata"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/tracing"
)

type entry struct{ senddata.Prefix }

type entryParams struct {
	Path  string
	Width uint
}

var ImageResizer = &entry{"image-resizer:"}

// httpTransport defines a http.Transport with values
// that are more restrictive than for http.DefaultTransport,
// they define shorter TLS Handshake, and more aggressive connection closing
// to prevent the connection hanging and reduce FD usage
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

func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

func readAllData(path string) ([]byte, error) {
	// TODO: super unsafe: size, and path no validation of the source
	if !isURL(path) {
		return ioutil.ReadFile(path)
	}

	res, err := httpClient.Get(path)
	if err != nil {
		return nil, err
	}
	defer io.Copy(ioutil.Discard, res.Body)
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return ioutil.ReadAll(res.Body)
	}

	return nil, fmt.Errorf("cannot read data from %q: %d %s",
		path, res.StatusCode, res.Status)
}

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

	switch format {
	case ImageFormatPNG:
		w.WriteHeader(http.StatusOK)
		err := png.Encode(w, resizedImg)
		helper.LogError(r, err)

	case ImageFormatJPEG:
		w.WriteHeader(http.StatusOK)
		err := jpeg.Encode(w, resizedImg, nil)
		helper.LogError(r, err)

	default:
		helper.Fail500(w, r, fmt.Errorf("Unexpected format %v", format))
	}
}
