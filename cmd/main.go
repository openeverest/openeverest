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
// Package main is the entry point of the service.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openeverest/openeverest/v2/cmd/config"
	"github.com/openeverest/openeverest/v2/internal/server"
	"github.com/openeverest/openeverest/v2/pkg/logger"
)

const (
	contextTimeout = 10 * time.Second
)

func main() {
	logger := logger.MustInitLogger(true, "everest")
	defer logger.Sync() //nolint:errcheck
	l := logger.Sugar()

	// This is required because controller-runtime requires a logger
	// to be set within 30 seconds of the program initialization.
	log := zapr.NewLogger(logger)
	ctrlruntimelog.SetLogger(log)

	c, err := config.ParseConfig()
	if err != nil {
		l.Fatalf("Failed parsing config: %+v", err)
	}
	if !c.Verbose {
		logger = logger.WithOptions(zap.IncreaseLevel(zap.InfoLevel))
		l = logger.Sugar()
	}
	l.Debug("Debug logging enabled")

	tCtx, tCancel := context.WithCancel(context.Background())

	server, err := server.NewEverestServer(tCtx, c, l)
	if err != nil {
		l.Fatalf("Error creating Everest Server\n: %s", err)
	}

	go func() {
		err := server.Start(tCtx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Fatal(err)
		}
	}()

	if !c.DisableTelemetry {
		// To prevent leaking test data to prod,
		// the prod TelemetryURL is set for the release builds during the build time.
		// The dev TelemetryURL is set only when running `make run-debug`.
		if c.TelemetryURL != "" {
			l.Info("Telemetry is running")
			go server.RunTelemetryJob(tCtx, c)
		} else {
			l.Info("Telemetry is not running, the TELEMETRY_URL is not set")
		}
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	tCancel()
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		l.Error(errors.Join(err, errors.New("could not shut down Everest")))
	} else {
		l.Info("Everest shut down")
	}

	l.Info("Exiting")
}
