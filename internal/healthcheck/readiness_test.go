package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/config"
)

func TestReadinessProbe(t *testing.T) {
	handler := Readiness(nil)

	require.HTTPSuccess(t, handler.ServeHTTP, "GET", "/readiness", nil)
	require.HTTPBodyContains(t, handler.ServeHTTP, "GET", "/readiness", nil, `"status":"ok"`)
}

func TestReadinessWithInvalidProbe(t *testing.T) {
	config := &config.ReadinessConfig{
		ProbesURL: []string{"invalid://probe"},
	}
	handler := Readiness(config)

	require.HTTPError(t, handler.ServeHTTP, "GET", "/readiness", nil)
	require.HTTPBodyContains(t, handler.ServeHTTP, "GET", "/readiness", nil, `"status":"fail"`)
}

func TestReadinessWithNotRespondingEndpoint(t *testing.T) {
	config := &config.ReadinessConfig{
		ProbesURL: []string{"invalid://probe"},
	}
	handler := Readiness(config)

	require.HTTPError(t, handler.ServeHTTP, "GET", "/readiness", nil)
	require.HTTPBodyContains(t, handler.ServeHTTP, "GET", "/readiness", nil, `"status":"fail"`)
}

func TestReadinessProbeWithValidEndpoint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	config := &config.ReadinessConfig{
		ProbesURL: []string{ts.URL},
	}
	handler := Readiness(config)

	require.HTTPSuccess(t, handler.ServeHTTP, "GET", "/readiness", nil)
	require.HTTPBodyContains(t, handler.ServeHTTP, "GET", "/readiness", nil, `"status":"ok"`)
}

func TestReadinessProbeWithFailingEndpoint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	config := &config.ReadinessConfig{
		ProbesURL: []string{ts.URL},
	}
	handler := Readiness(config)

	require.HTTPError(t, handler.ServeHTTP, "GET", "/readiness", nil)
	require.HTTPBodyContains(t, handler.ServeHTTP, "GET", "/readiness", nil, `"status":"fail"`)
}
