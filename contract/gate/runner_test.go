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

package gate_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/contract/gate"
)

// cluster serves the reads of a Runner from scripted answers: the last one
// repeats once the others are used up.
type cluster struct {
	roots    []rootAnswer
	cores    []coreAnswer
	writes   []error
	written  []string
	coreRead int
	held     []gate.Decision
	failed   []error
}

type rootAnswer struct {
	root gate.Root
	err  error
}

type coreAnswer struct {
	core gate.Core
	err  error
}

func next[T any](answers []T, i int) T {
	return answers[min(i, len(answers)-1)]
}

func (c *cluster) runner() gate.Runner {
	var rootRead, write int
	return gate.Runner{
		Name:     "psmdb",
		Linked:   "2.1.0",
		Interval: time.Millisecond,
		ReadRoot: func(context.Context) (gate.Root, error) {
			a := next(c.roots, rootRead)
			rootRead++
			return a.root, a.err
		},
		ReadCore: func(context.Context) (gate.Core, error) {
			a := next(c.cores, c.coreRead)
			c.coreRead++
			return a.core, a.err
		},
		WriteObserved: func(_ context.Context, version string) error {
			c.written = append(c.written, version)
			err := next(c.writes, write)
			write++
			return err
		},
		Held:        func(_ context.Context, d gate.Decision) { c.held = append(c.held, d) },
		WriteFailed: func(_ context.Context, err error) { c.failed = append(c.failed, err) },
	}
}

// supported and tooOld answer with a window that supports the runner's
// build, and with one that is too old for it.
func supported() coreAnswer { return coreAnswer{core: window("2.0.0", "2.1.0")} }
func tooOld() coreAnswer    { return coreAnswer{core: window("2.0.0", "2.0.0")} }

func TestRunnerWait(t *testing.T) {
	t.Parallel()

	errUnreachable := errors.New("connection refused")
	tests := []struct {
		name       string
		roots      []rootAnswer
		cores      []coreAnswer
		reason     string
		coreReads  int
		heldReason []string
	}{
		{
			name:      "a supported build starts at once",
			roots:     []rootAnswer{{}},
			cores:     []coreAnswer{supported()},
			reason:    contract.ReasonSupported,
			coreReads: 1,
		},
		{
			name:   "the build that last started does not read the core",
			roots:  []rootAnswer{{root: gate.Root{Observed: "2.1.0"}}},
			cores:  []coreAnswer{{err: errUnreachable}},
			reason: gate.ReasonAlreadyRunning,
		},
		{
			name:       "a held build starts once core is upgraded, reported once while the message stays",
			roots:      []rootAnswer{{root: gate.Root{Observed: "2.0.0"}}},
			cores:      []coreAnswer{tooOld(), tooOld(), tooOld(), supported()},
			reason:     contract.ReasonSupported,
			coreReads:  4,
			heldReason: []string{contract.ReasonCoreTooOld},
		},
		{
			name:       "every change of the decision is reported",
			roots:      []rootAnswer{{}},
			cores:      []coreAnswer{{core: gate.Core{}}, tooOld(), tooOld(), supported()},
			reason:     contract.ReasonSupported,
			coreReads:  4,
			heldReason: []string{contract.ReasonCoreTooOld, contract.ReasonCoreTooOld},
		},
		{
			name:       "a root object that cannot be read holds the build until it can",
			roots:      []rootAnswer{{err: errUnreachable}, {err: errUnreachable}, {root: gate.Root{Observed: "2.1.0"}}},
			reason:     gate.ReasonAlreadyRunning,
			heldReason: []string{gate.ReasonReadFailed},
		},
		{
			name:       "a core that cannot be read holds a new build until it can",
			roots:      []rootAnswer{{}},
			cores:      []coreAnswer{{err: errUnreachable}, supported()},
			reason:     contract.ReasonSupported,
			coreReads:  2,
			heldReason: []string{gate.ReasonReadFailed},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &cluster{roots: tt.roots, cores: tt.cores}
			d, err := c.runner().Wait(t.Context())
			require.NoError(t, err)
			assert.True(t, d.Start)
			assert.Equal(t, tt.reason, d.Reason)
			assert.Equal(t, tt.coreReads, c.coreRead)
			held := make([]string, 0, len(c.held))
			for _, h := range c.held {
				assert.False(t, h.Start)
				assert.NotEmpty(t, h.Message)
				held = append(held, h.Reason)
			}
			if len(tt.heldReason) == 0 {
				assert.Empty(t, held)
			} else {
				assert.Equal(t, tt.heldReason, held)
			}
		})
	}
}

