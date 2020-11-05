package imageresizer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/config"

	"gitlab.com/gitlab-org/labkit/log"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/testhelper"
)

func TestMain(m *testing.M) {
	if err := testhelper.BuildExecutables(); err != nil {
		log.WithError(err).Fatal()
	}

	os.Exit(m.Run())
}

func requestScaledImage(t *testing.T, requestURL string, httpHeaders http.Header, params resizeParams, cfg config.ImageResizerConfig) *httptest.ResponseRecorder {
	requestHandler := func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)

		paramsJSON := encodeParams(t, &params)
		NewResizer(config.Config{ImageResizerConfig: cfg}).Inject(w, r, paramsJSON)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/image", requestHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	httpRequest, err := http.NewRequest("GET", server.URL+requestURL, nil)
	require.NoError(t, err)
	if httpHeaders != nil {
		httpRequest.Header = httpHeaders
	}

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httpRequest)
	return response
}

func TestRequestScaledImageFromPath(t *testing.T) {
	cfg := config.DefaultImageResizerConfig
	params := resizeParams{Location: "../../testdata/image.png", ContentType: "image/png", Width: 64}
	resp := requestScaledImage(t, "/image", nil, params, cfg)

	require.Equal(t, http.StatusOK, resp.Code)

	img, err := png.Decode(bytes.NewReader(resp.Body.Bytes()))
	require.NoError(t, err, "decode resized image")

	bounds := img.Bounds()
	require.Equal(t, int(params.Width), bounds.Max.X-bounds.Min.X, "wrong width after resizing")
}

func TestServeOriginalImageWhenSourceImageTooLarge(t *testing.T) {
	originalImage, err := png.Decode(testImage(t))
	require.NoError(t, err, "decode original image")

	cfg := config.ImageResizerConfig{MaxScalerProcs: 1, MaxFilesize: 1}
	params := resizeParams{Location: "../../testdata/image.png", ContentType: "image/png", Width: 64}
	resp := requestScaledImage(t, "/image", nil, params, cfg)

	require.Equal(t, http.StatusOK, resp.Code)

	img, err := png.Decode(bytes.NewReader(resp.Body.Bytes()))
	require.NoError(t, err, "decode resized image")

	require.Equal(t, originalImage.Bounds(), img.Bounds(), "expected original image size")
}

func TestFailFastOnContentTypeMismatch(t *testing.T) {
	cfg := config.DefaultImageResizerConfig
	params := resizeParams{Location: "../../testdata/image.png", ContentType: "image/jpeg", Width: 64}
	resp := requestScaledImage(t, "/image", nil, params, cfg)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestUnpackParametersReturnsParamsInstanceForValidInput(t *testing.T) {
	r := Resizer{}
	inParams := resizeParams{Location: "/path/to/img", Width: 64, ContentType: "image/png"}

	outParams, err := r.unpackParameters(encodeParams(t, &inParams))

	require.NoError(t, err, "unexpected error when unpacking params")
	require.Equal(t, inParams, *outParams)
}

func TestUnpackParametersReturnsErrorWhenLocationBlank(t *testing.T) {
	r := Resizer{}
	inParams := resizeParams{Location: "", Width: 64, ContentType: "image/jpg"}

	_, err := r.unpackParameters(encodeParams(t, &inParams))

	require.Error(t, err, "expected error when Location is blank")
}

func TestUnpackParametersReturnsErrorWhenContentTypeBlank(t *testing.T) {
	r := Resizer{}
	inParams := resizeParams{Location: "/path/to/img", Width: 64, ContentType: ""}

	_, err := r.unpackParameters(encodeParams(t, &inParams))

	require.Error(t, err, "expected error when ContentType is blank")
}

// The Rails applications sends a Base64 encoded JSON string carrying
// these parameters in an HTTP response header
func encodeParams(t *testing.T, p *resizeParams) string {
	json, err := json.Marshal(*p)
	if err != nil {
		require.NoError(t, err, "JSON encoder encountered unexpected error")
	}
	return base64.StdEncoding.EncodeToString(json)
}

func testImage(t *testing.T) *os.File {
	f, err := os.Open("../../testdata/image.png")
	require.NoError(t, err)
	return f
}
