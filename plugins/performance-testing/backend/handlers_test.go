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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestServer(secrets ...*corev1.Secret) *server {
	kube := fake.NewSimpleClientset()
	for _, secret := range secrets {
		_, _ = kube.CoreV1().Secrets(secret.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	}
	return &server{kube: kube, store: newMemoryStore()}
}

func testSecret(namespace, instance string) *corev1.Secret {
	return &corev1.Secret{ //nolint:exhaustruct // TypeMeta/other fields unused by the fake clientset in these tests
		ObjectMeta: metav1.ObjectMeta{
			Name:      credentialsSecretName(instance),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"pgbouncer-host": []byte("pg-poc-pgbouncer.everest.svc"),
			"pgbouncer-port": []byte("5432"),
			"user":           []byte("postgres"),
			"password":       []byte("does-not-matter-for-this-test"),
		},
	}
}

func TestHandleCreateRun_MissingSecret_Returns404(t *testing.T) {
	t.Parallel()

	s := newTestServer()
	body := strings.NewReader(`{"instance":"pg-poc","namespace":"everest","workload":"smoke"}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/runs", body)
	rec := httptest.NewRecorder()

	s.handleCreateRun(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleCreateRun_UnknownWorkload_Returns400(t *testing.T) {
	t.Parallel()

	s := newTestServer(testSecret("everest", "pg-poc"))
	body := strings.NewReader(`{"instance":"pg-poc","namespace":"everest","workload":"not-a-workload"}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/runs", body)
	rec := httptest.NewRecorder()

	s.handleCreateRun(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleCreateRun_MissingFields_Returns400(t *testing.T) {
	t.Parallel()

	s := newTestServer()
	body := strings.NewReader(`{"workload":"smoke"}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/runs", body)
	rec := httptest.NewRecorder()

	s.handleCreateRun(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleCreateRun_ValidRequest_CreatesJobAndStoresRun(t *testing.T) {
	t.Parallel()

	s := newTestServer(testSecret("everest", "pg-poc"))
	body := strings.NewReader(`{"instance":"pg-poc","namespace":"everest","workload":"smoke"}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/runs", body)
	rec := httptest.NewRecorder()

	s.handleCreateRun(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	runID := resp["id"]
	require.NotEmpty(t, runID)

	// The run is stored immediately (Running), independent of the
	// background job-tracking goroutine's eventual completion.
	run, ok := s.store.Get(runID)
	require.True(t, ok)
	assert.Equal(t, RunStatusRunning, run.Status)
	assert.Equal(t, "pg-poc", run.Instance)

	job, err := s.kube.BatchV1().Jobs("everest").Get(req.Context(), "perf-test-"+runID, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, sysbenchImage, job.Spec.Template.Spec.Containers[0].Image)
}

func TestHandleGetRun_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	s := newTestServer()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/runs/does-not-exist", nil)
	req.SetPathValue("id", "does-not-exist")
	rec := httptest.NewRecorder()

	s.handleGetRun(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleGetRun_Found_ReturnsRun(t *testing.T) {
	t.Parallel()

	s := newTestServer()
	require.NoError(t, s.store.Save(Run{ID: "run-1", Instance: "pg-poc", Status: RunStatusSucceeded}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/runs/run-1", nil)
	req.SetPathValue("id", "run-1")
	rec := httptest.NewRecorder()

	s.handleGetRun(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var got Run
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "pg-poc", got.Instance)
}

func TestHandleListRuns_ReturnsAllStoredRuns(t *testing.T) {
	t.Parallel()

	s := newTestServer()
	require.NoError(t, s.store.Save(Run{ID: "run-1", Instance: "pg-poc"}))
	require.NoError(t, s.store.Save(Run{ID: "run-2", Instance: "pg-poc"}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/runs", nil)
	rec := httptest.NewRecorder()

	s.handleListRuns(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var got []Run
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Len(t, got, 2)
}
