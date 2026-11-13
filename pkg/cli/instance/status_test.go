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

package instance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

func TestInstanceStatus_HappyPath(t *testing.T) {
	t.Parallel()

	inst := instanceFromJSON(`{"metadata":{"name":"my-mongo","generation":1},"status":{
		"observedGeneration":1,"phase":"Provisioned","version":"1.22.0",
		"components":[{"name":"engine","readyReplicas":3,"replicas":3}],
		"conditions":[
			{"type":"Available","status":"True","reason":"Available","message":""},
			{"type":"Ready","status":"True","reason":"Ready","message":"The instance is ready"}
		]
	}}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inst)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewInstanceStatusRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), StatusOptions{
		Name:      "my-mongo",
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.NoError(t, err)
}

// The cases come from the fixtures shared with the other readers of an
// Instance's status, so what is printed here is what they are held to.
func TestPrintInstanceStatus_Fixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fixture string
		want    string
	}{
		{
			fixture: "rolling-upgrade",
			want: `Instance:  rolling-upgrade
Namespace: demo
Status:    Up · Upgrading (for 4m)
Version:   -
Message:   Rolling out version 8.0.12 (1 of 3 pods updated)

Checks:
  0 of 1 passing, 1 pending
  PENDING     MESSAGE
  Pods ready  engine: 1 pod is not ready

In progress:
  OPERATION  AGE  MESSAGE
  Upgrading  4m   Rolling out version 8.0.12 (1 of 3 pods updated)

Conditions:
  TYPE            STATUS   REASON               MESSAGE
  Available       True     Available            -
  Ready           False    Upgrading            Rolling out version 8.0.12 (1 of 3 pods updated)
  Reconciling     True     Upgrading            Rolling out version 8.0.12 (1 of 3 pods updated)
  Stalled         False    NoProblems           -
  NeedsAttention  False    NoProblems           -
  PodsReady       Unknown  NotReady             engine: 1 pod is not ready
  Upgrading       True     MinorVersionRolling  Rolling out version 8.0.12 (1 of 3 pods updated)
`,
		},
		{
			// The problem began 18 minutes ago and the Instance gave up on it
			// 8 minutes ago: the header counts from the stall.
			fixture: "crash-loop-all-replicas-stalled",
			want: `Instance:  crash-loop-all-replicas-stalled
Namespace: demo
Status:    Down · Crash loop (for 8m)
Version:   -
Message:   engine: container mongod last exited with Error (exit code 14)

Problems:
  PROBLEM     CHECK       AGE  MESSAGE
  Crash loop  Pods ready  18m  engine: container mongod last exited with Error (exit code 14)

Checks:
  0 of 1 passing, 1 failing

Conditions:
  TYPE            STATUS  REASON            MESSAGE
  Available       False   NotAvailable      No replica set member is primary
  Ready           False   CrashLoopBackOff  engine: container mongod last exited with Error (exit code 14)
  Reconciling     False   CrashLoopBackOff  -
  Stalled         True    CrashLoopBackOff  engine: container mongod last exited with Error (exit code 14)
  NeedsAttention  True    CrashLoopBackOff  engine: container mongod last exited with Error (exit code 14)
  PodsReady       False   CrashLoopBackOff  engine: container mongod last exited with Error (exit code 14)
`,
		},
		{
			// No check is behind a rejected spec, so the cause the Instance
			// needs attention for is the problem.
			fixture: "rejected-spec",
			want: `Instance:  rejected-spec
Namespace: demo
Status:    Invalid configuration (for 3m)
Version:   -
Message:   spec.components.engine.replicas: must be odd

Problems:
  PROBLEM                CHECK  AGE  MESSAGE
  Invalid configuration  -      3m   spec.components.engine.replicas: must be odd

Conditions:
  TYPE            STATUS  REASON            MESSAGE
  Available       False   NotCreated        The database has not been available yet
  Ready           False   ValidationFailed  spec.components.engine.replicas: must be odd
  Reconciling     False   ValidationFailed  -
  Stalled         True    ValidationFailed  spec.components.engine.replicas: must be odd
  NeedsAttention  True    ValidationFailed  spec.components.engine.replicas: must be odd
`,
		},
		{
			// The status is about the spec before the fix: the header does
			// not repeat it, and the note says why the details still do.
			fixture: "fix-pushed-to-stalled",
			want: `Instance:  fix-pushed-to-stalled
Namespace: demo
Status:    Creating
Version:   -
Message:   -
Note:      The latest change has not been observed yet. The details below describe the instance before it.

Problems:
  PROBLEM                CHECK  AGE  MESSAGE
  Invalid configuration  -      30m  spec.components.engine.replicas: must be odd

Conditions:
  TYPE            STATUS  REASON            MESSAGE
  Available       False   NotCreated        The database has not been available yet
  Ready           False   ValidationFailed  spec.components.engine.replicas: must be odd
  Reconciling     False   ValidationFailed  -
  Stalled         True    ValidationFailed  spec.components.engine.replicas: must be odd
  NeedsAttention  True    ValidationFailed  spec.components.engine.replicas: must be odd
`,
		},
		{
			fixture: "never-reconciled-old",
			want: `Instance:  never-reconciled-old
Namespace: demo
Status:    Not picked up by provider
Version:   -
Message:   -
Note:      No provider is reconciling this instance. Check that its provider is installed and running.
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.fixture, func(t *testing.T) {
			t.Parallel()
			fixture := displayFixtureNamed(t, tc.fixture)
			var out bytes.Buffer

			printInstanceStatus(&out, &fixture.Instance, "demo", fixture.Now)

			assert.Equal(t, tc.want, out.String())
		})
	}
}

