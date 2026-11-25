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

package gate

import (
	"context"
	"time"

	"github.com/openeverest/openeverest/v2/contract"
)

// DefaultInterval is how often a held build reads its root object and the
// core again when Runner.Interval is zero. A held build waits for a core
// upgrade, which takes minutes, so reading more often would only load the
// API server.
const DefaultInterval = 15 * time.Second

// firstRetry is how soon a read or a write is tried again after it fails.
// Most failures at startup are the API server being briefly unreachable,
// and a build that already runs should not wait a whole interval to start
// again. The delay doubles on every failure in a row, up to the interval.
const firstRetry = time.Second

// Runner holds a new build of an extension at startup until the core
// installed in the cluster supports it, and then records on the root object
// the version that started. It reads and writes the cluster through the
// functions it is given: ReadRoot, ReadCore and WriteObserved are required.
type Runner struct {
	// Name is the name of the extension's root object, by which the core's
	// acknowledgements name the extension.
	Name string
	// Linked is the contract version the build links. It is
	// contract.Version when empty; only tests set it.
	Linked string
	// Interval is how often a held build reads again. It is
	// DefaultInterval when zero.
	Interval time.Duration

	// ReadRoot reads the root object. It returns the zero Root when the
	// object does not exist.
	ReadRoot func(ctx context.Context) (Root, error)
	// ReadCore reads what the core publishes on the CRD of the root kind,
	// with CoreFromCRD. It returns the zero Core when the CRD does not
	// exist. It must read the API server rather than a cache that may be
	// behind, as a build held on a stale window would wait for nothing.
	ReadCore func(ctx context.Context) (Core, error)
	// WriteObserved writes version on the root object as the contract
	// version observed on its status.
	WriteObserved func(ctx context.Context, version string) error

	// Held, when set, is told that the build is held: with the first
	// decision that holds it and with every one whose message differs from
	// the last. It is where a workload logs the decision and records an
	// Event on its root object.
	Held func(ctx context.Context, d Decision)
	// WriteFailed, when set, is told about every write of the observed
	// version that fails.
	WriteFailed func(ctx context.Context, err error)
}

// Wait decides whether the build may start, again every Interval while it
// is held, and returns the decision that lets it start, or ctx's error once
// ctx is done. A build with the version observed on its root object starts
// without reading the core, so a core that cannot be read never keeps a
// build that already runs from starting again.
func (r Runner) Wait(ctx context.Context) (Decision, error) {
	linked := r.linked()
	var last string
	retry := firstRetry
	for {
		d, failed := r.decide(ctx, linked)
		if d.Start {
			return d, nil
		}
		if d.Message != last && r.Held != nil {
			r.Held(ctx, d)
		}
		last = d.Message
		wait := r.interval()
		if failed {
			wait, retry = min(retry, wait), min(2*retry, wait)
		} else {
			retry = firstRetry
		}
		if err := sleep(ctx, wait); err != nil {
			return Decision{}, err
		}
	}
}

// Record writes the version the build links on the root object, as its
// observed version, after d let the build start. It writes again every
// Interval until a write succeeds, as the root object may not exist yet
// when the build starts: Helm, for one, creates custom resources after the
// workloads of a release. It returns nil once a write succeeds, or at once
// when d is ReasonAlreadyRunning, which has nothing new to record, and
// ctx's error once ctx is done.
func (r Runner) Record(ctx context.Context, d Decision) error {
	if d.Reason == ReasonAlreadyRunning {
		return nil
	}
	linked := r.linked()
	retry := firstRetry
	for {
		err := r.WriteObserved(ctx, linked)
		if err == nil {
			return nil
		}
		if r.WriteFailed != nil {
			r.WriteFailed(ctx, err)
		}
		wait := min(retry, r.interval())
		retry = min(2*retry, r.interval())
		if err := sleep(ctx, wait); err != nil {
			return err
		}
	}
}

// decide reads what Decide needs and decides. failed reports that a read
// failed, in which case the decision holds the build.
func (r Runner) decide(ctx context.Context, linked string) (Decision, bool) {
	root, err := r.ReadRoot(ctx)
	if err != nil {
		return readFailed(linked, err), true
	}
	var core Core
	if root.Observed == "" || root.Observed != linked {
		if core, err = r.ReadCore(ctx); err != nil {
			return readFailed(linked, err), true
		}
	}
	return Decide(r.Name, linked, root, core), false
}

func readFailed(linked string, err error) Decision {
	return Decision{
		Reason:  ReasonReadFailed,
		Message: "this build" + builtFor(linked) + " is waiting until it can read whether the installed OpenEverest supports it: " + err.Error(),
	}
}

func (r Runner) linked() string {
	if r.Linked == "" {
		return contract.Version
	}
	return r.Linked
}

func (r Runner) interval() time.Duration {
	if r.Interval <= 0 {
		return DefaultInterval
	}
	return r.Interval
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
