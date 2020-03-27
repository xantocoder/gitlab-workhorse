package contentprocessor

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/headers"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/testhelper"

	"github.com/stretchr/testify/require"
)

func TestFailSetContentTypeAndDisposition(t *testing.T) {
	testCaseBody := "Hello world!"

	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, testCaseBody)
		require.NoError(t, err)
	})

	resp := makeRequest(t, h, testCaseBody, "", "")

	require.Equal(t, "", resp.Header.Get(headers.ContentDispositionHeader))
	require.Equal(t, "", resp.Header.Get(headers.ContentTypeHeader))
}

func TestSuccessSetContentTypeAndDispositionFeatureEnabled(t *testing.T) {
	testCaseBody := "Hello world!"

	resp := makeRequest(t, nil, testCaseBody, "", "")

	require.Equal(t, "inline", resp.Header.Get(headers.ContentDispositionHeader))
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get(headers.ContentTypeHeader))
}

func TestSetProperContentDisposition(t *testing.T) {
	testCases := []struct {
		desc                       string
		contentType                string
		contentDisposition         string
		expectedContentDisposition string
		body                       string
	}{
		{
			desc:                       "Text can be inline",
			contentType:                "text/plain",
			contentDisposition:         "inline",
			expectedContentDisposition: "inline",
			body:                       "Hello world!",
		},
		{
			desc:                       "Text can be attachment",
			contentType:                "text/plain",
			contentDisposition:         "attachment",
			expectedContentDisposition: "attachment",
			body:                       "Hello world!",
		},
		{
			desc:                       "Images can be inline",
			contentType:                "image/png",
			contentDisposition:         "inline",
			expectedContentDisposition: "inline",
			body:                       testhelper.LoadFile(t, "testdata/image.png"),
		},
		{
			desc:                       "Images can be attachment",
			contentType:                "image/png",
			contentDisposition:         "attachment",
			expectedContentDisposition: "attachment",
			body:                       testhelper.LoadFile(t, "testdata/image.png"),
		},
		{
			desc:                       "SVG can be attachment",
			contentType:                "image/svg+xml",
			contentDisposition:         "attachment",
			expectedContentDisposition: "attachment",
			body:                       testhelper.LoadFile(t, "testdata/image.svg"),
		},
		{
			desc:                       "SVG can not be inline",
			contentType:                "image/svg+xml",
			contentDisposition:         "inline",
			expectedContentDisposition: "attachment",
			body:                       testhelper.LoadFile(t, "testdata/image.svg"),
		},
		{
			desc:                       "PDF can be inline",
			contentType:                "application/pdf",
			contentDisposition:         "inline",
			expectedContentDisposition: "inline",
			body:                       testhelper.LoadFile(t, "testdata/file.pdf"),
		},
		{
			desc:                       "PDF can be attachment",
			contentType:                "application/pdf",
			contentDisposition:         "attachment",
			expectedContentDisposition: "attachment",
			body:                       testhelper.LoadFile(t, "testdata/file.pdf"),
		},
		{
			desc:                       "PDF file with non-ASCII characters in filename",
			contentType:                "application/pdf",
			contentDisposition:         `attachment; filename="file-ä.pdf"; filename*=UTF-8''file-%c3.pdf`,
			expectedContentDisposition: `attachment; filename="file-ä.pdf"; filename*=UTF-8''file-%c3.pdf`,
			body:                       testhelper.LoadFile(t, "testdata/file-ä.pdf"),
		},
		{
			desc:                       "Video type can be inline",
			contentType:                "video/mp4",
			contentDisposition:         "inline",
			expectedContentDisposition: "inline",
			body:                       testhelper.LoadFile(t, "testdata/video.mp4"),
		},
		{
			desc:                       "Video type can be attachment",
			contentType:                "video/mp4",
			contentDisposition:         "attachment",
			expectedContentDisposition: "attachment",
			body:                       testhelper.LoadFile(t, "testdata/video.mp4"),
		},
		{
			desc:                       "Audio can be inline",
			contentType:                "audio/mpeg",
			contentDisposition:         "inline",
			expectedContentDisposition: "inline",
			body:                       testhelper.LoadFile(t, "testdata/audio.mp3"),
		},
		{
			desc:                       "Audio can be attachment",
			contentType:                "audio/mpeg",
			contentDisposition:         "attachment",
			expectedContentDisposition: "attachment",
			body:                       testhelper.LoadFile(t, "testdata/audio.mp3"),
		},
		{
			desc:                       "Application executable can be attachment",
			contentType:                "application/x-shockwave-flash",
			contentDisposition:         "attachment",
			expectedContentDisposition: "attachment",
			body:                       testhelper.LoadFile(t, "testdata/file.swf"),
		},
		{
			desc:                       "Application executable can not be inline",
			contentType:                "application/x-shockwave-flash",
			contentDisposition:         "inline",
			expectedContentDisposition: "attachment",
			body:                       testhelper.LoadFile(t, "testdata/file.swf"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			resp := makeRequest(t, nil, tc.body, tc.contentDisposition, tc.contentType)

			require.Equal(t, tc.expectedContentDisposition, resp.Header.Get(headers.ContentDispositionHeader))
		})
	}
}

