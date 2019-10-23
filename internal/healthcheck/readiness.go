package healthcheck

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/config"
)

// This uses the definition of Kubernetes for readiness probes.
//
// Readiness probes is used to decide when the application
// is available for accepting traffic. If a application is
// not ready, it is removed from service load balancers.
//
// Since the Workhorse is deeply interlocked with the instance
// of Unicorn/Puma running on the single node, we cannot
// consider Workhorse to be ready to accept connections
// if the underlying service is not functioning correctly.
//
// The endpoint will talk to downstream (deeply dependent) service,
// and present its and own status in readiness result.
//
// Reference: https://blog.colinbreck.com/kubernetes-liveness-and-readiness-probes-how-to-avoid-shooting-yourself-in-the-foot/

var httpTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 10 * time.Second,
	}).DialContext,
	MaxIdleConns:          2,
	IdleConnTimeout:       30 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 10 * time.Second,
	ResponseHeaderTimeout: 10 * time.Second,
}

var httpClient = &http.Client{
	Transport: httpTransport,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Timeout: 15 * time.Second,
}

func probeStatus(probeURL string) error {
	req, err := http.NewRequest("GET", probeURL, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	// drain resp.Body
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	// we consider everything that is not 2xx
	// as a failure
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("response status: %d, %v",
			resp.StatusCode, resp.Status)
	}

	return nil
}

func checkAllProbes(config *config.ReadinessConfig) (bool, map[string]error) {
	if config == nil {
		return true, nil
	}

	all := make(map[string]error)
	ok := true

	for _, probeURL := range config.ProbesURL {
		err := probeStatus(probeURL)
		all[probeURL] = err
		if err != nil {
			ok = false
		}
	}

	return ok, all
}

func Readiness(config *config.ReadinessConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, all := checkAllProbes(config)
		status := createStatus(ok, all)
		status.Write(w)
	})
}