// Every section at once, on an Instance that is ready: what does not gate
// readiness is still shown, and a condition of an unknown type is left to
// the table of all conditions.
func TestPrintInstanceStatus_AllSections(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	inst := instanceFromJSON(`{"metadata":{"name":"orders","generation":7},"status":{
		"observedGeneration":7,"phase":"Provisioned","version":"8.0.12",
		"components":[
			{"name":"engine","readyReplicas":3,"replicas":3},
			{"name":"proxy","readyReplicas":2,"replicas":2}
		],
		"conditions":[
			{"type":"Available","status":"True","reason":"Available","message":"","lastTransitionTime":"2026-02-01T12:00:00Z"},
			{"type":"Ready","status":"True","reason":"StorageResizing","message":"Expanding volumes to 50Gi","lastTransitionTime":"2026-02-27T12:00:00Z"},
			{"type":"Reconciling","status":"False","reason":"NoProblems","message":"","lastTransitionTime":"2026-02-27T12:00:00Z"},
			{"type":"Stalled","status":"False","reason":"NoProblems","message":"","lastTransitionTime":"2026-02-27T12:00:00Z"},
			{"type":"NeedsAttention","status":"False","reason":"NoProblems","message":"","lastTransitionTime":"2026-02-27T12:00:00Z"},
			{"type":"ReferencesResolved","status":"True","reason":"Resolved","message":"","lastTransitionTime":"2026-02-01T12:00:00Z"},
			{"type":"Synced","status":"True","reason":"Synced","message":"","lastTransitionTime":"2026-02-01T12:00:00Z"},
			{"type":"PodsScheduled","status":"True","reason":"Scheduled","message":"","lastTransitionTime":"2026-02-01T12:00:00Z"},
			{"type":"PodsReady","status":"True","reason":"Ready","message":"","lastTransitionTime":"2026-02-27T12:00:00Z"},
			{"type":"EngineHealthy","status":"True","reason":"Healthy","message":"","lastTransitionTime":"2026-02-01T12:00:00Z"},
			{"type":"MonitoringReady","status":"False","reason":"MonitoringNotReady","lastTransitionTime":"2026-03-01T10:00:00Z",
			 "message":"pmm: the destination is not ready\nvictoria: the destination is not ready"},
			{"type":"StorageResizing","status":"True","reason":"ExpansionTriggered","message":"Expanding volumes to 50Gi","lastTransitionTime":"2026-03-01T11:54:00Z"},
			{"type":"Upgrading","status":"False","reason":"Succeeded","message":"Upgraded to 8.0.12","lastTransitionTime":"2026-02-27T12:00:00Z"},
			{"type":"MaintenancePending","status":"True","reason":"AwaitingApproval","message":"1 action awaits approval","lastTransitionTime":"2026-03-01T09:00:00Z"},
			{"type":"acme.example.com/Sharded","status":"True","reason":"Balanced","message":"","lastTransitionTime":"2026-02-01T12:00:00Z"}
		]
	}}`)
	var out bytes.Buffer

	printInstanceStatus(&out, inst, "shop", now)

	want := `Instance:  orders
Namespace: shop
Status:    Up · Resizing storage (for 6m)
Version:   8.0.12
Message:   Expanding volumes to 50Gi

Components:
  NAME    READY
  engine  3/3
  proxy   2/2

Problems:
  PROBLEM               CHECK             AGE  MESSAGE
  Monitoring not ready  Monitoring ready  2h   pmm: the destination is not ready
                                               victoria: the destination is not ready

Checks:
  5 of 6 passing, 1 failing

In progress:
  OPERATION         AGE  MESSAGE
  Resizing storage  6m   Expanding volumes to 50Gi

Notices:
  NOTICE               AGE  MESSAGE
  Maintenance pending  3h   1 action awaits approval

Conditions:
  TYPE                      STATUS  REASON              MESSAGE
  Available                 True    Available           -
  Ready                     True    StorageResizing     Expanding volumes to 50Gi
  Reconciling               False   NoProblems          -
  Stalled                   False   NoProblems          -
  NeedsAttention            False   NoProblems          -
  ReferencesResolved        True    Resolved            -
  Synced                    True    Synced              -
  PodsScheduled             True    Scheduled           -
  PodsReady                 True    Ready               -
  EngineHealthy             True    Healthy             -
  MonitoringReady           False   MonitoringNotReady  pmm: the destination is not ready
                                                        victoria: the destination is not ready
  StorageResizing           True    ExpansionTriggered  Expanding volumes to 50Gi
  Upgrading                 False   Succeeded           Upgraded to 8.0.12
  MaintenancePending        True    AwaitingApproval    1 action awaits approval
  acme.example.com/Sharded  True    Balanced            -
`
	assert.Equal(t, want, out.String())
}

