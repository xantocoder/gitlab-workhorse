package healthcheck

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLivenessProbe(t *testing.T) {
	handler := Liveness()

	require.HTTPSuccess(t, handler.ServeHTTP, "GET", "/liveness", nil)
	require.HTTPBodyContains(t, handler.ServeHTTP, "GET", "/liveness", nil, `"status":"ok"`)
}