func TestSetProperContentTypeWithExistingContentType(t *testing.T) {
	testCases := []struct {
		desc                string
		existingContentType string
		expectedContentType string
		body                string
	}{
		{
			desc:                "Text type",
			existingContentType: "text/plain; charset=utf-8",
			expectedContentType: "text/plain; charset=utf-8",
			body:                "Hello world!",
		},
		{
			desc:                "HTML type",
			existingContentType: "text/html",
			expectedContentType: "text/plain; charset=utf-8",
			body:                "<html><body>Hello world!</body></html>",
		},
		{
			desc:                "Javascript type",
			existingContentType: "application/javascript",
			expectedContentType: "text/plain; charset=utf-8",
			body:                "<script>alert(\"foo\")</script>",
		},
		{
			desc:                "Image type",
			existingContentType: "image/png",
			expectedContentType: "image/png",
			body:                testhelper.LoadFile(t, "testdata/image.png"),
		},
		{
			desc:                "SVG type",
			existingContentType: "image/svg+xml",
			expectedContentType: "image/svg+xml",
			body:                testhelper.LoadFile(t, "testdata/image.svg"),
		},
		{
			desc:                "Partial SVG type",
			existingContentType: "image/svg+xml",
			expectedContentType: "image/svg+xml",
			body:                "<svg xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\" viewBox=\"0 0 330 82\"><title>SVG logo combined with the W3C logo, set horizontally</title><desc>The logo combines three entities displayed horizontall</desc><metadata>",
		},
		{
			desc:                "PDF type",
			existingContentType: "application/pdf",
			expectedContentType: "application/pdf",
			body:                testhelper.LoadFile(t, "testdata/file.pdf"),
		},
		{
			desc:                "Application executable type",
			existingContentType: "application/x-shockwave-flash",
			expectedContentType: "application/x-shockwave-flash",
			body:                testhelper.LoadFile(t, "testdata/file.swf"),
		},
		{
			desc:                "Video type",
			existingContentType: "video/mp4",
			expectedContentType: "video/mp4",
			body:                testhelper.LoadFile(t, "testdata/video.mp4"),
		},
		{
			desc:                "Audio type",
			existingContentType: "audio/mpeg",
			expectedContentType: "audio/mpeg",
			body:                testhelper.LoadFile(t, "testdata/audio.mp3"),
		},
		{
			desc:                "JSON type",
			existingContentType: "application/json",
			expectedContentType: "text/plain; charset=utf-8",
			body:                "{ \"glossary\": { \"title\": \"example glossary\", \"GlossDiv\": { \"title\": \"S\" } } }",
		},
		{
			desc:                "Forged file with png extension but SWF content",
			existingContentType: "image/png",
			expectedContentType: "application/octet-stream",
			body:                testhelper.LoadFile(t, "testdata/forgedfile.png"),
		},
		{
			desc:                "BMPR file",
			existingContentType: "application/vnd.balsamiq.bmpr",
			expectedContentType: "application/vnd.balsamiq.bmpr",
			body:                testhelper.LoadFile(t, "testdata/file.bmpr"),
		},
		{
			desc:                "STL file",
			existingContentType: "model/stl",
			expectedContentType: "application/octet-stream",
			body:                testhelper.LoadFile(t, "testdata/file.stl"),
		},
		{
			desc:                "RDoc file",
			existingContentType: "text/plain; charset=utf-8",
			expectedContentType: "text/plain; charset=utf-8",
			body:                testhelper.LoadFile(t, "testdata/file.rdoc"),
		},
		{
			desc:                "IPYNB file",
			existingContentType: "application/x-ipynb+json",
			expectedContentType: "text/plain; charset=utf-8",
			body:                testhelper.LoadFile(t, "testdata/file.ipynb"),
		},
		{
			desc:                "Sketch file",
			existingContentType: "application/zip",
			expectedContentType: "application/zip",
			body:                testhelper.LoadFile(t, "testdata/file.sketch"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			resp := makeRequest(t, nil, tc.body, "", tc.existingContentType)

			require.Equal(t, tc.expectedContentType, resp.Header.Get(headers.ContentTypeHeader))
		})
	}
}

