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

package v1alpha1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/backupadmission"
)

const ns = "db"

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))
	return scheme
}

// objects returns a cluster where class "pbm" is claimed by psmdb, class
// "drift" is rejected by it, class "pending" is not claimed yet, storage
// "aws" is compatible with pbm and storage "gcs" is not, and Instance
// "mongo" runs on psmdb.
func objects() []client.Object {
	storageClass := &backupv1alpha1.BackupStorageClass{ObjectMeta: metav1.ObjectMeta{Name: "s3", UID: "s3-uid", Generation: 1}}
	class := func(name string, c backupv1alpha1.Compatibility, reason string) *backupv1alpha1.BackupClass {
		return &backupv1alpha1.BackupClass{
			ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID("uid-" + name), Generation: 1},
			Spec: backupv1alpha1.BackupClassSpec{
				ExecutionMode:   backupv1alpha1.BackupExecutionModeProviderManaged,
				ProviderManaged: &backupv1alpha1.ProviderManagedSpec{ProviderRef: common.ObjectRef{Name: "psmdb"}},
				SupportsImport:  true,
			},
			Status: backupv1alpha1.BackupClassStatus{Providers: []backupv1alpha1.BackupClassProviderCompatibility{{
				Name: "psmdb", Compatibility: c, Reason: reason, Message: "detail", ClassGeneration: 1,
			}}},
		}
	}
	pbm := class("pbm", backupv1alpha1.CompatibilityCompatible, "")
	storage := func(name string, c backupv1alpha1.Compatibility, reason string) *backupv1alpha1.BackupStorage {
		return &backupv1alpha1.BackupStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Generation: 1},
			Spec:       backupv1alpha1.BackupStorageSpec{BackupStorageClassRef: common.ObjectRef{Name: "s3"}},
			Status: backupv1alpha1.BackupStorageStatus{BackupClasses: []backupv1alpha1.BackupClassVerdict{{
				Name: "pbm", Compatibility: c, Reason: reason, Message: "detail",
				Observed: backupv1alpha1.BackupClassVerdictInputs{
					StorageGeneration: 1, ClassUID: pbm.UID, ClassGeneration: 1, StorageClassUID: "s3-uid", StorageClassGeneration: 1,
				},
			}}},
		}
	}
	return []client.Object{
		storageClass, pbm,
		class("drift", backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonSpecDrift),
		class("pending", backupv1alpha1.CompatibilityPending, backupv1alpha1.ReasonClassNotAccepted),
		storage("aws", backupv1alpha1.CompatibilityCompatible, ""),
		storage("gcs", backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonParameterUnsupported),
		&corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{Name: "mongo", Namespace: ns},
			Spec:       corev1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: "psmdb"}},
		},
		&backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "done", Namespace: ns},
			Spec:       backupv1alpha1.BackupSpec{ClassRef: common.ObjectRef{Name: "drift"}, StorageRef: common.ObjectRef{Name: "aws"}},
			Status:     backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded},
		},
	}
}

func newBackup(class, storage string) *backupv1alpha1.Backup {
	return &backupv1alpha1.Backup{
		TypeMeta:   metav1.TypeMeta{APIVersion: backupv1alpha1.GroupVersion.String(), Kind: "Backup"},
		ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: ns},
		Spec: backupv1alpha1.BackupSpec{
			Origin: backupv1alpha1.BackupOrigin{
				Type:        backupv1alpha1.BackupOriginTypeInstance,
				InstanceRef: &common.ObjectRef{Name: "mongo"},
			},
			ClassRef:   common.ObjectRef{Name: class},
			StorageRef: common.ObjectRef{Name: storage},
		},
	}
}

func newRestore(source string) *backupv1alpha1.Restore {
	return &backupv1alpha1.Restore{
		TypeMeta:   metav1.TypeMeta{APIVersion: backupv1alpha1.GroupVersion.String(), Kind: "Restore"},
		ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: ns},
		Spec: backupv1alpha1.RestoreSpec{
			InstanceRef: common.ObjectRef{Name: "mongo"},
			DataSource: backupv1alpha1.DataSource{
				Type:   backupv1alpha1.DataSourceTypeBackup,
				Backup: &backupv1alpha1.DataSourceBackup{BackupRef: common.ObjectRef{Name: source}},
			},
		},
	}
}

func newImport(class, storage string) *backupv1alpha1.BackupImport {
	return &backupv1alpha1.BackupImport{
		TypeMeta:   metav1.TypeMeta{APIVersion: backupv1alpha1.GroupVersion.String(), Kind: "BackupImport"},
		ObjectMeta: metav1.ObjectMeta{Name: "i", Namespace: ns},
		Spec:       backupv1alpha1.BackupImportSpec{ClassRef: common.ObjectRef{Name: class}, StorageRef: common.ObjectRef{Name: storage}},
	}
}

func webhookFor[T client.Object](t *testing.T, r backupadmission.Reader, decide Decide[T]) *admission.Webhook {
	t.Helper()
	return admission.WithValidator[T](newScheme(t), &OperationValidator[T]{Reader: r, Decide: decide})
}

