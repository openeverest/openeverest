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
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestCreateRestore_Validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "test-instance", Namespace: namespace},
		Spec:       corev1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: "test-provider"}},
	}
	// Point-in-time recovery resolves its read class from the Instance owning
	// the stream, so these carry backup config the plain fixture does not.
	newStreamInstance := func(className string) *corev1alpha1.Instance {
		return &corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{Name: "test-instance", Namespace: namespace},
			Spec: corev1alpha1.InstanceSpec{
				ProviderRef: common.ObjectRef{Name: "test-provider"},
				Backup: &corev1alpha1.InstanceBackupSpec{
					Enabled:  true,
					ClassRef: common.ObjectRef{Name: className},
					Storages: []corev1alpha1.InstanceBackupStorage{{
						StorageRef: common.ObjectRef{Name: "s1"},
						PITR:       &corev1alpha1.InstanceBackupStoragePITR{Enabled: true},
					}},
				},
			},
		}
	}
	streamInstancePITR := newStreamInstance("pitr-class")
	streamInstanceNoPITR := newStreamInstance("no-pitr-class")
	pitrClass := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pitr-class"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:      backupv1alpha1.BackupExecutionModeProviderManaged,
			SupportedProviders: backupv1alpha1.ProviderNameList{"test-provider"},
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				SupportsPITR: true,
			},
		},
	}
	noPitrClass := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "no-pitr-class"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:      backupv1alpha1.BackupExecutionModeProviderManaged,
			SupportedProviders: backupv1alpha1.ProviderNameList{"test-provider"},
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				SupportsPITR: false,
			},
		},
	}
	unsupportedProviderClass := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "unsupported-provider-class"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:      backupv1alpha1.BackupExecutionModeProviderManaged,
			SupportedProviders: backupv1alpha1.ProviderNameList{"other-provider"},
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				SupportsPITR: true,
			},
		},
	}
	succeededBackup := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "succeeded-backup", Namespace: namespace},
		Spec:       backupv1alpha1.BackupSpec{ClassRef: common.ObjectRef{Name: "pitr-class"}},
		Status:     backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded},
	}
	succeededBackupNoPitr := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "succeeded-backup-no-pitr", Namespace: namespace},
		Spec:       backupv1alpha1.BackupSpec{ClassRef: common.ObjectRef{Name: "no-pitr-class"}},
		Status:     backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded},
	}
	succeededBackupUnsupportedProvider := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "succeeded-backup-unsupported-provider", Namespace: namespace},
		Spec:       backupv1alpha1.BackupSpec{ClassRef: common.ObjectRef{Name: "unsupported-provider-class"}},
		Status:     backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded},
	}
	runningBackup := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "running-backup", Namespace: namespace},
		Spec:       backupv1alpha1.BackupSpec{ClassRef: common.ObjectRef{Name: "pitr-class"}},
		Status:     backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateRunning},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))

	newRestore := func(backupName string) *backupv1alpha1.Restore {
		return &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: "test-restore", Namespace: namespace},
			Spec: backupv1alpha1.RestoreSpec{
				InstanceRef: common.ObjectRef{Name: "test-instance"},
				DataSource: backupv1alpha1.DataSource{
					Type: backupv1alpha1.DataSourceTypeBackup,
					Backup: &backupv1alpha1.DataSourceBackup{
						BackupRef: common.ObjectRef{Name: backupName},
					},
				},
			},
		}
	}

	newPITRRestore := func(target backupv1alpha1.RecoveryTarget, storage string) *backupv1alpha1.Restore {
		return &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: "test-restore", Namespace: namespace},
			Spec: backupv1alpha1.RestoreSpec{
				InstanceRef: common.ObjectRef{Name: "test-instance"},
				DataSource: backupv1alpha1.DataSource{
					Type: backupv1alpha1.DataSourceTypePointInTime,
					PointInTime: &backupv1alpha1.DataSourcePointInTime{
						Source:         backupv1alpha1.StreamSource{StorageRef: common.ObjectRef{Name: storage}},
						RecoveryTarget: target,
					},
				},
			},
		}
	}

	errServerUnavailable := errors.New("server unavailable")

	tests := []struct {
		name               string
		restore            *backupv1alpha1.Restore
		objects            []ctrlclient.Object
		err                string
		wantSentinel       error  // if set, err must also satisfy errors.Is(err, wantSentinel)
		wantInvalidRequest bool   // true for validation failures (ErrInvalidRequest); false for infrastructure/runtime errors
		getErrName         string // if set, a Get for an object with this name fails with getErr instead of hitting the fake client
		getErr             error
	}{
		{
			name:               "missing backup fails",
			restore:            newRestore("missing-backup"),
			objects:            nil,
			err:                "backup not found: 'missing-backup' in namespace 'test-namespace'",
			wantSentinel:       controller.ErrBackupNotFound,
			wantInvalidRequest: true,
		},
		{
			name:               "backup not succeeded fails",
			restore:            newRestore("running-backup"),
			objects:            []ctrlclient.Object{runningBackup},
			err:                "backup not succeeded: 'running-backup' is in state 'Running'",
			wantSentinel:       controller.ErrBackupNotSucceeded,
			wantInvalidRequest: true,
		},
		{
			name:               "backup lookup server error fails",
			restore:            newRestore("server-error-backup"),
			objects:            nil,
			err:                "failed to get backup 'server-error-backup'",
			wantInvalidRequest: false,
			getErrName:         "server-error-backup",
			getErr:             errServerUnavailable,
		},
		{
			name:    "succeeded backup, no PITR, passes",
			restore: newRestore("succeeded-backup-no-pitr"),
			objects: []ctrlclient.Object{succeededBackupNoPitr, noPitrClass, instance},
			err:     "",
		},
		{
			name:               "missing instance fails",
			restore:            newRestore("succeeded-backup-no-pitr"),
			objects:            []ctrlclient.Object{succeededBackupNoPitr, noPitrClass},
			err:                "instance 'test-instance' does not exist",
			wantSentinel:       controller.ErrInstanceNotFound,
			wantInvalidRequest: true,
		},
		{
			name:               "class does not support instance's provider fails",
			restore:            newRestore("succeeded-backup-unsupported-provider"),
			objects:            []ctrlclient.Object{succeededBackupUnsupportedProvider, unsupportedProviderClass, instance},
			err:                "provider not supported by backup class: class 'unsupported-provider-class', provider 'test-provider'",
			wantSentinel:       controller.ErrProviderUnsupported,
			wantInvalidRequest: true,
		},
		{
			name:    "succeeded backup passes regardless of the class's PITR support",
			restore: newRestore("succeeded-backup"),
			objects: []ctrlclient.Object{succeededBackup, pitrClass, instance},
			err:     "",
		},
		{
			name:               "point-in-time requested, class does not support PITR fails",
			restore:            newPITRRestore(backupv1alpha1.RecoveryTargetLatest, "s1"),
			objects:            []ctrlclient.Object{noPitrClass, streamInstanceNoPITR},
			err:                "point-in-time recovery is not supported",
			wantSentinel:       controller.ErrRestorePITRUnsupported,
			wantInvalidRequest: true,
		},
		{
			name:               "point-in-time date target without date fails",
			restore:            newPITRRestore(backupv1alpha1.RecoveryTargetDate, "s1"),
			objects:            []ctrlclient.Object{pitrClass, streamInstancePITR},
			err:                "point-in-time recovery date is required",
			wantSentinel:       controller.ErrRestorePITRDateRequired,
			wantInvalidRequest: true,
		},
		{
			name:    "point-in-time requested, class supports PITR passes",
			restore: newPITRRestore(backupv1alpha1.RecoveryTargetLatest, "s1"),
			objects: []ctrlclient.Object{pitrClass, streamInstancePITR},
			err:     "",
		},
		{
			name:               "point-in-time on a storage without PITR enabled fails",
			restore:            newPITRRestore(backupv1alpha1.RecoveryTargetLatest, "not-archiving"),
			objects:            []ctrlclient.Object{pitrClass, streamInstancePITR},
			err:                "point-in-time recovery is not supported",
			wantSentinel:       controller.ErrRestorePITRUnsupported,
			wantInvalidRequest: true,
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
			builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...)
			if tt.getErr != nil {
				builder = builder.WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, c ctrlclient.WithWatch, key ctrlclient.ObjectKey, obj ctrlclient.Object, opts ...ctrlclient.GetOption) error {
						if key.Name == tt.getErrName {
							return tt.getErr
						}
						return c.Get(ctx, key, obj, opts...)
					},
				})
			}
			fakeClient := builder.Build()
			kubeConnector := kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient)

			mockNext := &handlers.MockHandler{}
			mockNext.On("CreateRestore", mock.Anything, mock.Anything).Return(tt.restore, nil)

			handler := &validateHandler{
				log:           zap.NewNop().Sugar(),
				kubeConnector: kubeConnector,
				next:          mockNext,
			}

			_, err := handler.CreateRestore(ctx, tt.restore)
			if tt.err == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.err)
			if tt.wantInvalidRequest {
				require.ErrorIs(t, err, ErrInvalidRequest)
			} else {
				require.NotErrorIs(t, err, ErrInvalidRequest)
			}
			if tt.wantSentinel != nil {
				require.ErrorIs(t, err, tt.wantSentinel)
			}
			if tt.getErr != nil {
				require.ErrorIs(t, err, tt.getErr)
			}
			mockNext.AssertNotCalled(t, "CreateRestore", mock.Anything, mock.Anything)
		})
	}
}
