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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/openeverest/openeverest/v2/pkg/inuse/uninstall"
)

// uninstallCheckCommand is the subcommand that refuses uninstalling core
// while it still protects objects with an in-use finalizer. The chart's
// pre-delete hook runs it.
const uninstallCheckCommand = "uninstall-check"

// Exit codes of uninstallCheckCommand.
const (
	exitNothingHeld = 0
	exitHeld        = 1
	exitError       = 2
)

// uninstallCheckConfig is the parsed command line of uninstallCheckCommand.
type uninstallCheckConfig struct {
	timeout time.Duration
	zap     zap.Options
}

func parseUninstallCheckFlags(args []string, output io.Writer) (uninstallCheckConfig, error) {
	cfg := uninstallCheckConfig{zap: zap.Options{Development: true}}
	fs := flag.NewFlagSet(uninstallCheckCommand, flag.ContinueOnError)
	fs.SetOutput(output)
	fs.DurationVar(&cfg.timeout, "timeout", time.Minute, "How long the check may take.")
	cfg.zap.BindFlags(fs)
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if fs.NArg() > 0 {
		return cfg, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	return cfg, nil
}

// runUninstallCheck runs uninstallCheckCommand with args and returns the
// process's exit code: exitNothingHeld, exitHeld with the objects held on
// stderr, or exitError.
func runUninstallCheck(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseUninstallCheckFlags(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return exitNothingHeld
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: %v\n", uninstallCheckCommand, err)
		return exitError
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&cfg.zap), zap.WriteTo(stderr)))

	restConfig, err := ctrl.GetConfig()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: load the Kubernetes client configuration: %v\n", uninstallCheckCommand, err)
		return exitError
	}
	c, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: create the Kubernetes client: %v\n", uninstallCheckCommand, err)
		return exitError
	}
	ctx, cancel := context.WithTimeout(ctrl.SetupSignalHandler(), cfg.timeout)
	defer cancel()
	return uninstallCheck(ctx, c, stdout, stderr)
}

// uninstallCheck runs uninstall.Check and reports its outcome.
func uninstallCheck(ctx context.Context, r client.Reader, stdout, stderr io.Writer) int {
	err := uninstall.Check(ctx, r)
	if err == nil {
		_, _ = fmt.Fprintln(stdout, "OpenEverest protects no object any more; it can be uninstalled.")
		return exitNothingHeld
	}
	_, _ = fmt.Fprintln(stderr, err)
	if _, held := errors.AsType[*uninstall.HeldError](err); held {
		return exitHeld
	}
	return exitError
}