func TestSetProperContentTypeWithoutExistingContentType(t *testing.T) {
	testCases := []struct {
		desc                string
		expectedContentType string
		body                string
	}{
		{
			desc:                "Text type",
			expectedContentType: "text/plain; charset=utf-8",
			body:                "Hello world!",
		},
		{
			desc:                "HTML type",
			expectedContentType: "text/plain; charset=utf-8",
			body:                "<html><body>Hello world!</body></html>",
		},
		{
			desc:                "Javascript type",
			expectedContentType: "text/plain; charset=utf-8",
			body:                "<script>alert(\"foo\")</script>",
		},
		{
			desc:                "Image type",
			expectedContentType: "image/png",
			body:                testhelper.LoadFile(t, "testdata/image.png"),
		},
		{
			desc:                "SVG type",
			expectedContentType: "image/svg+xml",
			body:                testhelper.LoadFile(t, "testdata/image.svg"),
		},
		{
			desc:                "Partial SVG type",
			expectedContentType: "image/svg+xml",
			body:                "<svg xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\" viewBox=\"0 0 330 82\"><title>SVG logo combined with the W3C logo, set horizontally</title><desc>The logo combines three entities displayed horizontall</desc><metadata>",
		},
		{
			desc:                "PDF type",
			expectedContentType: "application/pdf",
			body:                testhelper.LoadFile(t, "testdata/file.pdf"),
		},
		{
			desc:                "Application executable type",
			expectedContentType: "application/octet-stream",
			body:                testhelper.LoadFile(t, "testdata/file.swf"),
		},
		{
			desc:                "Video type",
			expectedContentType: "video/mp4",
			body:                testhelper.LoadFile(t, "testdata/video.mp4"),
		},
		{
			desc:                "Audio type",
			expectedContentType: "audio/mpeg",
			body:                testhelper.LoadFile(t, "testdata/audio.mp3"),
		},
		{
			desc:                "JSON type",
			expectedContentType: "text/plain; charset=utf-8",
			body:                "{ \"glossary\": { \"title\": \"example glossary\", \"GlossDiv\": { \"title\": \"S\" } } }",
		},
		{
			desc:                "Forged file with png extension but SWF content",
			expectedContentType: "application/octet-stream",
			body:                testhelper.LoadFile(t, "testdata/forgedfile.png"),
		},
		{
			desc:                "BMPR file",
			expectedContentType: "application/octet-stream",
			body:                testhelper.LoadFile(t, "testdata/file.bmpr"),
		},
		{
			desc:                "STL file",
			expectedContentType: "application/octet-stream",
			body:                testhelper.LoadFile(t, "testdata/file.stl"),
		},
		{
			desc:                "RDoc file",
			expectedContentType: "text/plain; charset=utf-8",
			body:                testhelper.LoadFile(t, "testdata/file.rdoc"),
		},
		{
			desc:                "IPYNB file",
			expectedContentType: "text/plain; charset=utf-8",
			body:                testhelper.LoadFile(t, "testdata/file.ipynb"),
		},
		{
			desc:                "Sketch file",
			expectedContentType: "application/zip",
			body:                testhelper.LoadFile(t, "testdata/file.sketch"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			resp := makeRequest(t, nil, tc.body, "", "")

			require.Equal(t, tc.expectedContentType, resp.Header.Get(headers.ContentTypeHeader))
		})
	}
}

