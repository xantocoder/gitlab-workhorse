package sentry

import (
	"fmt"
	"os"
)

const dsnEnvKey = "SENTRY_DSN"

// The configuration for Wrap
type initializationConfig struct {
	dsn     string
	release string
}

// InitializationOption will configure a sentry handler
type InitializationOption func(*initializationConfig)

func applyInitializationOptions(opts []InitializationOption) initializationConfig {
	config := initializationConfig{
		dsn: os.Getenv("SENTRY_DSN"),
	}

	for _, v := range opts {
		v(&config)
	}

	return config
}

// WithRelease sets the release name.
func WithRelease(release string) InitializationOption {
	return func(config *initializationConfig) {
		config.release = release
	}
}

// WithRelease allows the DSN to be overridden. By default
// this will be retrieved from the SENTRY_DSN environment variable.
func WithDSN(dsn string) InitializationOption {
	return func(config *initializationConfig) {
		config.dsn = dsn
	}
}

// WithEnvScopedDSN sets DSN to the content of the "<appName>_SENTRY_DSN" environment variable.
func WithEnvScopedDSN(appName string) InitializationOption {
	return func(config *initializationConfig) {
		config.dsn = os.Getenv(fmt.Sprintf("%s_%s", appName, dsnEnvKey))
	}
}
