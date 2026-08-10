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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

func TestInstanceStatus_HappyPath(t *testing.T) {
	t.Parallel()

	phase := client.InstanceStatusPhaseReady
	version := "1.22.0"
	msg := "All replicas are ready"
	ready := int32(3)
	total := int32(3)
	state := "Running"
	condStatus := client.InstanceStatusConditionsStatusTrue

	inst := &client.Instance{
		Metadata: &metav1.ObjectMeta{Name: "my-mongo"},
		Status: &struct {
			Backup *struct {
				Storages *[]struct {
					Name string `json:"name"`
					Pitr *struct {
						EarliestRestorableTime *time.Time                                    `json:"earliestRestorableTime,omitempty"`
						LatestRestorableTime   *time.Time                                    `json:"latestRestorableTime,omitempty"`
						Message                *string                                       `json:"message,omitempty"`
						Reason                 *string                                       `json:"reason,omitempty"`
						State                  *client.InstanceStatusBackupStoragesPitrState `json:"state,omitempty"`
					} `json:"pitr,omitempty"`
				} `json:"storages,omitempty"`
			} `json:"backup,omitempty"`
			Components *[]struct {
				PodRefs *[]struct {
					Name string `json:"name"`
				} `json:"podRefs,omitempty"`
				Ready *int32  `json:"ready,omitempty"`
				State *string `json:"state,omitempty"`
				Total *int32  `json:"total,omitempty"`
			} `json:"components,omitempty"`
			Conditions *[]struct {
				LastTransitionTime time.Time                             `json:"lastTransitionTime"`
				Message            string                                `json:"message"`
				ObservedGeneration *int64                                `json:"observedGeneration,omitempty"`
				Reason             string                                `json:"reason"`
				Status             client.InstanceStatusConditionsStatus `json:"status"`
				Type               string                                `json:"type"`
			} `json:"conditions,omitempty"`
			ConnectionSecretRef *struct {
				Name string `json:"name"`
			} `json:"connectionSecretRef,omitempty"`
			Message *string                     `json:"message,omitempty"`
			Phase   *client.InstanceStatusPhase `json:"phase,omitempty"`
			Version *string                     `json:"version,omitempty"`
		}{
			Phase:   &phase,
			Version: &version,
			Message: &msg,
			Components: &[]struct {
				PodRefs *[]struct {
					Name string `json:"name"`
				} `json:"podRefs,omitempty"`
				Ready *int32  `json:"ready,omitempty"`
				State *string `json:"state,omitempty"`
				Total *int32  `json:"total,omitempty"`
			}{
				{Ready: &ready, Total: &total, State: &state},
			},
			Conditions: &[]struct {
				LastTransitionTime time.Time                             `json:"lastTransitionTime"`
				Message            string                                `json:"message"`
				ObservedGeneration *int64                                `json:"observedGeneration,omitempty"`
				Reason             string                                `json:"reason"`
				Status             client.InstanceStatusConditionsStatus `json:"status"`
				Type               string                                `json:"type"`
			}{
				{Type: "Available", Status: condStatus, Reason: "Ready", Message: "All replicas are ready"},
			},
		},
	}

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

	phase := client.InstanceStatusPhaseReady
	inst := &client.Instance{
		Metadata: &metav1.ObjectMeta{Name: "my-mongo"},
		Status: &struct {
			Backup *struct {
				Storages *[]struct {
					Name string `json:"name"`
					Pitr *struct {
						EarliestRestorableTime *time.Time                                    `json:"earliestRestorableTime,omitempty"`
						LatestRestorableTime   *time.Time                                    `json:"latestRestorableTime,omitempty"`
						Message                *string                                       `json:"message,omitempty"`
						Reason                 *string                                       `json:"reason,omitempty"`
						State                  *client.InstanceStatusBackupStoragesPitrState `json:"state,omitempty"`
					} `json:"pitr,omitempty"`
				} `json:"storages,omitempty"`
			} `json:"backup,omitempty"`
			Components *[]struct {
				PodRefs *[]struct {
					Name string `json:"name"`
				} `json:"podRefs,omitempty"`
				Ready *int32  `json:"ready,omitempty"`
				State *string `json:"state,omitempty"`
				Total *int32  `json:"total,omitempty"`
			} `json:"components,omitempty"`
			Conditions *[]struct {
				LastTransitionTime time.Time                             `json:"lastTransitionTime"`
				Message            string                                `json:"message"`
				ObservedGeneration *int64                                `json:"observedGeneration,omitempty"`
				Reason             string                                `json:"reason"`
				Status             client.InstanceStatusConditionsStatus `json:"status"`
				Type               string                                `json:"type"`
			} `json:"conditions,omitempty"`
			ConnectionSecretRef *struct {
				Name string `json:"name"`
			} `json:"connectionSecretRef,omitempty"`
			Message *string                     `json:"message,omitempty"`
			Phase   *client.InstanceStatusPhase `json:"phase,omitempty"`
			Version *string                     `json:"version,omitempty"`
		}{
			Phase: &phase,
		},
	}

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
	phase := client.InstanceStatusPhaseReady
	return &client.Instance{
		Metadata: &metav1.ObjectMeta{Name: "my-mongo"},
		Status: &struct {
			Backup *struct {
				Storages *[]struct {
					Name string `json:"name"`
					Pitr *struct {
						EarliestRestorableTime *time.Time                                    `json:"earliestRestorableTime,omitempty"`
						LatestRestorableTime   *time.Time                                    `json:"latestRestorableTime,omitempty"`
						Message                *string                                       `json:"message,omitempty"`
						Reason                 *string                                       `json:"reason,omitempty"`
						State                  *client.InstanceStatusBackupStoragesPitrState `json:"state,omitempty"`
					} `json:"pitr,omitempty"`
				} `json:"storages,omitempty"`
			} `json:"backup,omitempty"`
			Components *[]struct {
				PodRefs *[]struct {
					Name string `json:"name"`
				} `json:"podRefs,omitempty"`
				Ready *int32  `json:"ready,omitempty"`
				State *string `json:"state,omitempty"`
				Total *int32  `json:"total,omitempty"`
			} `json:"components,omitempty"`
			Conditions *[]struct {
				LastTransitionTime time.Time                             `json:"lastTransitionTime"`
				Message            string                                `json:"message"`
				ObservedGeneration *int64                                `json:"observedGeneration,omitempty"`
				Reason             string                                `json:"reason"`
				Status             client.InstanceStatusConditionsStatus `json:"status"`
				Type               string                                `json:"type"`
			} `json:"conditions,omitempty"`
			ConnectionSecretRef *struct {
				Name string `json:"name"`
			} `json:"connectionSecretRef,omitempty"`
			Message *string                     `json:"message,omitempty"`
			Phase   *client.InstanceStatusPhase `json:"phase,omitempty"`
			Version *string                     `json:"version,omitempty"`
		}{Phase: &phase},
	}
}

func TestWatch_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
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

	// cancel after a short delay
	go func() {
		time.Sleep(150 * time.Millisecond)
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
	assert.GreaterOrEqual(t, calls.Load(), int32(2), "expected at least 2 ticks before cancel")
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
