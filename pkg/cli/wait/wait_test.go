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

package wait_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

// phaseCond mirrors the real instance condition: Ready succeeds, Failed fails,
// everything else pends.
func phaseCond(phase string) (wait.Outcome, string) {
	switch phase {
	case "Ready":
		return wait.Succeeded, "ready"
	case "Failed":
		return wait.Failed, "instance entered Failed phase"
	default:
		return wait.Pending, "phase: " + phase
	}
}

func TestUntil_SucceedsImmediately(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	poll := func(_ context.Context) (string, error) {
		calls.Add(1)
		return "Ready", nil
	}

	err := wait.Until(context.Background(), poll, phaseCond,
		wait.Options{Interval: time.Millisecond, Timeout: time.Second},
	)

	require.NoError(t, err)
	// Terminal on the very first observation — no ticker wait needed.
	assert.Equal(t, int32(1), calls.Load())
}

func TestUntil_TransitionsThenSucceeds(t *testing.T) {
	t.Parallel()

	phases := []string{"Pending", "Provisioning", "Initializing", "Ready"}
	var idx atomic.Int32
	poll := func(_ context.Context) (string, error) {
		i := idx.Add(1) - 1
		if int(i) >= len(phases) {
			return phases[len(phases)-1], nil
		}
		return phases[i], nil
	}

	var updates []string
	err := wait.Until(context.Background(), poll, phaseCond,
		wait.Options{
			Interval: time.Millisecond,
			Timeout:  2 * time.Second,
			OnUpdate: func(m string) { updates = append(updates, m) },
		},
	)

	require.NoError(t, err)
	// Three pending phases seen before Ready — each surfaced once, in order.
	assert.Equal(t, []string{"phase: Pending", "phase: Provisioning", "phase: Initializing"}, updates)
}

func TestUntil_FailedPhaseReturnsFailedError(t *testing.T) {
	t.Parallel()

	poll := func(_ context.Context) (string, error) { return "Failed", nil }

	err := wait.Until(context.Background(), poll, phaseCond,
		wait.Options{Interval: time.Millisecond, Timeout: time.Second},
	)

	var fe *wait.FailedError
	require.ErrorAs(t, err, &fe)
	assert.Equal(t, "instance entered Failed phase", fe.Message)
}

func TestUntil_TimeoutReturnsErrTimeout(t *testing.T) {
	t.Parallel()

	// Never terminal — always pending, so only the timeout can stop it.
	poll := func(_ context.Context) (string, error) { return "Provisioning", nil }

	err := wait.Until(context.Background(), poll, phaseCond,
		wait.Options{Interval: 5 * time.Millisecond, Timeout: 40 * time.Millisecond},
	)

	require.ErrorIs(t, err, wait.ErrTimeout)
}

func TestUntil_ParentCancelReturnsContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	poll := func(_ context.Context) (string, error) { return "Provisioning", nil }

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := wait.Until(ctx, poll, phaseCond,
		// Long timeout so the cancellation, not the deadline, is what stops us.
		wait.Options{Interval: 5 * time.Millisecond, Timeout: 10 * time.Second},
	)

	require.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, wait.ErrTimeout)
}

func TestUntil_PollErrorIsTerminal(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("instance \"my-mongo\" has been deleted")
	var calls atomic.Int32
	poll := func(_ context.Context) (string, error) {
		calls.Add(1)
		return "", sentinel
	}

	err := wait.Until(context.Background(), poll, phaseCond,
		wait.Options{Interval: time.Millisecond, Timeout: time.Second},
	)

	require.ErrorIs(t, err, sentinel)
	// Returned on the first poll — not retried.
	assert.Equal(t, int32(1), calls.Load())
}

