package docker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/proxy"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/testhelper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/upstream/roundtripper"
)

func newProxy(url string) *proxy.Proxy {
	parsedURL := helper.URLMustParse(url)
	return proxy.NewProxy(parsedURL, "123", roundtripper.NewTestBackendRoundTripper(parsedURL))
}

func TestDockerClientRewriteRule(t *testing.T) {
	ts := testhelper.TestServerWithHandler(regexp.MustCompile(`.`), func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("RESPONSE"))

		if err != nil {
			t.Fatal(err)
		}
	})
	defer ts.Close()

	response := httptest.NewRecorder()
	body := strings.NewReader("my-request-body")
	request := httptest.NewRequest("GET", "/v2/my/awesome/request", body)

	rewriter := Rewriter(newProxy(ts.URL))

	rewriter.ServeHTTP(response, request)
	fmt.Println(request)
}
