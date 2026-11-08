// everest
// Copyright (C) 2023 Percona LLC
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

package backup

import (
	"encoding/json"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/kube-openapi/pkg/validation/spec"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/config/crd/bases"
)

// backupTypeConverter types server-side apply on backup objects with their
// CRD schemas, as the API server does.
func backupTypeConverter(t *testing.T) managedfields.TypeConverter { //nolint:ireturn // the fake client takes the interface
	t.Helper()
	models := make(map[string]*spec.Schema)
	for _, plural := range []string{"backupclasses", "backupstorages", "backupstorageclasses"} {
		raw, err := fs.ReadFile(bases.CRDs, "backup.openeverest.io_"+plural+".yaml")
		require.NoError(t, err)
		crd := &apiextensionsv1.CustomResourceDefinition{}
		require.NoError(t, yaml.Unmarshal(raw, crd))
		for _, v := range crd.Spec.Versions {
			js, err := json.Marshal(v.Schema.OpenAPIV3Schema)
			require.NoError(t, err)
			s := &spec.Schema{}
			require.NoError(t, json.Unmarshal(js, s))
			s.AddExtension("x-kubernetes-group-version-kind", []any{
				map[string]any{"group": crd.Spec.Group, "version": v.Name, "kind": crd.Spec.Names.Kind},
			})
			models[crd.Spec.Group+"."+v.Name+"."+crd.Spec.Names.Kind] = s
		}
	}
	tc, err := managedfields.NewTypeConverter(models, false)
	require.NoError(t, err)
	return tc
}

func newVerdictTestClient(t *testing.T, objs ...client.Object) client.Client { //nolint:ireturn
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithTypeConverters(backupTypeConverter(t), managedfields.NewDeducedTypeConverter()).
		WithObjects(objs...).
		WithStatusSubresource(&backupv1alpha1.BackupClass{}, &backupv1alpha1.BackupStorage{}).
		Build()
}

func claimedClass(gen int64) *backupv1alpha1.BackupClass {
	class := verdictClass("pbm", gen, backupv1alpha1.BackupStorageClassSupport{Name: "s3", AuthModes: []string{"AccessKey"}},
		acceptedAt(1, metav1.ConditionTrue, "Accepted"),
	)
	class.Status.Conditions[0].LastTransitionTime = metav1.Now()
	return class
}

func TestBackupClassReconcile(t *testing.T) {
	t.Parallel()

	c := newVerdictTestClient(t, claimedClass(1))
	r := &BackupClassReconciler{Client: c}
	req := ctrl.Request{NamespacedName: client.ObjectKey{Name: "pbm"}}

	_, err := r.Reconcile(t.Context(), req)
	require.NoError(t, err)
	got := &backupv1alpha1.BackupClass{}
	require.NoError(t, c.Get(t.Context(), req.NamespacedName, got))
	require.Len(t, got.Status.Providers, 1)
	assert.Equal(t, backupv1alpha1.ReasonProviderNotFound, got.Status.Providers[0].Reason)
	assert.True(t, got.IsAccepted(), "the executor's Accepted condition is kept")

	require.NoError(t, c.Create(t.Context(), &corev1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "mongo"}}))
	_, err = r.Reconcile(t.Context(), req)
	require.NoError(t, err)
	require.NoError(t, c.Get(t.Context(), req.NamespacedName, got))
	require.Len(t, got.Status.Providers, 1)
	assert.Equal(t, backupv1alpha1.CompatibilityCompatible, got.Status.Providers[0].Compatibility)
	assert.True(t, got.IsAccepted())
}

func TestBackupStorageReconcile_Verdicts(t *testing.T) {
	t.Parallel()

	storageClass := &backupv1alpha1.BackupStorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "s3"},
		Spec: backupv1alpha1.BackupStorageClassSpec{
			AuthModes: []common.AuthMode{{Name: "AccessKey", Credentials: &common.CredentialKeys{Required: []string{"key"}}}},
		},
	}
	storage := verdictStorage(`{}`, nil)
	storage.Spec.Parameters = nil
	creds := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "ns"}, Data: map[string][]byte{"key": []byte("v")}}
	other := verdictClass("swift", 1, backupv1alpha1.BackupStorageClassSupport{Name: "swift.example.com", AuthModes: []string{"Token"}},
		acceptedAt(1, metav1.ConditionTrue, "Accepted"),
	)
	other.UID = "uid-swift"
	other.Status.Conditions[0].LastTransitionTime = metav1.Now()
	c := newVerdictTestClient(t, storageClass, storage, creds, claimedClass(1), other)
	r := &BackupStorageReconciler{Client: c}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(storage)}

	_, err := r.Reconcile(t.Context(), req)
	require.NoError(t, err)
	got := &backupv1alpha1.BackupStorage{}
	require.NoError(t, c.Get(t.Context(), req.NamespacedName, got))
	cond := meta.FindStatusCondition(got.Status.Conditions, backupv1alpha1.BackupStorageConditionAccepted)
	require.NotNil(t, cond)
	require.Equal(t, metav1.ConditionTrue, cond.Status, cond.Message)
	require.Len(t, got.Status.BackupClasses, 2)
	assert.Equal(t, "pbm", got.Status.BackupClasses[0].Name)
	assert.Equal(t, backupv1alpha1.CompatibilityCompatible, got.Status.BackupClasses[0].Compatibility)
	assert.Equal(t, "swift", got.Status.BackupClasses[1].Name)
	assert.Equal(t, backupv1alpha1.ReasonBackupStorageClassUnsupported, got.Status.BackupClasses[1].Reason)

	pbm := &backupv1alpha1.BackupClass{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: "pbm"}, pbm))
	assert.Equal(t, backupv1alpha1.CompatibilityCompatible, backupv1alpha1.CurrentVerdict(got, pbm, storageClassFrom(t, c)).Compatibility,
		"the published verdict is current",
	)

	// A class whose claim is stale loses its verdict.
	pbm.Status.Conditions[0].ObservedGeneration = 0
	require.NoError(t, c.Status().Update(t.Context(), pbm))
	_, err = r.Reconcile(t.Context(), req)
	require.NoError(t, err)
	require.NoError(t, c.Get(t.Context(), req.NamespacedName, got))
	require.Len(t, got.Status.BackupClasses, 1)
	assert.Equal(t, "swift", got.Status.BackupClasses[0].Name)
}

func storageClassFrom(t *testing.T, c client.Client) *backupv1alpha1.BackupStorageClass {
	t.Helper()
	sc := &backupv1alpha1.BackupStorageClass{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: "s3"}, sc))
	return sc
}
