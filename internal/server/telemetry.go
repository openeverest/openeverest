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

package server

import (
	"context"
	"errors"
	"time"

	"github.com/percona/everest/cmd/config"
	"github.com/percona/everest/internal/server/telemetry"
)

// RunTelemetryJob runs the background telemetry collector. It delegates to
// the telemetry package and never returns an error: launch as a goroutine
// and let it tear down when ctx is canceled.
func (e *EverestServer) RunTelemetryJob(ctx context.Context, c *config.EverestConfig) {
	interval, err := time.ParseDuration(c.TelemetryInterval)
	if err != nil {
		e.l.Error(errors.Join(err, errors.New("could not parse telemetry interval; telemetry disabled")))
		return
	}

	transport := telemetry.NewScarfTransport(c.TelemetryURL)
	runner := telemetry.NewRunner(e.kubeConnector, e.l, transport, interval, nil)
	runner.Run(ctx)
}
