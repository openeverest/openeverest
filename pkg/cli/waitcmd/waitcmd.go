// everest
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

// Package waitcmd holds the command-layer plumbing shared by every
// `<resource> create --wait` command (flag validation and exit-code mapping),
// so it lives in one place instead of drifting between per-resource copies.
package waitcmd

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// Exit codes for any `<resource> create --wait` command, chosen so CI scripts
// can tell the failure modes apart without scraping stderr:
//
//	0   success
//	1   the resource reached a Failed state (or any other error)
//	124 the --wait timeout elapsed (matches timeout(1))
//	130 the wait was cancelled with Ctrl-C (128 + SIGINT)
const (
	ExitFailed   = 1
	ExitTimeout  = 124
	ExitCanceled = 130
)

// ValidateWaitFlags rejects --timeout without --wait and non-positive timeouts.
func ValidateWaitFlags(waitSet, timeoutChanged bool, timeout time.Duration) error {
	if timeoutChanged && !waitSet {
		return errors.New("--timeout is only valid together with --wait")
	}
	if waitSet && timeout <= 0 {
		return errors.New("--timeout must be a positive duration (use e.g. --timeout 10m)")
	}
	return nil
}

// ExitCode maps a create/wait result to an exit code (see the Exit*
// constants) and prints the matching message.
func ExitCode(err error, l *zap.SugaredLogger, pretty bool, cancelMsg, timeoutMsg string) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, context.Canceled):
		output.PrintWarn(cancelMsg, l, pretty)
		return ExitCanceled
	case errors.Is(err, wait.ErrTimeout):
		output.PrintError(errors.New(timeoutMsg), l, pretty)
		return ExitTimeout
	default:
		// *wait.FailedError and everything else.
		output.PrintError(err, l, pretty)
		return ExitFailed
	}
}