func TestUntil_OnUpdateSuppressesDuplicates(t *testing.T) {
	t.Parallel()

	// Same pending phase repeatedly, then Ready — OnUpdate should fire once.
	var idx atomic.Int32
	poll := func(_ context.Context) (string, error) {
		if idx.Add(1) >= 4 {
			return "Ready", nil
		}
		return "Provisioning", nil
	}

	var updates atomic.Int32
	err := wait.Until(context.Background(), poll, phaseCond,
		wait.Options{
			Interval: time.Millisecond,
			Timeout:  time.Second,
			OnUpdate: func(_ string) { updates.Add(1) },
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int32(1), updates.Load())
}

func TestUntil_DefaultIntervalWhenUnset(t *testing.T) {
	t.Parallel()

	// Immediate success means the (large) default interval is never waited on,
	// so this stays fast while covering the zero-interval branch.
	poll := func(_ context.Context) (string, error) { return "Ready", nil }

	err := wait.Until(context.Background(), poll, phaseCond,
		wait.Options{Timeout: time.Second},
	)
	require.NoError(t, err)
}

func TestUntil_RetryableIsRetriedNotTerminal(t *testing.T) {
	t.Parallel()

	// First two polls fail transiently, then Ready — the wait should keep going
	// and report each transient error via OnRetry rather than stopping.
	var calls atomic.Int32
	poll := func(_ context.Context) (string, error) {
		if calls.Add(1) < 3 {
			return "", &wait.RetryableError{Err: errors.New("transient fetch failure")}
		}
		return "Ready", nil
	}

	var retries atomic.Int32
	err := wait.Until(context.Background(), poll, phaseCond, wait.Options{
		Interval: time.Millisecond,
		Timeout:  time.Second,
		OnRetry:  func(_ error) { retries.Add(1) },
	})

	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
	assert.Equal(t, int32(2), retries.Load())
}

func TestUntil_ZeroTimeoutMeansNoTimeout(t *testing.T) {
	t.Parallel()

	// Timeout left unset (zero). A pending-then-ready poll must still succeed
	// rather than time out immediately.
	var calls atomic.Int32
	poll := func(_ context.Context) (string, error) {
		if calls.Add(1) < 2 {
			return "Provisioning", nil
		}
		return "Ready", nil
	}

	err := wait.Until(context.Background(), poll, phaseCond, wait.Options{
		Interval: time.Millisecond,
	})
	require.NoError(t, err)
}

type widget struct {
	Name string
}

func TestFetchPoll_NotFound_IsSuccessPolicy_ReturnsNil(t *testing.T) {
	t.Parallel()

	poll := wait.FetchPoll[widget]("widget", "my-widget", wait.NotFoundIsSuccess, func(context.Context) (int, string, *widget, error) {
		return http.StatusNotFound, "404 Not Found", nil, nil
	})

	v, err := poll(context.Background())
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestFetchPoll_NotFound_TerminalPolicy_ReturnsError(t *testing.T) {
	t.Parallel()

	poll := wait.FetchPoll[widget]("widget", "my-widget", wait.NotFoundTerminal, func(context.Context) (int, string, *widget, error) {
		return http.StatusNotFound, "404 Not Found", nil, nil
	})

	_, err := poll(context.Background())
	require.Error(t, err)
	var re *wait.RetryableError
	assert.NotErrorAs(t, err, &re, "a 404 under NotFoundTerminal must be terminal, not retryable")
	assert.Contains(t, err.Error(), `widget "my-widget" no longer exists`)
}

func TestFetchPoll_StillPresent_ReturnsBody(t *testing.T) {
	t.Parallel()

	want := &widget{Name: "my-widget"}
	poll := wait.FetchPoll[widget]("widget", "my-widget", wait.NotFoundIsSuccess, func(context.Context) (int, string, *widget, error) {
		return http.StatusOK, "200 OK", want, nil
	})

	v, err := poll(context.Background())
	require.NoError(t, err)
	assert.Same(t, want, v)
}

func TestFetchPoll_EmptyBody_Retryable(t *testing.T) {
	t.Parallel()

	poll := wait.FetchPoll[widget]("widget", "my-widget", wait.NotFoundTerminal, func(context.Context) (int, string, *widget, error) {
		return http.StatusOK, "200 OK", nil, nil
	})

	_, err := poll(context.Background())
	require.Error(t, err)
	var re *wait.RetryableError
	require.ErrorAs(t, err, &re)
	assert.Contains(t, err.Error(), `empty response body fetching widget "my-widget"`)
}

func TestFetchPoll_Unauthorized_Terminal(t *testing.T) {
	t.Parallel()

	poll := wait.FetchPoll[widget]("widget", "my-widget", wait.NotFoundTerminal, func(context.Context) (int, string, *widget, error) {
		return http.StatusUnauthorized, "401 Unauthorized", nil, nil
	})

	_, err := poll(context.Background())
	require.Error(t, err)
	var re *wait.RetryableError
	assert.NotErrorAs(t, err, &re, "401 must be terminal, not retryable")
	assert.Contains(t, err.Error(), "run 'everestctl auth login' again")
}

func TestFetchPoll_UnexpectedStatus_Retryable(t *testing.T) {
	t.Parallel()

	poll := wait.FetchPoll[widget]("widget", "my-widget", wait.NotFoundTerminal, func(context.Context) (int, string, *widget, error) {
		return http.StatusInternalServerError, "500 Internal Server Error", nil, nil
	})

	_, err := poll(context.Background())
	require.Error(t, err)
	var re *wait.RetryableError
	require.ErrorAs(t, err, &re)
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}

func TestFetchPoll_FetchError_Retryable(t *testing.T) {
	t.Parallel()

	poll := wait.FetchPoll[widget]("widget", "my-widget", wait.NotFoundTerminal, func(context.Context) (int, string, *widget, error) {
		return 0, "", nil, errors.New("connection reset")
	})

	_, err := poll(context.Background())
	require.Error(t, err)
	var re *wait.RetryableError
	require.ErrorAs(t, err, &re)
}

func TestFetchPoll_TokenRefreshFailure_Terminal(t *testing.T) {
	t.Parallel()

	poll := wait.FetchPoll[widget]("widget", "my-widget", wait.NotFoundTerminal, func(context.Context) (int, string, *widget, error) {
		return 0, "", nil, fmt.Errorf("refresh: %w", authcli.ErrTokenRefresh)
	})

	_, err := poll(context.Background())
	require.Error(t, err)
	var re *wait.RetryableError
	assert.NotErrorAs(t, err, &re, "a failed token refresh must be terminal, not retryable")
	assert.ErrorIs(t, err, authcli.ErrTokenRefresh)
}