func TestFailOverrideContentType(t *testing.T) {
	testCase := struct {
		contentType string
		body        string
	}{
		contentType: "text/plain; charset=utf-8",
		body:        "<html><body>Hello world!</body></html>",
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// We are pretending to be upstream or an inner layer of the ResponseWriter chain
		w.Header().Set(headers.GitlabWorkhorseDetectContentTypeHeader, "true")
		w.Header().Set(headers.ContentTypeHeader, "text/html; charset=utf-8")
		_, err := io.WriteString(w, testCase.body)
		require.NoError(t, err)
	})

	resp := makeRequest(t, h, testCase.body, "", "")

	require.Equal(t, testCase.contentType, resp.Header.Get(headers.ContentTypeHeader))
}

func TestSuccessOverrideContentDispositionFromInlineToAttachment(t *testing.T) {
	testCaseBody := "Hello world!"

	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// We are pretending to be upstream or an inner layer of the ResponseWriter chain
		w.Header().Set(headers.ContentDispositionHeader, "attachment")
		w.Header().Set(headers.GitlabWorkhorseDetectContentTypeHeader, "true")
		_, err := io.WriteString(w, testCaseBody)
		require.NoError(t, err)
	})

	resp := makeRequest(t, h, testCaseBody, "", "")

	require.Equal(t, "attachment", resp.Header.Get(headers.ContentDispositionHeader))
}

func TestFailOverrideContentDispositionFromAttachmentToInline(t *testing.T) {
	testCaseBody := testhelper.LoadFile(t, "testdata/image.svg")

	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// We are pretending to be upstream or an inner layer of the ResponseWriter chain
		w.Header().Set(headers.ContentDispositionHeader, "inline")
		w.Header().Set(headers.GitlabWorkhorseDetectContentTypeHeader, "true")
		_, err := io.WriteString(w, testCaseBody)
		require.NoError(t, err)
	})

	resp := makeRequest(t, h, testCaseBody, "", "")

	require.Equal(t, "attachment", resp.Header.Get(headers.ContentDispositionHeader))
}

func TestHeadersDelete(t *testing.T) {
	for _, code := range []int{200, 400} {
		recorder := httptest.NewRecorder()
		rw := &contentDisposition{rw: recorder}
		for _, name := range headers.ResponseHeaders {
			rw.Header().Set(name, "foobar")
		}

		rw.WriteHeader(code)

		for _, name := range headers.ResponseHeaders {
			if header := recorder.Header().Get(name); header != "" {
				t.Fatalf("HTTP %d response: expected header to be empty, found %q", code, name)
			}
		}
	}
}

func TestWriteHeadersCalledOnce(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &contentDisposition{rw: recorder}
	rw.WriteHeader(400)
	require.Equal(t, 400, rw.status)
	require.Equal(t, true, rw.sentStatus)

	rw.WriteHeader(200)
	require.Equal(t, 400, rw.status)
}

func makeRequest(t *testing.T, handler http.HandlerFunc, body string, disposition string, contentType string) *http.Response {
	if handler == nil {
		handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// We are pretending to be upstream
			w.Header().Set(headers.GitlabWorkhorseDetectContentTypeHeader, "true")
			w.Header().Set(headers.ContentDispositionHeader, disposition)
			w.Header().Set(headers.ContentTypeHeader, contentType)
			_, err := io.WriteString(w, body)
			require.NoError(t, err)
		})
	}
	req, _ := http.NewRequest("GET", "/", nil)

	rw := httptest.NewRecorder()
	SetContentHeaders(handler).ServeHTTP(rw, req)

	resp := rw.Result()
	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, body, string(respBody))

	return resp
}
