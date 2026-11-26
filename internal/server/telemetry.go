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

package server

import (
	"context"
	"errors"
	"time"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/cmd/config"
	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/internal/telemetry"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/version"
)

const telemetryDocsURL = "https://openeverest.io/docs/reference/telemetry"

// StartTelemetry starts sending the anonymous usage report in the background,
// or logs why it doesn't. Either way it logs one line saying so.
func (e *EverestServer) StartTelemetry(ctx context.Context) {
	settings, err := telemetry.Resolve(telemetry.Config{
		Disabled:   e.config.DisableTelemetry,
		DoNotTrack: e.config.DoNotTrack,
		URL:        e.config.TelemetryURL,
		Interval:   e.config.TelemetryInterval,
		DryRun:     e.config.TelemetryDryRun,
	})
	if errors.Is(err, telemetry.ErrDisabled) {
		e.l.Infow("Telemetry is off", "reason", err.Error())
		return
	}
	if err != nil {
		e.l.Errorw("Telemetry is off", "error", err)
		return
	}

	// A direct client: the report is built once a day, which doesn't justify
	// caching every kind the inventory covers in the API server.
	reader, err := ctrlclient.New(e.kubeConnector.Config(), ctrlclient.Options{Scheme: kubernetes.CreateScheme()})
	if err != nil {
		e.l.Errorw("Telemetry is off", "error", err)
		return
	}

	var transport telemetry.Transport = telemetry.NewHTTPTransport(settings.URL, "openeverest/"+version.Version)
	destination := settings.URL
	if settings.DryRun {
		transport = telemetry.LogTransport{Log: e.l}
		destination = "log (dry run)"
	}
	e.l.Infow("Telemetry is on: an anonymous usage report is sent periodically",
		"destination", destination,
		"interval", settings.Interval.String(),
		"turnOff", "set the Helm value telemetry.enabled=false",
		"details", telemetryDocsURL,
	)

	runner := &telemetry.Runner{
		Interval:     settings.Interval,
		InitialDelay: telemetry.InitialDelay,
		Sources:      []telemetry.SummarySource{telemetry.LocalSource{Reader: reader}},
		Installation: e.telemetryInstallation,
		Transport:    transport,
		Log:          e.l,
		Now:          time.Now,
	}
	go runner.Run(ctx)
}

func (e *EverestServer) telemetryInstallation(ctx context.Context) (telemetry.Installation, error) {
	id, err := e.kubeConnector.GetEverestID(ctx)
	if err != nil {
		return telemetry.Installation{}, err
	}
	return telemetry.Installation{
		ID:           id,
		BuildChannel: config.BuildChannel,
		Version:      version.Version,
		Contract:     contract.Version,
	}, nil
}
