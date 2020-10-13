package staticpages

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/testhelper"

	"github.com/stretchr/testify/require"
)

func TestServingNonExistingFile(t *testing.T) {
	dir := "/path/to/non/existing/directory"
	httpRequest, _ := http.NewRequest("GET", "/file", nil)

	w := httptest.NewRecorder()
	st := &Static{dir, ""}
	st.ServeExisting("/", CacheDisabled, nil).ServeHTTP(w, httpRequest)
	require.Equal(t, 404, w.Code)
}

func TestServingDirectory(t *testing.T) {
	dir, err := ioutil.TempDir("", "deploy")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	httpRequest, _ := http.NewRequest("GET", "/file", nil)
	w := httptest.NewRecorder()
	st := &Static{dir, ""}
	st.ServeExisting("/", CacheDisabled, nil).ServeHTTP(w, httpRequest)
	require.Equal(t, 404, w.Code)
}

func TestServingMalformedUri(t *testing.T) {
	dir := "/path/to/non/existing/directory"
	httpRequest, _ := http.NewRequest("GET", "/../../../static/file", nil)

	w := httptest.NewRecorder()
	st := &Static{dir, ""}
	st.ServeExisting("/", CacheDisabled, nil).ServeHTTP(w, httpRequest)
	require.Equal(t, 404, w.Code)
}

func TestExecutingHandlerWhenNoFileFound(t *testing.T) {
	dir := "/path/to/non/existing/directory"
	dir2 := "/path/to/non/existing/directory2"
	httpRequest, _ := http.NewRequest("GET", "/file", nil)

	testCases := []struct {
		desc            string
		documentRoot    string
		altDocumentRoot string
	}{
		{"search in document root", dir, ""},
		{"search in alternate document root", dir, dir2},
	}

	for _, tc := range testCases {
		executed := false
		st := &Static{tc.documentRoot, tc.altDocumentRoot}
		st.ServeExisting("/", CacheDisabled, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			executed = (r == httpRequest)
		})).ServeHTTP(nil, httpRequest)
		if !executed {
			t.Error("The handler should get executed")
		}
	}
}

func TestServingTheActualFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "deploy")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dir2, err := ioutil.TempDir("", "deploy2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir2)

	testCases := []struct {
		desc            string
		documentRoot    string
		altDocumentRoot string
		filePath        string
	}{
		{"file in single document root", dir, "", dir},
		{"file in alternate document root", dir, dir2, dir2},
	}

	for _, tc := range testCases {
		httpRequest, _ := http.NewRequest("GET", "/file", nil)

		fileContent := "STATIC"
		ioutil.WriteFile(filepath.Join(tc.filePath, "file"), []byte(fileContent), 0600)

		w := httptest.NewRecorder()
		st := &Static{tc.documentRoot, tc.altDocumentRoot}
		st.ServeExisting("/", CacheDisabled, nil).ServeHTTP(w, httpRequest)
		require.Equal(t, 200, w.Code)
		if w.Body.String() != fileContent {
			t.Error("We should serve the file: ", w.Body.String())
		}
	}
}

func testServingThePregzippedFile(t *testing.T, enableGzip bool) {
	dir, err := ioutil.TempDir("", "deploy")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dir2, err := ioutil.TempDir("", "deploy2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir2)

	testCases := []struct {
		desc            string
		documentRoot    string
		altDocumentRoot string
		filePath        string
	}{
		{"file in document root", dir, "", dir},
		{"file in alternate document root", dir, dir2, dir2},
	}

	for _, tc := range testCases {
		httpRequest, _ := http.NewRequest("GET", "/file", nil)

		if enableGzip {
			httpRequest.Header.Set("Accept-Encoding", "gzip, deflate")
		}

		fileContent := "STATIC"

		var fileGzipContent bytes.Buffer
		fileGzip := gzip.NewWriter(&fileGzipContent)
		fileGzip.Write([]byte(fileContent))
		fileGzip.Close()

		ioutil.WriteFile(filepath.Join(tc.filePath, "file.gz"), fileGzipContent.Bytes(), 0600)
		ioutil.WriteFile(filepath.Join(tc.filePath, "file"), []byte(fileContent), 0600)

		w := httptest.NewRecorder()
		st := &Static{dir, ""}
		st.ServeExisting("/", CacheDisabled, nil).ServeHTTP(w, httpRequest)
		require.Equal(t, 200, w.Code)
		if enableGzip {
			testhelper.RequireResponseHeader(t, w, "Content-Encoding", "gzip")
			if !bytes.Equal(w.Body.Bytes(), fileGzipContent.Bytes()) {
				t.Error("We should serve the pregzipped file")
			}
		} else {
			require.Equal(t, 200, w.Code)
			testhelper.RequireResponseHeader(t, w, "Content-Encoding")
			if w.Body.String() != fileContent {
				t.Error("We should serve the file: ", w.Body.String())
			}
		}
	}
}

func TestServingThePregzippedFile(t *testing.T) {
	testServingThePregzippedFile(t, true)
}

func TestServingThePregzippedFileWithoutEncoding(t *testing.T) {
	testServingThePregzippedFile(t, false)
}
