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

package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestCreateBackup_Validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	cluster := "test-cluster"

	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "test-instance", Namespace: namespace},
		Spec:       corev1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: "test-provider"}},
	}
	storage := &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: "test-storage", Namespace: namespace},
	}
	supportedClass := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "supported-class"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:      backupv1alpha1.BackupExecutionModeJob,
			SupportedProviders: backupv1alpha1.ProviderNameList{"test-provider"},
		},
	}
	unsupportedClass := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "unsupported-class"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:      backupv1alpha1.BackupExecutionModeJob,
			SupportedProviders: backupv1alpha1.ProviderNameList{"other-provider"},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))

	tests := []struct {
		name    string
		backup  *backupv1alpha1.Backup
		objects []ctrlclient.Object
		err     string
	}{
		{
			name: "instance not found fails",
			backup: &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "b1", Namespace: namespace},
				Spec: backupv1alpha1.BackupSpec{
					InstanceRef: common.ObjectRef{Name: "missing-instance"},
					StorageRef:  common.ObjectRef{Name: "test-storage"},
					ClassRef:    common.ObjectRef{Name: "supported-class"},
				},
			},
			objects: []ctrlclient.Object{storage, supportedClass},
			err:     "instance 'missing-instance' does not exist",
		},
		{
			name: "storage not found fails",
			backup: &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "b2", Namespace: namespace},
				Spec: backupv1alpha1.BackupSpec{
					InstanceRef: common.ObjectRef{Name: "test-instance"},
					StorageRef:  common.ObjectRef{Name: "missing-storage"},
					ClassRef:    common.ObjectRef{Name: "supported-class"},
				},
			},
			objects: []ctrlclient.Object{instance, supportedClass},
			err:     "backup storage 'missing-storage' does not exist",
		},
		{
			name: "class not found fails",
			backup: &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "b3", Namespace: namespace},
				Spec: backupv1alpha1.BackupSpec{
					InstanceRef: common.ObjectRef{Name: "test-instance"},
					StorageRef:  common.ObjectRef{Name: "test-storage"},
					ClassRef:    common.ObjectRef{Name: "missing-class"},
				},
			},
			objects: []ctrlclient.Object{instance, storage},
			err:     "backup class 'missing-class' does not exist",
		},
		{
			name: "class not supporting provider fails",
			backup: &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "b4", Namespace: namespace},
				Spec: backupv1alpha1.BackupSpec{
					InstanceRef: common.ObjectRef{Name: "test-instance"},
					StorageRef:  common.ObjectRef{Name: "test-storage"},
					ClassRef:    common.ObjectRef{Name: "unsupported-class"},
				},
			},
			objects: []ctrlclient.Object{instance, storage, unsupportedClass},
			err:     "backup class 'unsupported-class' does not support provider 'test-provider'",
		},
		{
			name: "valid refs pass",
			backup: &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "b5", Namespace: namespace},
				Spec: backupv1alpha1.BackupSpec{
					InstanceRef: common.ObjectRef{Name: "test-instance"},
					StorageRef:  common.ObjectRef{Name: "test-storage"},
					ClassRef:    common.ObjectRef{Name: "supported-class"},
				},
			},
			objects: []ctrlclient.Object{instance, storage, supportedClass},
			err:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Use DeepCopy to avoid race conditions since the fake client
			// modifies the objects' ResourceVersion during Build().
			objs := make([]ctrlclient.Object, len(tt.objects))
			for i, obj := range tt.objects {
				objs[i] = obj.DeepCopyObject().(ctrlclient.Object) //nolint:forcetypeassert
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			kubeConnector := kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient)

			mockNext := &handlers.MockHandler{}
			mockNext.On("CreateBackup", mock.Anything, mock.Anything, mock.Anything).Return(tt.backup, nil)

			handler := &validateHandler{
				log:           zap.NewNop().Sugar(),
				kubeConnector: kubeConnector,
				next:          mockNext,
			}

			_, err := handler.CreateBackup(ctx, cluster, tt.backup)
			if tt.err == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.err)
		})
	}
}
