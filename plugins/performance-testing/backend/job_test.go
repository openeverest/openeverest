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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestBuildJob_UnknownWorkload_Errors(t *testing.T) {
	t.Parallel()

	_, err := buildJob("run-1", "pg-poc", "everest", "postgres", Workload("not-a-real-workload"))
	require.Error(t, err)
}

func TestBuildJob_WiresCredentialsFromInstanceSecret_NotLiteralValues(t *testing.T) {
	t.Parallel()

	job, err := buildJob("run-1", "pg-poc", "everest", "postgres", WorkloadSmoke)
	require.NoError(t, err)

	assert.Equal(t, "perf-test-run-1", job.Name)
	assert.Equal(t, "everest", job.Namespace)
	assert.Equal(t, sysbenchImage, job.Spec.Template.Spec.Containers[0].Image)

	env := job.Spec.Template.Spec.Containers[0].Env
	require.Len(t, env, 5)
	for _, e := range env[:4] {
		require.NotNil(t, e.ValueFrom, "credential env var %q must be sourced via secretKeyRef, never a literal Value", e.Name)
		assert.Equal(t, "everest-secrets-pg-poc", e.ValueFrom.SecretKeyRef.Name)
		assert.Empty(t, e.Value, "credential env var %q must not carry a literal value", e.Name)
	}
	assert.Equal(t, "SB_PGDB", env[4].Name)
	assert.Equal(t, "postgres", env[4].Value)
}

func TestBuildJob_AntiAffinityTargetsInstanceLabel(t *testing.T) {
	t.Parallel()

	job, err := buildJob("run-1", "pg-poc", "everest", "postgres", WorkloadSmoke)
	require.NoError(t, err)

	terms := job.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	require.Len(t, terms, 1)
	assert.Equal(t, "pg-poc", terms[0].PodAffinityTerm.LabelSelector.MatchLabels["app.kubernetes.io/instance"])
	assert.Equal(t, "kubernetes.io/hostname", terms[0].PodAffinityTerm.TopologyKey)
}

func TestBuildJob_WorkloadSelectsCorrectSysbenchTest(t *testing.T) {
	t.Parallel()

	cases := map[Workload]string{
		WorkloadSmoke:      "oltp_read_only",
		WorkloadReadHeavy:  "oltp_read_only",
		WorkloadWriteHeavy: "oltp_write_only",
		WorkloadMixedOLTP:  "oltp_read_write",
	}
	for workload, wantTest := range cases {
		job, err := buildJob("run-1", "pg-poc", "everest", "postgres", workload)
		require.NoError(t, err)
		script := job.Spec.Template.Spec.Containers[0].Command[2]
		assert.Contains(t, script, "sysbench "+wantTest+" ",
			"workload %q: expected script to run %q, got:\n%s", workload, wantTest, script)
	}
}

func TestBuildJob_FailFastOnce(t *testing.T) {
	t.Parallel()

	job, err := buildJob("run-1", "pg-poc", "everest", "postgres", WorkloadSmoke)
	require.NoError(t, err)
	require.NotNil(t, job.Spec.BackoffLimit)
	assert.Equal(t, int32(0), *job.Spec.BackoffLimit)
	assert.Equal(t, corev1.RestartPolicyNever, job.Spec.Template.Spec.RestartPolicy)
}
