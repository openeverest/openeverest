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

// Package wait is a resource-agnostic polling waiter shared by CLI commands
// that block until a resource reaches a terminal state (e.g. instance create
// --wait). Callers supply a PollFunc and a Condition.
package wait

import (
	"context"
	"errors"
	"time"
)

// Outcome is the classification a Condition assigns to one observation.
type Outcome int

const (
	// Pending means the resource is not in a terminal state yet; keep polling.
	Pending Outcome = iota
	// Succeeded means the resource reached the desired terminal state.
	Succeeded
	// Failed means the resource reached a terminal failure state.
	Failed
)

// DefaultInterval applies when Options.Interval is not positive.
const DefaultInterval = 2 * time.Second

// ErrTimeout is returned when the timeout elapses first. Callers map it to a
// distinct exit code so scripts can tell a timeout apart from a real failure.
var ErrTimeout = errors.New("timed out waiting for condition")

// FailedError carries the reason a resource reached a terminal failure state.
type FailedError struct {
	Message string
}

func (e *FailedError) Error() string {
	if e.Message == "" {
		return "resource entered a failed state"
	}
	return e.Message
}

// Condition classifies a fetched resource. Its message is surfaced via
// Options.OnUpdate when Pending, or becomes the FailedError text when Failed.
type Condition[T any] func(T) (Outcome, string)

// PollFunc fetches the current resource state. A non-nil error is terminal and
// returned unchanged, so callers can stop early instead of waiting out the timeout.
type PollFunc[T any] func(ctx context.Context) (T, error)

// Options tunes the waiter.
type Options struct {
	// Interval between polls. Internal knob (no CLI flag); the zero value means
	// unset and falls back to DefaultInterval, so a bad value can't be typed.
	Interval time.Duration
	// Timeout bounds the total wait and must be positive — the command layer
	// rejects zero/negative.
	Timeout time.Duration
	// OnUpdate, if set, receives the Condition message on each Pending poll.
	// Consecutive duplicates are suppressed so output advances only on change.
	OnUpdate func(message string)
}

// Until polls until cond returns Succeeded (nil), Failed (*FailedError), poll
// errors, the timeout elapses (ErrTimeout), or ctx is cancelled. It polls once
// immediately, then on each tick.
func Until[T any](ctx context.Context, poll PollFunc[T], cond Condition[T], opts Options) error {
	interval := opts.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}

	// Own timeout context so deadlineErr can tell a timeout from a Ctrl-C.
	tctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	var lastMsg string
	first := true
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if !first {
			select {
			case <-tctx.Done():
				return deadlineErr(ctx, tctx)
			case <-ticker.C:
			}
		}
		first = false

		inst, err := poll(tctx)
		if err != nil {
			// Report a cancellation/deadline as such, not a generic fetch failure.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				if de := deadlineErr(ctx, tctx); de != nil {
					return de
				}
			}
			return err
		}

		outcome, msg := cond(inst)
		switch outcome {
		case Succeeded:
			return nil
		case Failed:
			return &FailedError{Message: msg}
		case Pending:
			if opts.OnUpdate != nil && msg != lastMsg {
				lastMsg = msg
				opts.OnUpdate(msg)
			}
		}
	}
}

// deadlineErr returns context.Canceled if the parent was cancelled, ErrTimeout
// if the deadline elapsed, or nil while tctx is still live.
func deadlineErr(parent, tctx context.Context) error {
	select {
	case <-tctx.Done():
		if parent.Err() != nil {
			return parent.Err()
		}
		return ErrTimeout
	default:
		return nil
	}
}
