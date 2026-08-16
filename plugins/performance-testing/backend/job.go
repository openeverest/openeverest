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
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// sysbenchImage is built from ../Dockerfile.sysbench (debian:trixie-slim's
// packaged sysbench 1.0.20). The obvious first choice,
// severalnines/sysbench:latest, was tried and rejected: its bundled libpq
// predates PostgreSQL's SCRAM-SHA-256 default and fails against pg-poc
// with "SCRAM authentication requires libpq version 10 or above". This
// image was hand-verified against the same live cluster (a real
// prepare/cleanup round-trip succeeded) before being trusted, the same way
// the MinIO provider's image tag was hand-verified rather than assumed.
const sysbenchImage = "localhost/performance-testing-sysbench:0.1.0"

// credentialsSecretName is the naming convention observed on the real
// everest-poc cluster's DatabaseCluster credentials Secret
// ("everest-secrets-<cluster-name>"). This convention does not appear
// anywhere in the current release-2.0 source tree — there is no generic,
// API-brokered way to obtain DatabaseCluster credentials (the connection
// broker at GET .../instances/{instance}/connection only covers the newer
// spec-001 Instance type, which has zero real database engines running on
// it in the current release). Reading this Secret directly, scoped by the
// plugin's own RBAC to only this one key pattern, mirrors how
// internal/controller/backup/backup_controller.go already reads a
// connection Secret directly for its own Job's env — this is not a novel
// credential-handling pattern, it fills the same gap the same way for a
// resource type the existing broker doesn't cover.
func credentialsSecretName(instance string) string {
	return "everest-secrets-" + instance
}

// sysbench test names, extracted to constants since oltp_read_only backs
// two different workload profiles (smoke and read-heavy differ only in
// concurrency/duration, not which test runs).
const (
	sysbenchOLTPReadOnly  = "oltp_read_only"
	sysbenchOLTPWriteOnly = "oltp_write_only" //nolint:gosec // sysbench test name, not a credential
	sysbenchOLTPReadWrite = "oltp_read_write"
)

// workloadProfile maps a named Workload onto sysbench parameters. Hiding
// raw sysbench flags behind these names is the "Benchmark Workload
// Profiles" idea multiple applicants proposed publicly on #2464 — Postgres
// is the only engine this PoC exercises, so only the pgsql-relevant knobs
// are modelled.
type workloadProfile struct {
	sysbenchTest string
	threads      int
	durationSecs int
}

//nolint:gochecknoglobals // workloadProfiles is a fixed lookup table, not mutable shared state; same pattern as provider.go's other package-level maps in this repo.
var workloadProfiles = map[Workload]workloadProfile{
	WorkloadSmoke:      {sysbenchTest: sysbenchOLTPReadOnly, threads: 2, durationSecs: 10},
	WorkloadReadHeavy:  {sysbenchTest: sysbenchOLTPReadOnly, threads: 8, durationSecs: 30},
	WorkloadWriteHeavy: {sysbenchTest: sysbenchOLTPWriteOnly, threads: 8, durationSecs: 30},
	WorkloadMixedOLTP:  {sysbenchTest: sysbenchOLTPReadWrite, threads: 8, durationSecs: 30},
}

const (
	sysbenchTables    = 4
	sysbenchTableSize = 10000
	jobActiveDeadline = 600 // seconds; caps a hung benchmark, mirrors the BackoffLimit:0 "fail fast" precedent in backup_controller.go
)

// secretEnvVar builds an env var sourced from a key in the target
// DatabaseCluster's own credentials Secret. Values are injected by the
// kubelet at container start — this process never reads or holds the raw
// credential.
func secretEnvVar(name, secretName, key string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  key,
			},
		},
	}
}

// benchmarkScript renders the prepare/run/cleanup shell script. All
// interpolated values (test name, thread count, duration) come from the
// fixed workloadProfiles table, never from caller input, so this isn't
// building a shell command out of untrusted strings.
func benchmarkScript(profile workloadProfile) string {
	args := fmt.Sprintf(
		`--db-driver=pgsql --pgsql-host="$SB_PGHOST" --pgsql-port="$SB_PGPORT" --pgsql-user="$SB_PGUSER" --pgsql-password="$SB_PGPASSWORD" --pgsql-db="$SB_PGDB" --tables=%d --table-size=%d`,
		sysbenchTables, sysbenchTableSize,
	)
	return fmt.Sprintf(
		"set -e\n"+
			"sysbench %s %s prepare\n"+
			"sysbench %s %s --threads=%d --time=%d run\n"+
			"sysbench %s %s cleanup\n",
		profile.sysbenchTest, args,
		profile.sysbenchTest, args, profile.threads, profile.durationSecs,
		profile.sysbenchTest, args,
	)
}

// benchmarkEnv wires the container's SB_PG* env vars: credentials sourced
// from the target's own Secret via secretKeyRef, plus the plain-value
// database name.
func benchmarkEnv(secretName, dbName string) []corev1.EnvVar {
	return []corev1.EnvVar{
		secretEnvVar("SB_PGHOST", secretName, "pgbouncer-host"),
		secretEnvVar("SB_PGPORT", secretName, "pgbouncer-port"),
		secretEnvVar("SB_PGUSER", secretName, "user"),
		secretEnvVar("SB_PGPASSWORD", secretName, "password"),
		{Name: "SB_PGDB", Value: dbName},
	}
}

// antiAffinity prefers scheduling the benchmark Job away from the target
// database's own pods, answering the node-isolation open question raised
// on #2464 by Aryanbhargava18: the load generator shouldn't compete with
// the database it's measuring for the same node's CPU/memory ("observer
// effect"). Soft preference, not a hard requirement — a single-node dev
// cluster (like everest-poc) must still be able to run the Job at all.
func antiAffinity(instance string) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/instance": instance,
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}
}

// buildJob renders the benchmark Job.
func buildJob(runID, instance, namespace, dbName string, workload Workload) (*batchv1.Job, error) {
	profile, ok := workloadProfiles[workload]
	if !ok {
		return nil, fmt.Errorf("unknown workload %q", workload)
	}

	backoffLimit := int32(0)
	activeDeadline := int64(jobActiveDeadline)
	labels := map[string]string{
		"app.kubernetes.io/managed-by":                   "performance-testing-plugin",
		"performance-testing.plugins.openeverest.io/run": runID,
	}

	return &batchv1.Job{ //nolint:exhaustruct // TypeMeta and other unset fields use their zero values deliberately.
		ObjectMeta: metav1.ObjectMeta{
			Name:      "perf-test-" + runID,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          &backoffLimit,
			ActiveDeadlineSeconds: &activeDeadline,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Affinity:      antiAffinity(instance),
					Containers: []corev1.Container{
						{
							Name:    "sysbench",
							Image:   sysbenchImage,
							Command: []string{"sh", "-c", benchmarkScript(profile)},
							Env:     benchmarkEnv(credentialsSecretName(instance), dbName),
						},
					},
				},
			},
		},
	}, nil
}
