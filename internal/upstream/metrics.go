package upstream

import (
	"net/http"

	"gitlab.com/gitlab-org/labkit/metrics"
)

const (
	namespace = "gitlab_workhorse"
)

var metricsHandlerFactory = metrics.NewHandlerFactory(
	metrics.WithLabels("route"),
	metrics.WithNamespace(namespace),
)

// instrumentRoute adds the standard labkit metrics routes middleware to the handler chain
func instrumentRoute(next http.Handler, method string, regexpStr string) http.Handler {
	return metricsHandlerFactory(next, metrics.WithLabelValues(map[string]string{"route": regexpStr}))
}
