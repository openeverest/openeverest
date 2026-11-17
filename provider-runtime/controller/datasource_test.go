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

package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// seedingInstance returns an Instance seeded from a Backup that no longer
// exists, as after its source Instance was deleted with the Cascade policy.
func seedingInstance(conditions ...metav1.Condition) *v1alpha1.Instance {
	return &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "clone", Namespace: "ns"},
		Spec: v1alpha1.InstanceSpec{
			DataSource: &backupv1alpha1.DataSource{
				Type: backupv1alpha1.DataSourceTypeBackup,
				Backup: &backupv1alpha1.DataSourceBackup{
					BackupRef: common.ObjectRef{Name: "deleted-with-its-instance"},
				},
			},
		},
		Status: v1alpha1.InstanceStatus{Conditions: conditions},
	}
}

func seededCondition() metav1.Condition {
	return metav1.Condition{
		Type:    v1alpha1.ConditionDataSourceReady,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.ReasonDataSourceSucceeded,
		Message: `Instance seeded from Backup "deleted-with-its-instance"`,
	}
}

// seededStatus is what a seeded Instance reports, whatever the source's state.
func seededStatus() DataSourceStatus {
	return DataSourceStatus{
		Done:    true,
		State:   DataSourceStateSucceeded,
		Reason:  v1alpha1.ReasonDataSourceSucceeded,
		Message: seededCondition().Message,
	}
}

func newDataSourceTestContext(t *testing.T, in *v1alpha1.Instance) *Context {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(s))
	require.NoError(t, backupv1alpha1.AddToScheme(s))
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(in).Build()

	return NewContext(context.Background(), cl, in, "test-provider")
}

func TestReconcileDataSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		conditions []metav1.Condition
		expected   DataSourceStatus
	}{
		{
			name: "waits for a missing source until seeded",
			expected: DataSourceStatus{
				Done:    false,
				State:   DataSourceStateWaiting,
				Reason:  v1alpha1.ReasonDataSourceSourceBackupNotFound,
				Message: `source Backup "deleted-with-its-instance" not found in namespace "ns"`,
			},
		},
		{
			name:       "stays seeded after the source is deleted",
			conditions: []metav1.Condition{seededCondition()},
			expected:   seededStatus(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := newDataSourceTestContext(t, seedingInstance(tc.conditions...))

			got, err := c.ReconcileDataSource()
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
			assert.Equal(t, &tc.expected, c.GetDataSourceStatus())

			restores := &backupv1alpha1.RestoreList{}
			require.NoError(t, c.Client().List(context.Background(), restores))
			assert.Empty(t, restores.Items)
		})
	}
}

func TestSetDataSourceStatus(t *testing.T) {
	t.Parallel()

	waitingForCluster := DataSourceStatus{
		Done:    false,
		State:   DataSourceStateWaiting,
		Reason:  v1alpha1.ReasonDataSourceWaitingForCluster,
		Message: "waiting for the cluster to be Ready",
	}

	tests := []struct {
		name       string
		conditions []metav1.Condition
		expected   DataSourceStatus
	}{
		{
			name:     "stages the provider status until seeded",
			expected: waitingForCluster,
		},
		{
			name:       "keeps a seeded instance seeded",
			conditions: []metav1.Condition{seededCondition()},
			expected:   seededStatus(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := newDataSourceTestContext(t, seedingInstance(tc.conditions...))

			c.SetDataSourceStatus(waitingForCluster)
			assert.Equal(t, &tc.expected, c.GetDataSourceStatus())
		})
	}
}
