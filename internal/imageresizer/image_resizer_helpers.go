package imageresizer

import (
	"net/http"
	"time"
	"net"
	"strings"
	"io"
	"io/ioutil"
	"fmt"

	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/tracing"
)

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