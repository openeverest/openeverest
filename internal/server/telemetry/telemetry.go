// everest
// Copyright (C) 2025 Percona LLC
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

// Package telemetry collects and ships anonymous usage data to Scarf Gateway.
//
// Layering: Probes (data sources) → Runner (orchestration) → Transport (wire).
// The Transport interface isolates Scarf-specific code to transport.go so the
// backend can be swapped without touching probe logic.
package telemetry

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/percona/everest/cmd/config"
	"github.com/percona/everest/pkg/kubernetes"
	"github.com/percona/everest/pkg/version"
)

const (
	// schemaVersion bumps on material, breaking changes to the property
	// vocabulary. Additive properties do not bump this.
	schemaVersion = "1"

	// eventName groups all heartbeat events on the Scarf side.
	eventName = "heartbeat"

	// initialDelay prevents flooding on rapid restart loops.
	initialDelay = 5 * time.Minute
)

// Runner periodically collects telemetry via Probes and ships it via Transport.
// It never fails the caller: every error is logged and execution continues.
type Runner struct {
	kc        kubernetes.KubernetesConnector
	l         *zap.SugaredLogger
	transport Transport
	interval  time.Duration
	probes    []Probe
}

// NewRunner constructs a Runner. interval is the cadence after the initial
// delay; probes defaults to the v1 set when nil.
func NewRunner(
	kc kubernetes.KubernetesConnector,
	l *zap.SugaredLogger,
	transport Transport,
	interval time.Duration,
	probes []Probe,
) *Runner {
	if probes == nil {
		probes = defaultProbes()
	}
	return &Runner{
		kc:        kc,
		l:         l,
		transport: transport,
		interval:  interval,
		probes:    probes,
	}
}

// Run blocks until ctx is canceled. It performs an initial collection after
// initialDelay and then once per interval.
func (r *Runner) Run(ctx context.Context) {
	r.l.Debug("telemetry runner: starting")

	timer := time.NewTimer(initialDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			r.l.Debug("telemetry runner: context canceled, stopping")
			return
		case <-timer.C:
			r.collectAndSend(ctx)
			timer.Reset(r.interval)
		}
	}
}

// collectAndSend gathers properties from every probe and ships a single event.
// Per-probe errors are logged but do not abort the overall send: we always
// publish whatever was collected successfully plus the always-on properties.
func (r *Runner) collectAndSend(ctx context.Context) {
	props := r.alwaysOnProperties(ctx)

	for _, p := range r.probes {
		got, err := p.Collect(ctx, r.kc)
		if err != nil {
			r.l.Errorw("telemetry probe failed", "probe", p.Name(), "error", err)
			continue
		}
		for k, v := range got {
			props[k] = v
		}
	}

	if err := r.transport.Send(ctx, props); err != nil {
		// Includes the env-opt-out case from the Scarf SDK; log at Debug
		// rather than Error so opt-out is silent.
		r.l.Debugw("telemetry send did not complete", "error", err)
	}
}

// alwaysOnProperties returns the small set of properties emitted on every
// event, even if every probe fails. instance_id is best-effort.
func (r *Runner) alwaysOnProperties(ctx context.Context) map[string]string {
	props := map[string]string{
		"event":           eventName,
		"schema_version":  schemaVersion,
		"everest_version": version.Version,
		"build_channel":   config.BuildChannel,
	}
	if r.kc != nil {
		if id, err := r.kc.GetEverestID(ctx); err == nil {
			props["instance_id"] = id
		}
	}
	return props
}