// A watch prints again when what it shows changed, not because the durations
// in it went up.
func TestChangeKey(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	soon, later := created.Add(time.Minute), created.Add(time.Hour)
	const upgradingDoc = `{"metadata":{"name":"my-mongo","generation":1,"resourceVersion":%q},"status":{
		"observedGeneration":1,"phase":"Provisioned","conditions":[
			{"type":"Ready","status":"True","reason":"Ready","message":"","lastTransitionTime":"2026-03-01T12:00:00Z"},
			{"type":"Upgrading","status":"True","reason":"Rolling","message":%q,"lastTransitionTime":"2026-03-01T12:00:00Z"}
		]}}`
	upgrading := func(resourceVersion, progress string) *client.Instance {
		return instanceFromJSON(fmt.Sprintf(upgradingDoc, resourceVersion, progress))
	}
	unreconciled := func() *client.Instance {
		return instanceFromJSON(`{"metadata":{"name":"my-mongo","generation":1,"creationTimestamp":"2026-03-01T12:00:00Z"},
			"status":{"observedGeneration":-1}}`)
	}

	pretty := NewInstanceStatusRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	raw := NewInstanceStatusRunner(Config{Pretty: false}, zap.NewNop().Sugar())

	tests := []struct {
		name          string
		runner        *InstanceStatusRunner
		before, after *client.Instance
		wantSame      bool
	}{
		{
			name:     "time passing on an unchanged instance prints nothing",
			runner:   pretty,
			before:   upgrading("1", "1 of 3 pods"),
			after:    upgrading("1", "1 of 3 pods"),
			wantSame: true,
		},
		{
			name:   "a change in what is shown prints again",
			runner: pretty,
			before: upgrading("1", "1 of 3 pods"),
			after:  upgrading("2", "2 of 3 pods"),
		},
		{
			name:     "a change in what is not shown prints nothing",
			runner:   pretty,
			before:   upgrading("1", "1 of 3 pods"),
			after:    upgrading("2", "1 of 3 pods"),
			wantSame: true,
		},
		{
			name:   "an instance nobody reconciles comes to read as not picked up",
			runner: pretty,
			before: unreconciled(),
			after:  unreconciled(),
		},
		{
			name:     "the raw object is emitted again only when it changed",
			runner:   raw,
			before:   unreconciled(),
			after:    unreconciled(),
			wantSame: true,
		},
		{
			name:   "the raw object changed",
			runner: raw,
			before: upgrading("1", "1 of 3 pods"),
			after:  upgrading("2", "1 of 3 pods"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			before := tc.runner.changeKey(tc.before, "everest", soon)
			after := tc.runner.changeKey(tc.after, "everest", later)
			if tc.wantSame {
				assert.Equal(t, before, after)
			} else {
				assert.NotEqual(t, before, after)
			}
		})
	}
}

