package serviceproxy

import (
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
)

func transformRedirection(resp *http.Response, w http.ResponseWriter, origHost string) {
	if resp.StatusCode < http.StatusMultipleChoices || resp.StatusCode > http.StatusBadRequest {
		return
	}

	location := w.Header().Get("Location")
	if location == "" {
		return
	}

	u, err := url.Parse(location)
	if err != nil || u.Host != resp.Request.URL.Host {
		return
	}

	// If we're redirecting to the same host
	// we need to change it to the original custom domain
	// and use the suitable scheme
	u.Host = origHost
	u.Scheme = protocolFor(resp.Request)
	w.Header().Set("Location", u.String())
}

func protocolFor(r *http.Request) string {
	if r.TLS == nil {
		return "http"
	}

	return "https"
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func isSubdomain(host string, proxyDomain string) bool {
	return helper.MatchDomain(host, "."+proxyDomain)
}
