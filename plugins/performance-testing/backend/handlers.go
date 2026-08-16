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
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultDBName   = "postgres"
	jobPollInterval = 2 * time.Second
	jobPollTimeout  = (jobActiveDeadline + 60) * time.Second
)

// server holds the dependencies the HTTP handlers need. Deliberately not a
// framework-specific type (no Echo, no chi) — the generic-plugin-template
// reference plugin uses plain net/http, and matching that convention keeps
// this backend swappable/comparable against other real plugins in the org.
type server struct {
	kube  kubernetes.Interface
	store RunStore
}

func (s *server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

type createRunRequest struct {
	Instance  string   `json:"instance"`
	Namespace string   `json:"namespace"`
	Workload  Workload `json:"workload"`
	DBName    string   `json:"dbName,omitempty"`
}

// validateCreateRunRequest checks the request shape and fills in the
// dbName default. Split out of handleCreateRun purely to keep that
// handler short — no independent reuse yet.
func validateCreateRunRequest(req createRunRequest) (string, error) {
	if req.Instance == "" || req.Namespace == "" {
		return "", fmt.Errorf("instance and namespace are required")
	}
	if _, ok := workloadProfiles[req.Workload]; !ok {
		return "", fmt.Errorf("unknown workload %q", req.Workload)
	}
	if req.DBName != "" {
		return req.DBName, nil
	}
	return defaultDBName, nil
}

// checkCredentialsSecretExists confirms the target's credentials Secret is
// present before creating a Job that would otherwise fail inside the
// cluster with a much less legible error.
func (s *server) checkCredentialsSecretExists(ctx context.Context, namespace, instance string) error {
	secretName := credentialsSecretName(instance)
	_, err := s.kube.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("credentials secret %q not found in namespace %q — is %q a running DatabaseCluster? %w",
			secretName, namespace, instance, errNotFound)
	}
	return err
}

var errNotFound = errors.New("not found")

func (s *server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req createRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	dbName, err := validateCreateRunRequest(req)
	if err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	if err := s.checkCredentialsSecretExists(ctx, req.Namespace, req.Instance); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errNotFound) {
			status = http.StatusNotFound
		}
		apiError(w, status, err.Error())
		return
	}

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	job, err := buildJob(runID, req.Instance, req.Namespace, dbName, req.Workload)
	if err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := s.kube.BatchV1().Jobs(req.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		apiError(w, http.StatusInternalServerError, "creating benchmark job: "+err.Error())
		return
	}

	run := Run{ //nolint:exhaustruct // FinishedAt/Result/Error are filled in later, once the job completes.
		ID:        runID,
		Instance:  req.Instance,
		Namespace: req.Namespace,
		Workload:  req.Workload,
		Status:    RunStatusRunning,
		JobName:   created.Name,
		StartedAt: time.Now(),
	}
	if err := s.store.Save(run); err != nil {
		apiError(w, http.StatusInternalServerError, "saving run: "+err.Error())
		return
	}

	// The Job runs prepare+run+cleanup sequentially and can take tens of
	// seconds; the HTTP request returns immediately with the run ID and a
	// background goroutine tracks completion, matching "hits run and sees
	// results" as a poll-for-status UX rather than a blocking request.
	// context.Background() is deliberate here, not an oversight: the
	// tracking goroutine must outlive this request's own context, which is
	// cancelled the moment the HTTP response is written.
	go s.trackRun(context.Background(), run) //nolint:contextcheck,gosec // deliberately detached from the request context, see comment above

	writeJSON(w, http.StatusAccepted, map[string]string{"id": runID})
}

func (s *server) trackRun(ctx context.Context, run Run) {
	ctx, cancel := context.WithTimeout(ctx, jobPollTimeout)
	defer cancel()

	ticker := time.NewTicker(jobPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			run.Status = RunStatusFailed
			run.Error = "timed out waiting for benchmark job to finish"
			s.finishRun(run)
			return
		case <-ticker.C:
			job, err := s.kube.BatchV1().Jobs(run.Namespace).Get(ctx, run.JobName, metav1.GetOptions{})
			if err != nil {
				log.Printf("run %s: get job %s: %v", run.ID, run.JobName, err)
				continue
			}
			if job.Status.Succeeded > 0 {
				s.completeRun(ctx, run, job)
				return
			}
			if job.Status.Failed > 0 {
				run.Status = RunStatusFailed
				run.Error = "benchmark job failed — see kubectl logs for the job's pod"
				s.finishRun(run)
				return
			}
		}
	}
}

func (s *server) completeRun(ctx context.Context, run Run, job *batchv1.Job) {
	output, err := s.readJobLogs(ctx, job)
	if err != nil {
		run.Status = RunStatusFailed
		run.Error = "job succeeded but reading its logs failed: " + err.Error()
		s.finishRun(run)
		return
	}
	run.Status = RunStatusSucceeded
	run.Result = parseSysbenchOutput(output)
	s.finishRun(run)
}

func (s *server) finishRun(run Run) {
	now := time.Now()
	run.FinishedAt = &now
	if err := s.store.Save(run); err != nil {
		log.Printf("run %s: save final state: %v", run.ID, err)
	}
}

// readJobLogs finds the (single, RestartPolicy: Never) pod the Job created
// and reads its combined stdout/stderr, which is where sysbench's
// human-readable report lands — sysbench has no --output=json mode to
// unmarshal instead (see parser.go).
func (s *server) readJobLogs(ctx context.Context, job *batchv1.Job) (string, error) {
	pods, err := s.kube.CoreV1().Pods(job.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "job-name=" + job.Name,
	})
	if err != nil {
		return "", fmt.Errorf("list pods for job %s: %w", job.Name, err)
	}
	if len(pods.Items) == 0 {
		return "", errors.New("no pods found for completed job")
	}
	pod := pods.Items[0]

	stream, err := s.kube.CoreV1().Pods(job.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream(ctx) //nolint:exhaustruct // only Container matters and the Job has a single container
	if err != nil {
		return "", fmt.Errorf("stream logs for pod %s: %w", pod.Name, err)
	}
	defer func() { _ = stream.Close() }()

	data, err := io.ReadAll(stream)
	if err != nil {
		return "", fmt.Errorf("read logs for pod %s: %w", pod.Name, err)
	}
	return string(data), nil
}

func (s *server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, ok := s.store.Get(id)
	if !ok {
		apiError(w, http.StatusNotFound, "run not found: "+id)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *server) handleListRuns(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.List())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON error: %v", err)
	}
}

func apiError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