func TestRunnerWaitReportsWhatItWaitsFor(t *testing.T) {
	t.Parallel()

	c := &cluster{roots: []rootAnswer{{}}, cores: []coreAnswer{tooOld(), supported()}}
	_, err := c.runner().Wait(t.Context())
	require.NoError(t, err)
	require.Len(t, c.held, 1)
	assert.Equal(t, "2.1.0", c.held[0].WaitingFor)
	assert.Equal(t, "a new build (for OpenEverest 2.1.0) is waiting until OpenEverest 2.1.0 is installed", c.held[0].Message)
}

func TestRunnerWaitStopsWithTheContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	c := &cluster{roots: []rootAnswer{{}}, cores: []coreAnswer{tooOld()}}
	r := c.runner()
	r.Interval = time.Hour
	r.Held = func(context.Context, gate.Decision) { cancel() }
	d, err := r.Wait(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, d.Start)
}

func TestRunnerLinksTheContractVersionByDefault(t *testing.T) {
	t.Parallel()

	c := &cluster{roots: []rootAnswer{{root: gate.Root{Observed: contract.Version}}}, writes: []error{nil}}
	r := c.runner()
	r.Linked = ""
	d, err := r.Wait(t.Context())
	require.NoError(t, err)
	assert.Equal(t, gate.ReasonAlreadyRunning, d.Reason)

	require.NoError(t, r.Record(t.Context(), gate.Decision{Start: true, Reason: contract.ReasonSupported}))
	assert.Equal(t, []string{contract.Version}, c.written)
}

func TestRunnerRecord(t *testing.T) {
	t.Parallel()

	errNotFound := errors.New("not found")
	tests := []struct {
		name    string
		reason  string
		writes  []error
		written []string
		failed  int
	}{
		{
			name:    "a new build records its version",
			reason:  contract.ReasonSupported,
			writes:  []error{nil},
			written: []string{"2.1.0"},
		},
		{
			name:    "an acknowledged build records its version",
			reason:  contract.ReasonCoreTooOldAcknowledged,
			writes:  []error{nil},
			written: []string{"2.1.0"},
		},
		{
			name:    "a write is tried again until the root object exists",
			reason:  contract.ReasonSupported,
			writes:  []error{errNotFound, errNotFound, nil},
			written: []string{"2.1.0", "2.1.0", "2.1.0"},
			failed:  2,
		},
		{
			name:   "the build that last started has nothing to record",
			reason: gate.ReasonAlreadyRunning,
			writes: []error{errNotFound},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &cluster{writes: tt.writes}
			require.NoError(t, c.runner().Record(t.Context(), gate.Decision{Start: true, Reason: tt.reason}))
			assert.Equal(t, tt.written, c.written)
			assert.Len(t, c.failed, tt.failed)
		})
	}
}

func TestRunnerRecordStopsWithTheContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	c := &cluster{writes: []error{errors.New("forbidden")}}
	r := c.runner()
	r.Interval = time.Hour
	r.WriteFailed = func(context.Context, error) { cancel() }
	err := r.Record(ctx, gate.Decision{Start: true, Reason: contract.ReasonSupported})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []string{"2.1.0"}, c.written)
}
