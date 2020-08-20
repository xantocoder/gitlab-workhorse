package imageresizer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"gitlab.com/gitlab-org/labkit/log"

	"github.com/stretchr/testify/require"
)

var r = resizer{}
var logger = log.ContextLogger(context.TODO())

func TestUnpackParametersReturnsParamsInstanceForValidInput(t *testing.T) {
	inParams := resizeParams{Location: "/path/to/img", Width: 64, Format: "image/png"}

	outParams, err := r.unpackParameters(encodeParams(t, &inParams))

	require.NoError(t, err, "unexpected error when unpacking params")
	require.Equal(t, inParams, *outParams)
}

func TestUnpackParametersReturnsErrorWhenLocationBlank(t *testing.T) {
	inParams := resizeParams{Location: "", Width: 64, Format: "image/jpg"}

	_, err := r.unpackParameters(encodeParams(t, &inParams))

	require.Error(t, err, "expected error when Location is blank")
}

func TestUnpackParametersReturnsErrorWhenFormatBlank(t *testing.T) {
	inParams := resizeParams{Location: "/path/to/img", Width: 64, Format: ""}

	_, err := r.unpackParameters(encodeParams(t, &inParams))

	require.Error(t, err, "expected error when Format is blank")
}

func TestDetermineFilePrefixFromMimeType(t *testing.T) {
	require.Equal(t, "png:", determineFilePrefix("image/png"))
	require.Equal(t, "jpg:", determineFilePrefix("image/jpeg"))
	require.Equal(t, "", determineFilePrefix("unsupported"))
}

func TestTryResizeImageSuccess(t *testing.T) {
	inParams := resizeParams{Location: "/path/to/img", Width: 64, Format: "image/png"}
	inFile := testImage(t)

	reader, cmd := tryResizeImage(context.TODO(), inFile, &inParams, logger)

	require.NotNil(t, cmd)
	require.NotNil(t, reader)
	require.NotEqual(t, inFile, reader)
}

func TestTryResizeImageFailsOverToOriginalImageWhenFormatNotSupported(t *testing.T) {
	inParams := resizeParams{Location: "/path/to/img", Width: 64, Format: "not supported"}
	inFile := testImage(t)

	reader, cmd := tryResizeImage(context.TODO(), inFile, &inParams, logger)

	require.Nil(t, cmd)
	require.Equal(t, inFile, reader)
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
