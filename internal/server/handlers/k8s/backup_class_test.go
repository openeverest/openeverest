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

package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestFilterBackupClasses(t *testing.T) {
	t.Parallel()

	classes := []backupv1alpha1.BackupClass{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pxc-provider"},
			Spec: backupv1alpha1.BackupClassSpec{
				ExecutionMode:      backupv1alpha1.BackupExecutionModeProviderManaged,
				SupportedProviders: []string{"pxc"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pxc-job"},
			Spec: backupv1alpha1.BackupClassSpec{
				ExecutionMode:      backupv1alpha1.BackupExecutionModeJob,
				SupportedProviders: []string{"pxc"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "unrestricted-job"},
			Spec: backupv1alpha1.BackupClassSpec{
				ExecutionMode: backupv1alpha1.BackupExecutionModeJob,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "psmdb-provider"},
			Spec: backupv1alpha1.BackupClassSpec{
				ExecutionMode:      backupv1alpha1.BackupExecutionModeProviderManaged,
				SupportedProviders: []string{"psmdb"},
			},
		},
	}

	tests := []struct {
		name   string
		input  []backupv1alpha1.BackupClass
		params *api.ListBackupClassesParams
		want   []string
	}{
		{
			name:  "provider includes all execution modes and unrestricted classes",
			input: classes,
			params: &api.ListBackupClassesParams{
				Provider: new("pxc"),
			},
			want: []string{"pxc-provider", "pxc-job", "unrestricted-job"},
		},
		{
			name:  "execution mode",
			input: classes,
			params: &api.ListBackupClassesParams{
				ExecutionMode: ptr.To(api.ListBackupClassesParamsExecutionMode(backupv1alpha1.BackupExecutionModeProviderManaged)),
			},
			want: []string{"pxc-provider", "psmdb-provider"},
		},
		{
			name:  "provider and execution mode",
			input: classes,
			params: &api.ListBackupClassesParams{
				Provider:      new("pxc"),
				ExecutionMode: ptr.To(api.ListBackupClassesParamsExecutionMode(backupv1alpha1.BackupExecutionModeProviderManaged)),
			},
			want: []string{"pxc-provider"},
		},
		{
			name:  "empty input",
			input: nil,
			params: &api.ListBackupClassesParams{
				Provider: new("pxc"),
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filtered := filterBackupClasses(tt.input, tt.params)
			if len(filtered) != len(tt.want) {
				t.Fatalf("got %d classes, want %d", len(filtered), len(tt.want))
			}
			for i, name := range tt.want {
				if filtered[i].Name != name {
					t.Errorf("class %d = %q, want %q", i, filtered[i].Name, name)
				}
			}
		})
	}
}

func TestListBackupClassesWithoutFilters(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	backupClasses := []backupv1alpha1.BackupClass{
		{ObjectMeta: metav1.ObjectMeta{Name: "provider-managed"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "job"}},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&backupClasses[0],
		&backupClasses[1],
	).Build()
	handler := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "").WithKubernetesClient(fakeClient),
		log:           zap.NewNop().Sugar(),
	}

	list, err := handler.ListBackupClasses(context.Background(), "main", nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, backupClasses, list.Items)
}
