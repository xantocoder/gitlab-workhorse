package sentry

import (
	"net/http"

	"github.com/getsentry/raven-go"
)

func Wrap(h http.Handler, opts ...InitializationOption) http.Handler {
	config := applyInitializationOptions(opts)

	raven.SetDSN(config.dsn) // sentryDSN may be empty

	if config.dsn == "" {
		return h
	}

	raven.DefaultClient.SetRelease(config.release)

	return http.HandlerFunc(raven.RecoveryHandler(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					cleanHeadersForRaven(r)
					panic(p)
				}
			}()

			h.ServeHTTP(w, r)
		}))
}
