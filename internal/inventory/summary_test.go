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

package inventory

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

func instance(ns, name, provider string, conditions ...metav1.Condition) corev1alpha1.Instance {
	return corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec:       corev1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: provider}},
		Status:     corev1alpha1.InstanceStatus{Conditions: conditions},
	}
}

func TestSummarize(t *testing.T) {
	t.Parallel()

	ready := metav1.Condition{Type: corev1alpha1.ConditionReady, Status: metav1.ConditionTrue}
	stalled := metav1.Condition{Type: corev1alpha1.ConditionStalled, Status: metav1.ConditionTrue}
	withResources := instance("team-a", "db1", "percona-server-mongodb", ready)
	withResources.Spec.Components = map[string]corev1alpha1.ComponentSpec{
		"engine": {
			Replicas: ptr.To[int32](3),
			Resources: &corev1.ResourceRequirements{Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			}},
			Storage: &corev1alpha1.Storage{Size: resource.MustParse("10Gi")},
		},
		"proxy": {Resources: &corev1.ResourceRequirements{Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("500m"),
		}}},
	}
	withResources.Spec.Backup = &corev1alpha1.InstanceBackupSpec{Storages: []corev1alpha1.InstanceBackupStorage{{}}}
	withResources.Status.Backup = &corev1alpha1.InstanceBackupStatus{Storages: []corev1alpha1.InstanceBackupStorageStatus{
		{Name: "store", PITR: &corev1alpha1.InstanceBackupStoragePITRStatus{}},
	}}
	now := time.Unix(1_700_000_000, 0)

	snap := &Snapshot{
		Providers: []corev1alpha1.Provider{
			{ObjectMeta: metav1.ObjectMeta{Name: "percona-server-mongodb"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "acme-internal-db"}},
		},
		Plugins: []extensionsv1alpha1.Plugin{
			{ObjectMeta: metav1.ObjectMeta{Name: "plugin-audit"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "acme-dashboard"}},
		},
		Instances: []corev1alpha1.Instance{
			withResources,
			instance("team-a", "db2", "acme-internal-db", stalled),
			instance("team-b", "db3", "acme-internal-db"),
		},
		Backups: []backupv1alpha1.Backup{
			{Status: backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded}},
			{Status: backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateFailed}},
			{Status: backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateError}},
			{Status: backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateRunning}},
		},
		Restores: []backupv1alpha1.Restore{{}},
	}

	got := Summarize(snap, "cluster-1", now)

	assert.Equal(t, Summary{
		SchemaVersion:              SummarySchemaVersion,
		ClusterID:                  "cluster-1",
		CollectedAt:                now,
		ProviderInstallations:      2,
		Providers:                  []string{"percona-server-mongodb"},
		InstancesByProvider:        map[string]int{"percona-server-mongodb": 1, CustomProvider: 2},
		PluginInstallations:        2,
		Plugins:                    []string{"plugin-audit"},
		Instances:                  3,
		InstancesReady:             1,
		InstancesStalled:           1,
		NamespacesWithInstances:    2,
		CPURequestedCores:          3.5,
		MemoryRequestedBytes:       6 * 1024 * 1024 * 1024,
		StorageRequestedBytes:      30 * 1024 * 1024 * 1024,
		BackupsSucceeded:           1,
		BackupsFailed:              2,
		Restores:                   1,
		InstancesWithBackupStorage: 1,
		InstancesWithPITR:          1,
	}, got,
	)
}

func TestMerge(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	main := Summary{
		ClusterID: "main", CollectedAt: now,
		Providers: []string{"percona-server-mongodb"}, ProviderInstallations: 1,
		InstancesByProvider: map[string]int{"percona-server-mongodb": 2},
		Instances:           2, CPURequestedCores: 1.5,
	}
	agentOld := Summary{
		ClusterID: "agent-1", CollectedAt: now.Add(-2 * time.Hour),
		Instances: 100,
	}
	agentNew := Summary{
		ClusterID: "agent-1", CollectedAt: now.Add(-time.Hour),
		Providers: []string{"percona-server-mongodb", "valkey"}, ProviderInstallations: 2,
		InstancesByProvider: map[string]int{"percona-server-mongodb": 1, "valkey": 3, CustomProvider: 1},
		Instances:           5, CPURequestedCores: 2,
	}
	stale := Summary{ClusterID: "agent-2", CollectedAt: now.Add(-72 * time.Hour), Instances: 7}

	got := Merge([]Summary{agentOld, main, stale, agentNew}, now, 48*time.Hour)

	assert.Equal(t, 2, got.Clusters, "main and agent-1 count once each")
	assert.Equal(t, 1, got.StaleClusters)
	assert.Equal(t, 7, got.Instances, "agent-1 counts with its newest summary, the stale one is left out")
	assert.Equal(t, 3, got.ProviderInstallations)
	assert.Equal(t, []string{"percona-server-mongodb", "valkey"}, got.Providers, "names are unioned")
	assert.Equal(t, map[string]int{"percona-server-mongodb": 3, "valkey": 3, CustomProvider: 1}, got.InstancesByProvider)
	assert.InDelta(t, 3.5, got.CPURequestedCores, 0)
}

func TestSummaryFromAnOlderAgent(t *testing.T) {
	t.Parallel()

	// An agent built before backups were counted doesn't send those fields.
	const olderDocument = `{
		"schemaVersion": 1,
		"clusterId": "agent-1",
		"collectedAt": "2023-11-14T22:13:20Z",
		"instances": 4,
		"futureField": "ignored"
	}`
	var older Summary
	require.NoError(t, json.Unmarshal([]byte(olderDocument), &older))

	got := Merge([]Summary{older}, older.CollectedAt, time.Hour)

	assert.Equal(t, 1, got.Clusters)
	assert.Equal(t, 4, got.Instances)
	assert.Zero(t, got.BackupsSucceeded)
}
