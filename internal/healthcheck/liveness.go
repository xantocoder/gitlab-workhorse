package healthcheck

import (
	"net/http"
)

// This uses the definition of Kubernetes for liveness probes.
//
// If the endpoint is unresponsive perhaps the application is deadlocked
// due to a multi-threading defect.
//
// Reference: https://blog.colinbreck.com/kubernetes-liveness-and-readiness-probes-how-to-avoid-shooting-yourself-in-the-foot/

func Liveness() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := createStatus(true, nil)
		status.Write(w)
	})
}
