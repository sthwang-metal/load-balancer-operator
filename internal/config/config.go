// Package config provides a struct to store the applications config
package config

import (
	"time"

	"go.infratographer.com/x/gidx"
	"go.infratographer.com/x/oauth2x"

	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/events"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/otelx"
)

// AppConfig contains the application configuration structure.
var AppConfig struct {
	Logging            loggingx.Config
	Events             events.Config
	Server             echox.Config
	Tracing            otelx.Config
	OIDC               OIDCClientConfig
	Metadata           MetadataConfig
	ReconcilerInterval time.Duration
}

// MetadataConfig stores the configuration for metadata
type MetadataConfig struct {
	StatusNamespaceID gidx.PrefixedID `mapstructure:"status-namespace-id"`
	Endpoint          string
	Source            string
}

// OIDCClientConfig stores the configuration for an OIDC client
type OIDCClientConfig struct {
	Client oauth2x.Config
}
