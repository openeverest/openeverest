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

// Package telemetry sends the anonymous usage report of an OpenEverest
// installation.
//
// Only the main installation sends it. The report is the sum of the inventory
// summaries of every cluster the installation manages, each one obtained
// through a SummarySource: today the local cluster only, later whatever
// transport multicluster agents use. A Transport delivers the report: a plain
// JSON POST, so the collector behind the URL can change without a release, or
// the log, for a dry run.
package telemetry

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultInterval is how often the report is sent unless configured.
	DefaultInterval = 24 * time.Hour
	// MinInterval is the shortest interval accepted, so a misconfiguration
	// can't flood the collector or the API server.
	MinInterval = time.Hour
	// InitialDelay delays the first report after a start, so a crash loop
	// doesn't send one report per restart.
	InitialDelay = 5 * time.Minute
	// StaleAfter is how old a cluster's summary may be and still be added to
	// the report.
	StaleAfter = 48 * time.Hour
	// ReportTimeout bounds everything one report does: reading the clusters
	// and sending. A report that doesn't finish in time is dropped until the
	// next interval.
	ReportTimeout = 2 * time.Minute
)

// Config is how telemetry is configured.
type Config struct {
	// Disabled is DISABLE_TELEMETRY, set by the Helm value telemetry.enabled.
	Disabled bool
	// DoNotTrack is the cross-tool DO_NOT_TRACK variable.
	DoNotTrack string
	// URL is where the report is sent. Only release and RC builds have a
	// default; TELEMETRY_URL overrides it.
	URL string
	// Interval is how often the report is sent, empty for DefaultInterval.
	Interval string
	// DryRun logs the report instead of sending it.
	DryRun bool
}

// Settings is a Config telemetry runs with.
type Settings struct {
	URL      string
	Interval time.Duration
	DryRun   bool
}

// ErrDisabled wraps the reason telemetry doesn't run.
var ErrDisabled = errors.New("telemetry is disabled")

// Resolve decides whether telemetry runs. When it doesn't, the error says why
// and wraps ErrDisabled, or is a configuration error.
func Resolve(c Config) (Settings, error) {
	switch {
	case c.Disabled:
		return Settings{}, fmt.Errorf("%w: DISABLE_TELEMETRY is set (Helm value telemetry.enabled=false)", ErrDisabled)
	case doNotTrack(c.DoNotTrack):
		return Settings{}, fmt.Errorf("%w: DO_NOT_TRACK is set", ErrDisabled)
	case c.URL == "" && !c.DryRun:
		return Settings{}, fmt.Errorf("%w: this build has no telemetry endpoint and TELEMETRY_URL is not set", ErrDisabled)
	}
	interval := DefaultInterval
	if c.Interval != "" {
		var err error
		if interval, err = time.ParseDuration(c.Interval); err != nil {
			return Settings{}, fmt.Errorf("invalid TELEMETRY_INTERVAL %q: %w", c.Interval, err)
		}
		if interval < MinInterval {
			return Settings{}, fmt.Errorf("invalid TELEMETRY_INTERVAL %q: must be at least %s", c.Interval, MinInterval)
		}
	}
	return Settings{URL: c.URL, Interval: interval, DryRun: c.DryRun}, nil
}

// doNotTrack follows https://consoledonottrack.com: any value but empty,
// "0" and "false" opts out.
func doNotTrack(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	return true
}
