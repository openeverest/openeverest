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
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestCreateBackupImport_Validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))

	newImport := func(className, storageName string) *backupv1alpha1.BackupImport {
		return &backupv1alpha1.BackupImport{
			ObjectMeta: metav1.ObjectMeta{Name: "test-import", Namespace: "test-namespace"},
			Spec: backupv1alpha1.BackupImportSpec{
				ClassRef:   common.ObjectRef{Name: className},
				StorageRef: common.ObjectRef{Name: storageName},
			},
		}
	}

	importClass := judgedClass("import-class", "test-provider", backupv1alpha1.CompatibilityCompatible, "")
	noImportClass := judgedClass("no-import-class", "test-provider", backupv1alpha1.CompatibilityCompatible, "")
	noImportClass.Spec.SupportsImport = false
	driftClass := judgedClass("drift-class", "test-provider", backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonSpecDrift)
	pendingClass := judgedClass("pending-class", "test-provider", backupv1alpha1.CompatibilityPending, backupv1alpha1.ReasonClassNotAccepted)
	storage := judgedStorage("s1", "test-namespace",
		storageVerdict{class: importClass, compatibility: backupv1alpha1.CompatibilityCompatible},
		storageVerdict{class: noImportClass, compatibility: backupv1alpha1.CompatibilityCompatible},
		storageVerdict{class: driftClass, compatibility: backupv1alpha1.CompatibilityCompatible},
	)
	gcsStorage := judgedStorage("gcs", "test-namespace",
		storageVerdict{class: importClass, compatibility: backupv1alpha1.CompatibilityIncompatible, reason: backupv1alpha1.ReasonParameterUnsupported},
	)

	tests := map[string]struct {
		backupImport *backupv1alpha1.BackupImport
		importClass  *backupv1alpha1.BackupClass
		storage      *backupv1alpha1.BackupStorage
		err          string
		wantSentinel error
		wantWarnings []string
	}{
		"missing class": {
			backupImport: newImport("missing-class", "s1"),
			storage:      storage,
			err:          "backup class not found: 'missing-class'",
			wantSentinel: controller.ErrBackupClassNotFound,
		},
		"missing storage": {
			backupImport: newImport("import-class", "missing-storage"),
			importClass:  importClass,
			err:          "backup storage not found: backup storage 'missing-storage' does not exist",
			wantSentinel: controller.ErrBackupStorageNotFound,
		},
		"class does not support import": {
			backupImport: newImport("no-import-class", "s1"),
			importClass:  noImportClass,
			storage:      storage,
			err:          "backup import is not supported: class 'no-import-class'",
			wantSentinel: controller.ErrBackupImportUnsupported,
		},
		"class its provider rejects": {
			backupImport: newImport("drift-class", "s1"),
			importClass:  driftClass,
			storage:      storage,
			err:          `backup class incompatible: BackupClass "drift-class" is Incompatible with Provider "test-provider" (SpecDrift)`,
			wantSentinel: controller.ErrBackupClassIncompatible,
		},
		"storage the class cannot read": {
			backupImport: newImport("import-class", "gcs"),
			importClass:  importClass,
			storage:      gcsStorage,
			err:          `backup storage incompatible: BackupStorage "gcs" is Incompatible with BackupClass "import-class" (ParameterUnsupported)`,
			wantSentinel: controller.ErrBackupStorageIncompatible,
		},
		"compatible": {
			backupImport: newImport("import-class", "s1"),
			importClass:  importClass,
			storage:      storage,
		},
		"class not claimed yet": {
			backupImport: newImport("pending-class", "s1"),
			importClass:  pendingClass,
			storage:      storage,
			wantWarnings: []string{
				`BackupClass "pending-class" is Pending for Provider "test-provider" (ClassNotAccepted)`,
				`BackupStorage "s1" is Pending for BackupClass "pending-class" (VerdictPending)`,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Use DeepCopy to avoid race conditions since the fake client
			// modifies the objects' ResourceVersion during Build().
			objs := []ctrlclient.Object{s3StorageClass.DeepCopy()}
			if tt.importClass != nil {
				objs = append(objs, tt.importClass.DeepCopy())
			}

			if tt.storage != nil {
				objs = append(objs, tt.storage.DeepCopy())
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				Build()

			kubeConnector := kubernetes.NewEmpty(zap.NewNop().Sugar(), "test-namespace").
				WithKubernetesClient(fakeClient)

			mockNext := &handlers.MockHandler{}
			mockNext.
				On("CreateBackupImport", mock.Anything, mock.Anything, mock.Anything).
				Return(tt.backupImport, nil)

			handler := &validateHandler{
				log:           zap.NewNop().Sugar(),
				kubeConnector: kubeConnector,
				next:          mockNext,
			}

			ctx, warnings := kubernetes.WithWarningCollector(ctx)
			_, err := handler.CreateBackupImport(ctx, "prod", tt.backupImport)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				require.ErrorIs(t, err, ErrInvalidRequest)
				require.ErrorIs(t, err, tt.wantSentinel)
				mockNext.AssertNotCalled(t, "CreateBackupImport", mock.Anything, mock.Anything, mock.Anything)

				return
			}

			require.NoError(t, err)
			assertWarnings(t, tt.wantWarnings, warnings.Warnings())
		})
	}
}
