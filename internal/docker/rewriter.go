package docker

import (
	"net/http"
	"strings"
)

func Rewriter(proxy http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.URL.Path = rewrite(req.URL.Path)

		proxy.ServeHTTP(w, req)
	})
}

func rewrite(path string) string {
	return strings.Replace(path, "v2", "my/path", 1)
}
