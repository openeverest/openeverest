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

package telemetry

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/internal/inventory"
)

// Runner sends the report periodically. It never fails its caller: errors are
// logged and the next attempt comes at the next interval.
type Runner struct {
	Interval     time.Duration
	InitialDelay time.Duration
	Sources      []SummarySource
	// Installation returns the identity of the installation; it is called
	// for every report.
	Installation func(ctx context.Context) (Installation, error)
	Transport    Transport
	Log          *zap.SugaredLogger
	Now          func() time.Time
}

// Run sends a report after InitialDelay and then every Interval, until ctx
// is done.
func (r *Runner) Run(ctx context.Context) {
	timer := time.NewTimer(r.InitialDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := r.SendOnce(ctx); err != nil {
				r.Log.Warnw("Failed to send the telemetry report", "error", err)
			}
			timer.Reset(r.Interval)
		}
	}
}

// SendOnce builds one report and sends it.
func (r *Runner) SendOnce(ctx context.Context) error {
	var summaries []inventory.Summary
	for _, src := range r.Sources {
		got, err := src.Summaries(ctx)
		if err != nil {
			// One unreachable source must not hide the others.
			r.Log.Warnw("Failed to collect the inventory summaries of a telemetry source", "error", err)
			continue
		}
		summaries = append(summaries, got...)
	}
	if len(summaries) == 0 {
		return errors.New("no source returned an inventory summary")
	}
	inst, err := r.Installation(ctx)
	if err != nil {
		return err
	}
	fleet := inventory.Merge(summaries, r.Now(), StaleAfter)
	return r.Transport.Send(ctx, NewReport(inst, fleet))
}
