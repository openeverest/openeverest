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

func TestCreateBackup_Validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	cluster := "test-cluster"

	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "test-instance", Namespace: namespace},
		Spec:       corev1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: "test-provider"}},
	}
	supportedClass := judgedClass("supported-class", "test-provider", backupv1alpha1.CompatibilityCompatible, "")
	unsupportedClass := judgedClass("unsupported-class", "other-provider", backupv1alpha1.CompatibilityCompatible, "")
	driftClass := judgedClass("drift-class", "test-provider", backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonSpecDrift)
	pendingClass := judgedClass("pending-class", "test-provider", backupv1alpha1.CompatibilityPending, backupv1alpha1.ReasonClassNotAccepted)
	storage := judgedStorage("test-storage", namespace,
		storageVerdict{class: supportedClass, compatibility: backupv1alpha1.CompatibilityCompatible},
		storageVerdict{class: unsupportedClass, compatibility: backupv1alpha1.CompatibilityCompatible},
		storageVerdict{class: driftClass, compatibility: backupv1alpha1.CompatibilityCompatible},
	)
	gcsStorage := judgedStorage("gcs-storage", namespace,
		storageVerdict{class: supportedClass, compatibility: backupv1alpha1.CompatibilityIncompatible, reason: backupv1alpha1.ReasonParameterUnsupported},
	)
	newBackup := func(name, class, storage string) *backupv1alpha1.Backup {
		return &backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: backupv1alpha1.BackupSpec{
				Origin: backupv1alpha1.BackupOrigin{
					Type:        backupv1alpha1.BackupOriginTypeInstance,
					InstanceRef: &common.ObjectRef{Name: "test-instance"},
				},
				StorageRef: common.ObjectRef{Name: storage},
				ClassRef:   common.ObjectRef{Name: class},
			},
		}
	}

	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))

	errServerUnavailable := errors.New("server unavailable")

	tests := []struct {
		name               string
		backup             *backupv1alpha1.Backup
		objects            []ctrlclient.Object
		err                string
		wantSentinel       error  // if set, err must also satisfy errors.Is(err, wantSentinel)
		wantInvalidRequest bool   // true for validation failures (ErrInvalidRequest); false for infrastructure/runtime errors
		getErrName         string // if set, a Get for an object with this name fails with getErr instead of hitting the fake client
		getErr             error
		wantWarnings       []string // substrings of the warnings an admitted Backup is created with
	}{
		{
			name: "instance not found fails",
			backup: &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "b1", Namespace: namespace},
				Spec: backupv1alpha1.BackupSpec{
					Origin: backupv1alpha1.BackupOrigin{
						Type:        backupv1alpha1.BackupOriginTypeInstance,
						InstanceRef: &common.ObjectRef{Name: "missing-instance"},
					},
					StorageRef: common.ObjectRef{Name: "test-storage"},
					ClassRef:   common.ObjectRef{Name: "supported-class"},
				},
			},
			objects:            []ctrlclient.Object{storage, supportedClass},
			err:                "instance 'missing-instance' does not exist",
			wantSentinel:       controller.ErrInstanceNotFound,
			wantInvalidRequest: true,
		},
		{
			name: "storage not found fails",
			backup: &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "b2", Namespace: namespace},
				Spec: backupv1alpha1.BackupSpec{
					Origin: backupv1alpha1.BackupOrigin{
						Type:        backupv1alpha1.BackupOriginTypeInstance,
						InstanceRef: &common.ObjectRef{Name: "test-instance"},
					},
					StorageRef: common.ObjectRef{Name: "missing-storage"},
					ClassRef:   common.ObjectRef{Name: "supported-class"},
				},
			},
			objects:            []ctrlclient.Object{instance, supportedClass},
			err:                "backup storage 'missing-storage' does not exist",
			wantSentinel:       controller.ErrBackupStorageNotFound,
			wantInvalidRequest: true,
		},
		{
			name: "class not found fails",
			backup: &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "b3", Namespace: namespace},
				Spec: backupv1alpha1.BackupSpec{
					Origin: backupv1alpha1.BackupOrigin{
						Type:        backupv1alpha1.BackupOriginTypeInstance,
						InstanceRef: &common.ObjectRef{Name: "test-instance"},
					},
					StorageRef: common.ObjectRef{Name: "test-storage"},
					ClassRef:   common.ObjectRef{Name: "missing-class"},
				},
			},
			objects:            []ctrlclient.Object{instance, storage},
			err:                "backup class not found: 'missing-class'",
			wantSentinel:       controller.ErrBackupClassNotFound,
			wantInvalidRequest: true,
		},
		{
			name:               "class not run by the instance's provider fails",
			backup:             newBackup("b4", "unsupported-class", "test-storage"),
			objects:            []ctrlclient.Object{instance, storage, unsupportedClass, s3StorageClass},
			err:                `backup class incompatible: BackupClass "unsupported-class" is Incompatible with Provider "test-provider" (ProviderUnsupported)`,
			wantSentinel:       controller.ErrBackupClassIncompatible,
			wantInvalidRequest: true,
		},
		{
			name:               "class its provider rejects fails",
			backup:             newBackup("b7", "drift-class", "test-storage"),
			objects:            []ctrlclient.Object{instance, storage, driftClass, s3StorageClass},
			err:                `BackupClass "drift-class" is Incompatible with Provider "test-provider" (SpecDrift)`,
			wantSentinel:       controller.ErrBackupClassIncompatible,
			wantInvalidRequest: true,
		},
		{
			name:               "storage the class cannot write to fails",
			backup:             newBackup("b8", "supported-class", "gcs-storage"),
			objects:            []ctrlclient.Object{instance, gcsStorage, supportedClass, s3StorageClass},
			err:                `backup storage incompatible: BackupStorage "gcs-storage" is Incompatible with BackupClass "supported-class" (ParameterUnsupported)`,
			wantSentinel:       controller.ErrBackupStorageIncompatible,
			wantInvalidRequest: true,
		},
		{
			name:    "class not claimed yet passes with warnings",
			backup:  newBackup("b9", "pending-class", "test-storage"),
			objects: []ctrlclient.Object{instance, storage, pendingClass, s3StorageClass},
			wantWarnings: []string{
				`BackupClass "pending-class" is Pending for Provider "test-provider" (ClassNotAccepted)`,
				`BackupStorage "test-storage" is Pending for BackupClass "pending-class" (VerdictPending)`,
			},
		},
		{
			name: "instance lookup server error fails",
			backup: &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "b6", Namespace: namespace},
				Spec: backupv1alpha1.BackupSpec{
					Origin: backupv1alpha1.BackupOrigin{
						Type:        backupv1alpha1.BackupOriginTypeInstance,
						InstanceRef: &common.ObjectRef{Name: "server-error-instance"},
					},
					StorageRef: common.ObjectRef{Name: "test-storage"},
					ClassRef:   common.ObjectRef{Name: "supported-class"},
				},
			},
			objects:            []ctrlclient.Object{storage, supportedClass},
			err:                "failed to get instance 'server-error-instance'",
			wantInvalidRequest: false,
			getErrName:         "server-error-instance",
			getErr:             errServerUnavailable,
		},
		{
			name: "valid refs pass",
			backup: &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "b5", Namespace: namespace},
				Spec: backupv1alpha1.BackupSpec{
					Origin: backupv1alpha1.BackupOrigin{
						Type:        backupv1alpha1.BackupOriginTypeInstance,
						InstanceRef: &common.ObjectRef{Name: "test-instance"},
					},
					StorageRef: common.ObjectRef{Name: "test-storage"},
					ClassRef:   common.ObjectRef{Name: "supported-class"},
				},
			},
			objects: []ctrlclient.Object{instance, storage, supportedClass, s3StorageClass},
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
			mockNext.On("CreateBackup", mock.Anything, mock.Anything, mock.Anything).Return(tt.backup, nil)

			handler := &validateHandler{
				log:           zap.NewNop().Sugar(),
				kubeConnector: kubeConnector,
				next:          mockNext,
			}

			ctx, warnings := kubernetes.WithWarningCollector(ctx)
			_, err := handler.CreateBackup(ctx, cluster, tt.backup)
			if tt.err == "" {
				require.NoError(t, err)
				assertWarnings(t, tt.wantWarnings, warnings.Warnings())
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
			mockNext.AssertNotCalled(t, "CreateBackup", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
