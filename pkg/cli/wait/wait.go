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

// RetryableError marks a transient poll error: instead of stopping, the waiter
// reports it via Options.OnRetry and polls again on the next tick. Terminal
// poll errors are returned bare.
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

// Condition classifies a fetched resource. Its message is surfaced via
// Options.OnUpdate when Pending, or becomes the FailedError text when Failed.
type Condition[T any] func(T) (Outcome, string)

// PollFunc fetches the current resource state. A non-nil error is terminal and
// returned unchanged, so callers can stop early instead of waiting out the timeout.
type PollFunc[T any] func(ctx context.Context) (T, error)

// Options tunes the waiter.
type Options struct {
	// Interval between polls; the zero value falls back to DefaultInterval.
	Interval time.Duration
	// Timeout bounds the total wait. Zero or negative means no timeout — poll
	// until Succeeded/Failed, a terminal poll error, or ctx cancellation —
	// mirroring kubectl's --timeout=0 and Interval's zero-value behaviour. (The
	// instance-create CLI still requires a positive --timeout via its own flag
	// validation; this is only the package's self-defending default.)
	Timeout time.Duration
	// OnUpdate, if set, receives the Condition message on each Pending poll.
	// Consecutive duplicates are suppressed so output advances only on change.
	OnUpdate func(message string)
	// OnRetry, if set, is called with each transient (RetryableError) poll error
	// before the waiter polls again.
	OnRetry func(err error)
}

// Until polls until cond returns Succeeded (nil), Failed (*FailedError), poll
// errors, the timeout elapses (ErrTimeout), or ctx is cancelled. It polls once
// immediately, then on each tick.
func Until[T any](ctx context.Context, poll PollFunc[T], cond Condition[T], opts Options) error {
	interval := opts.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}

	// Own timeout context so deadlineErr can tell a timeout from a Ctrl-C. A
	// zero/negative Timeout means "no timeout": poll against the parent ctx.
	tctx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		tctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

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
			if done, rerr := classifyPollErr(ctx, tctx, err, opts.OnRetry); done {
				return rerr
			}
			continue
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

// classifyPollErr decides what to do with a poll error. It returns done=true
// with the error to surface for a terminal failure (including a cancellation or
// elapsed deadline), or done=false for a transient RetryableError — after
// reporting it via onRetry — so the caller polls again.
func classifyPollErr(parent, tctx context.Context, err error, onRetry func(error)) (bool, error) {
	// A cancellation/deadline surfacing through poll is reported as such.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		if de := deadlineErr(parent, tctx); de != nil {
			return true, de
		}
	}
	var re *RetryableError
	if errors.As(err, &re) {
		if onRetry != nil {
			onRetry(re.Err)
		}
		return false, nil
	}
	return true, err
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
