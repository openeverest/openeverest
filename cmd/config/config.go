// everest
// Copyright (C) 2023 Percona LLC
// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package config ...
package config

import (
	"crypto/aes"
	"path/filepath"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	// AES256BitKeySize is the size (bytes) of a 256-bit key.
	AES256BitKeySize = 2 * aes.BlockSize
)

//nolint:gochecknoglobals
var (
	// TelemetryURL Everest telemetry endpoint. The variable is set for the release builds via ldflags
	// to have the correct default telemetry url.
	TelemetryURL string
	// TelemetryInterval Everest telemetry sending frequency. The variable is set for the release builds via ldflags
	// to have the correct default telemetry interval.
	TelemetryInterval string
)

// EverestConfig stores the configuration for the application.
type EverestConfig struct {
	DSN string `default:"postgres://admin:pwd@127.0.0.1:5432/postgres?sslmode=disable" envconfig:"DSN"`
	// DEPRECATED: Use ListenPort instead.
	HTTPPort   int  `envconfig:"HTTP_PORT"`
	ListenPort int  `default:"8080" envconfig:"PORT"`
	Verbose    bool `default:"false" envconfig:"VERBOSE"`
	// TelemetryURL Everest telemetry endpoint.
	TelemetryURL string `envconfig:"TELEMETRY_URL"`
	// TelemetryInterval Everest telemetry sending frequency.
	TelemetryInterval string `envconfig:"TELEMETRY_INTERVAL"`
	// DisableTelemetry disable Everest and the upstream operators telemetry
	DisableTelemetry bool `default:"false" envconfig:"DISABLE_TELEMETRY"`
	// APIRequestsRateLimit allowed amount of API requests per second
	APIRequestsRateLimit int `default:"100" envconfig:"API_REQUESTS_RATE_LIMIT"`
	// LoginRateLimit is the maximum number of login (password grant) requests per second per IP.
	LoginRateLimit int `default:"5" envconfig:"LOGIN_RATE_LIMIT"`
	// AccessTokenTTL is the lifetime of access JWTs.
	AccessTokenTTL time.Duration `default:"15m" envconfig:"ACCESS_TOKEN_TTL"`
	// RefreshTokenTTL is the lifetime of refresh tokens.
	// The window is sliding: every refresh token rotation grants a fresh TTL.
	RefreshTokenTTL time.Duration `default:"720h" envconfig:"REFRESH_TOKEN_TTL"`
	// VersionServiceURL contains the URL of the version service.
	VersionServiceURL string `default:"https://check.percona.com" envconfig:"VERSION_SERVICE_URL"`
	// TLSCertsPath contains the path to the directory with the TLS certificates.
	// Setting this will enable HTTPS on ListenPort.
	TLSCertsPath string `envconfig:"TLS_CERTS_PATH"`
	// Namespace is the namespace where OpenEverest is installed.
	// Must be provided via the NAMESPACE env var (set by the Helm chart).
	Namespace string `envconfig:"NAMESPACE" required:"true"`
}

// ParseConfig parses env vars and fills EverestConfig.
func ParseConfig() (*EverestConfig, error) {
	c := &EverestConfig{}
	err := envconfig.Process("", c)
	if err != nil {
		return nil, err
	}

	if c.TelemetryURL == "" {
		c.TelemetryURL = TelemetryURL
	}
	if c.TelemetryInterval == "" {
		c.TelemetryInterval = TelemetryInterval
	}

	if c.TLSCertsPath != "" {
		c.TLSCertsPath = filepath.Clean(c.TLSCertsPath)
	}

	return c, nil
}
