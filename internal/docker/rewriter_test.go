package docker

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockProxy struct {
	mock.Mock
}

func (proxy *mockProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proxy.Called(w, r)
}

func TestDockerClientRewriteRule(t *testing.T) {
	response := httptest.NewRecorder()
	body := strings.NewReader("my-request-body")
	request := httptest.NewRequest("GET", "/v2/my/awesome/request", body)
	proxy := new(mockProxy)
	proxy.On("ServeHTTP", mock.Anything, mock.Anything).Once()

	rewriter := Rewriter(proxy)

	rewriter.ServeHTTP(response, request)
	assert.Contains(t, request.URL.Path, "my/path")
}