func TestPrintConditionTable_MultiLineMessage(t *testing.T) {
	t.Parallel()

	status := `{"status":{"conditions":[
		{"type":"Ready","status":"True","reason":"Ready","message":"","lastTransitionTime":"2026-10-02T12:00:00Z"},
		{"type":"PodsScheduled","status":"False","reason":"Unschedulable","lastTransitionTime":"2026-10-02T12:00:00Z",
		 "message":"configServer: 2 of 3 pods cannot be scheduled\nengine: 1 of 5 pods cannot be scheduled"}
	]}}`
	var inst client.Instance
	require.NoError(t, json.Unmarshal([]byte(status), &inst))
	var out bytes.Buffer

	printConditionTable(&out, &inst)

	want := "\nConditions:\n" +
		"  TYPE           STATUS  REASON         MESSAGE\n" +
		"  Ready          True    Ready          -\n" +
		"  PodsScheduled  False   Unschedulable  configServer: 2 of 3 pods cannot be scheduled\n" +
		"                                        engine: 1 of 5 pods cannot be scheduled\n"
	assert.Equal(t, want, out.String())
}

func TestInstanceStatus_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewInstanceStatusRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), StatusOptions{
		Name:      "no-such-instance",
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInstanceStatus_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewInstanceStatusRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), StatusOptions{
		Name:      "my-mongo",
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response")
}

func TestInstanceStatus_NoActiveContext(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		APIVersion: "config.openeverest.io/v1alpha1",
		Kind:       "ClientConfig",
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewInstanceStatusRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), StatusOptions{
		Name:      "my-mongo",
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active context")
}

func TestInstanceStatus_JSONOutput(t *testing.T) {
	t.Parallel()

	inst := minimalInst()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(inst)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	// Pretty=false → JSON path
	runner := NewInstanceStatusRunner(Config{Pretty: false}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), StatusOptions{
		Name:      "my-mongo",
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.NoError(t, err)
}

func minimalInst() *client.Instance {
	return instanceFromJSON(`{"metadata":{"name":"my-mongo","generation":1},"status":{"observedGeneration":1,"phase":"Provisioned"}}`)
}

// instanceFromJSON decodes an Instance the way the API serves it. Building
// it from JSON keeps the tests independent of the generated status struct,
// which changes whenever the Instance status gains a field.
func instanceFromJSON(doc string) *client.Instance {
	inst := &client.Instance{}
	if err := json.Unmarshal([]byte(doc), inst); err != nil {
		panic(err)
	}
	return inst
}

func TestWatch_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	polled := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// The first call is the initial fetch, the second the first poll.
		if calls.Add(1) == 2 {
			close(polled)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(minimalInst())
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	ctx, cancel := context.WithCancel(context.Background())

	runner := NewInstanceStatusRunner(Config{Pretty: true}, zap.NewNop().Sugar())

	// Cancel once the watch has polled. Waiting a fixed time instead fails on
	// a busy machine, where the first fetch alone can take that long.
	go func() {
		select {
		case <-polled:
		case <-time.After(30 * time.Second):
		}
		cancel()
	}()

	err := runner.Run(ctx, StatusOptions{
		Name:      "my-mongo",
		Namespace: "everest",
		Cluster:   "main",
		Watch:     true,
		Interval:  50 * time.Millisecond,
	}, cfgPath)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, calls.Load(), int32(2), "expected the watch to poll before the cancel")
}

func TestWatch_StopsOnTokenExpiry(t *testing.T) {
	t.Parallel()

	// POST = token refresh; return 401 so proactive rotation fails and the error surfaces.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(minimalInst())
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	// set token to expire in 10ms — well within the 30s proactive window
	cfg.Users[0].User.ExpiresAt = time.Now().Add(10 * time.Millisecond)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewInstanceStatusRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(context.Background(), StatusOptions{
		Name:      "my-mongo",
		Namespace: "everest",
		Cluster:   "main",
		Watch:     true,
		Interval:  50 * time.Millisecond,
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access token expired")
}

func TestWatch_StopsOnInstanceDeleted(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			// first call (from Run) succeeds
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(minimalInst())
			return
		}
		// subsequent calls: instance deleted
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewInstanceStatusRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(context.Background(), StatusOptions{
		Name:      "my-mongo",
		Namespace: "everest",
		Cluster:   "main",
		Watch:     true,
		Interval:  30 * time.Millisecond,
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer exists")
}

func TestWatch_TransientErrorKeepsPolling(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			// first call from Run — succeed
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(minimalInst())
			return
		}
		if n == 2 {
			// transient error on first watch tick
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// third call — succeed, then cancel
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(minimalInst())
		cancel()
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewInstanceStatusRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(ctx, StatusOptions{
		Name:      "my-mongo",
		Namespace: "everest",
		Cluster:   "main",
		Watch:     true,
		Interval:  30 * time.Millisecond,
	}, cfgPath)
	// context cancel → nil error
	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load(), "should have made 3 calls despite transient error")
}