// The webhooks answer an AdmissionReview the way kubectl shows it: a denial
// with the verdict's reason, or an admission with warnings.
func TestOperationWebhooks(t *testing.T) {
	t.Parallel()

	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(objects()...).Build()
	r := backupadmission.ClientReader{Reader: c}

	tests := []struct {
		name         string
		webhook      *admission.Webhook
		object       client.Object
		wantDenied   string
		wantWarnings []string
	}{
		{
			name:    "compatible backup",
			webhook: webhookFor(t, r, backupadmission.Backup),
			object:  newBackup("pbm", "aws"),
		},
		{
			name:       "backup to an incompatible storage",
			webhook:    webhookFor(t, r, backupadmission.Backup),
			object:     newBackup("pbm", "gcs"),
			wantDenied: `backup storage incompatible: BackupStorage "gcs" is Incompatible with BackupClass "pbm" (ParameterUnsupported): detail`,
		},
		{
			name:    "backup with a class not claimed yet",
			webhook: webhookFor(t, r, backupadmission.Backup),
			object:  newBackup("pending", "aws"),
			wantWarnings: []string{
				`BackupClass "pending" is Pending for Provider "psmdb" (ClassNotAccepted): detail`,
				`BackupStorage "aws" is Pending for BackupClass "pending" (VerdictPending): BackupStorage "aws" has not been judged against BackupClass "pending" yet`,
			},
		},
		{
			name:       "restore of a backup written with a drifted class",
			webhook:    webhookFor(t, r, backupadmission.Restore),
			object:     newRestore("done"),
			wantDenied: `backup class incompatible: BackupClass "drift" is Incompatible with Provider "psmdb" (SpecDrift): detail`,
		},
		{
			name:         "restore of a backup that does not exist yet",
			webhook:      webhookFor(t, r, backupadmission.Restore),
			object:       newRestore("later"),
			wantWarnings: []string{"backup not found: 'later' in namespace 'db'; the Restore waits until it exists"},
		},
		{
			name:    "compatible import",
			webhook: webhookFor(t, r, backupadmission.Import),
			object:  newImport("pbm", "aws"),
		},
		{
			name:       "import with a drifted class",
			webhook:    webhookFor(t, r, backupadmission.Import),
			object:     newImport("drift", "aws"),
			wantDenied: `backup class incompatible: BackupClass "drift" is Incompatible with Provider "psmdb" (SpecDrift): detail`,
			// The storage has no verdict for a class nobody claims.
			wantWarnings: []string{
				`BackupStorage "aws" is Pending for BackupClass "drift" (VerdictPending): BackupStorage "aws" has not been judged against BackupClass "drift" yet`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(tt.object)
			require.NoError(t, err)
			res := tt.webhook.Handle(context.Background(), admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				Namespace: ns,
				Name:      tt.object.GetName(),
				Object:    runtime.RawExtension{Raw: raw},
				DryRun:    new(true),
			}})

			if tt.wantDenied == "" {
				assert.True(t, res.Allowed, "denied: %v", res.Result)
			} else {
				assert.False(t, res.Allowed)
				assert.Equal(t, int32(http.StatusForbidden), res.Result.Code)
				assert.Equal(t, tt.wantDenied, res.Result.Message)
			}
			if len(tt.wantWarnings) == 0 {
				assert.Empty(t, res.Warnings)
			} else {
				assert.Equal(t, tt.wantWarnings, res.Warnings)
			}
		})
	}
}

// A failed read is an internal error, not a denial of the operation.
func TestOperationWebhookReadError(t *testing.T) {
	t.Parallel()

	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
			return errors.New("etcd unavailable")
		},
	}).Build()
	v := &OperationValidator[*backupv1alpha1.Backup]{Reader: backupadmission.ClientReader{Reader: c}, Decide: backupadmission.Backup}

	warnings, err := v.ValidateCreate(context.Background(), newBackup("pbm", "aws"))
	assert.Empty(t, warnings)
	require.ErrorContains(t, err, "etcd unavailable")
	var status interface{ Status() metav1.Status }
	require.ErrorAs(t, err, &status)
	assert.Equal(t, int32(http.StatusInternalServerError), status.Status().Code)
}

// Only creation is judged.
func TestOperationWebhookUpdateAndDelete(t *testing.T) {
	t.Parallel()

	v := &OperationValidator[*backupv1alpha1.Backup]{Decide: func(context.Context, backupadmission.Reader, *backupv1alpha1.Backup) (backupadmission.Decision, error) {
		return backupadmission.Decision{}, errors.New("not called")
	}}
	warnings, err := v.ValidateUpdate(context.Background(), newBackup("pbm", "gcs"), newBackup("pbm", "gcs"))
	require.NoError(t, err)
	assert.Empty(t, warnings)
	warnings, err = v.ValidateDelete(context.Background(), newBackup("pbm", "gcs"))
	require.NoError(t, err)
	assert.Empty(t, warnings)
}
