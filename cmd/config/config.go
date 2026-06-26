// everest
// Copyright (C) 2023 Percona LLC
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

	"github.com/kelseyhightower/envconfig"
)

const (
	// AES256BitKeySize is the size (bytes) of a 256-bit key.
	AES256BitKeySize = 2 * aes.BlockSize
)

//nolint:gochecknoglobals
var (
	// TelemetryURL is the default Everest telemetry endpoint. Points at the
	// Scarf Gateway Event Collection endpoint; can be overridden at build
	// time via ldflags or at runtime via the TELEMETRY_URL env var.
	TelemetryURL = "https://openeverest.gateway.scarf.sh/telemetry"
	// TelemetryInterval is the default telemetry sending frequency. Set via
	// ldflags for release builds.
	TelemetryInterval string
	// BuildChannel identifies the release channel of this binary
	// ("dev", "rc", or "release"). Set via ldflags for rc and release builds.
	BuildChannel = "dev"
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
	// CreateSessionRateLimit allowed amount of API requests per second to the /session method
	CreateSessionRateLimit int `default:"1" envconfig:"CREATE_SESSION_RATE_LIMIT"`
	// VersionServiceURL contains the URL of the version service.
	VersionServiceURL string `default:"https://check.percona.com" envconfig:"VERSION_SERVICE_URL"`
	// TLSCertsPath contains the path to the directory with the TLS certificates.
	// Setting this will enable HTTPS on ListenPort.
	TLSCertsPath string `envconfig:"TLS_CERTS_PATH"`
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
