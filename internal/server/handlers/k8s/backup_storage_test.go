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

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func testBackupStorage(name, class, mode string) *backupv1alpha1.BackupStorage {
	return &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns1"},
		Spec: backupv1alpha1.BackupStorageSpec{
			BackupStorageClassRef: common.ObjectRef{Name: class},
			Location:              backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "bucket-1"}},
			Parameters:            &runtime.RawExtension{Raw: []byte(`{"region":"us-east-1"}`)},
			Authentication: common.Authentication{
				Mode:                 mode,
				CredentialsSecretRef: &common.SecretRef{Name: name + "-creds"},
			},
		},
	}
}

// newBackupStorageHandler serves objs through a fake client that indexes the
// BackupStorage selectable fields, as the API server does.
func newBackupStorageHandler(objs ...ctrlclient.Object) *k8sHandler {
	storage := &backupv1alpha1.BackupStorage{}
	fakeClient := fake.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithObjects(objs...).
		WithIndex(storage, backupv1alpha1.BackupStorageClassField, func(o ctrlclient.Object) []string {
			bs, _ := o.(*backupv1alpha1.BackupStorage)
			return []string{bs.Spec.BackupStorageClassRef.Name}
		}).
		WithIndex(storage, backupv1alpha1.BackupStorageCredentialsSecretField, func(o ctrlclient.Object) []string {
			bs, _ := o.(*backupv1alpha1.BackupStorage)
			if ref := bs.Spec.Authentication.CredentialsSecretRef; ref != nil {
				return []string{ref.Name}
			}
			return []string{""}
		}).
		WithIndex(storage, backupv1alpha1.BackupStorageCASecretField, func(o ctrlclient.Object) []string {
			bs, _ := o.(*backupv1alpha1.BackupStorage)
			if bs.Spec.TLS != nil && bs.Spec.TLS.CA != nil && bs.Spec.TLS.CA.SecretKeyRef != nil {
				return []string{bs.Spec.TLS.CA.SecretKeyRef.Name}
			}
			return []string{""}
		}).
		Build()
	return &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
		log:           zap.NewNop().Sugar(),
	}
}

// withCA returns bs with its CA bundle in the Secret named name.
func withCA(bs *backupv1alpha1.BackupStorage, name string) *backupv1alpha1.BackupStorage {
	bs.Spec.TLS = &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: name, Key: "ca.crt"}}}
	return bs
}

func TestK8s_ListBackupStorages(t *testing.T) {
	t.Parallel()

	inOtherNamespace := testBackupStorage("other-namespace", "s3", "AccessKey")
	inOtherNamespace.Namespace = "ns2"
	inOtherNamespace.Spec.Authentication.CredentialsSecretRef.Name = "shared-ca"

	handler := newBackupStorageHandler(
		&backupv1alpha1.BackupClass{
			ObjectMeta: metav1.ObjectMeta{Name: "pbm"},
			Spec: backupv1alpha1.BackupClassSpec{BackupStorageClasses: []backupv1alpha1.BackupStorageClassSupport{
				{Name: "s3", AuthModes: []string{"AccessKey"}},
				{Name: "gcs", AuthModes: []string{"AccessKey"}},
			}},
		},
		withCA(testBackupStorage("writable", "s3", "AccessKey"), "shared-ca"),
		testBackupStorage("refused", "s3", "WorkloadIdentity"),
		withCA(testBackupStorage("other-class", "gcs", "AccessKey"), "shared-ca"),
		// Its own CA as credentials too: listed once.
		withCA(testBackupStorage("self", "s3", "AccessKey"), "self-creds"),
		inOtherNamespace,
	)

	tests := []struct {
		name   string
		params *api.ListBackupStoragesParams
		want   []string
	}{
		{name: "nil params", want: []string{"other-class", "refused", "self", "writable"}},
		{name: "empty filters are ignored", params: &api.ListBackupStoragesParams{BackupClass: new(""), BackupStorageClass: new(""), Secret: new("")}, want: []string{"other-class", "refused", "self", "writable"}},
		{name: "by backup class", params: &api.ListBackupStoragesParams{BackupClass: new("pbm")}, want: []string{"other-class", "self", "writable"}},
		{name: "by backup storage class", params: &api.ListBackupStoragesParams{BackupStorageClass: new("s3")}, want: []string{"refused", "self", "writable"}},
		{name: "by both classes", params: &api.ListBackupStoragesParams{BackupClass: new("pbm"), BackupStorageClass: new("s3")}, want: []string{"self", "writable"}},
		{name: "unknown backup class", params: &api.ListBackupStoragesParams{BackupClass: new("nope")}, want: []string{}},
		{name: "by credentials Secret", params: &api.ListBackupStoragesParams{Secret: new("refused-creds")}, want: []string{"refused"}},
		{name: "by CA Secret", params: &api.ListBackupStoragesParams{Secret: new("shared-ca")}, want: []string{"other-class", "writable"}},
		{name: "by a Secret used as both", params: &api.ListBackupStoragesParams{Secret: new("self-creds")}, want: []string{"self"}},
		{name: "by Secret and backup storage class", params: &api.ListBackupStoragesParams{Secret: new("shared-ca"), BackupStorageClass: new("s3")}, want: []string{"writable"}},
		{name: "by Secret and backup class", params: &api.ListBackupStoragesParams{Secret: new("refused-creds"), BackupClass: new("pbm")}, want: []string{}},
		{name: "by an unused Secret", params: &api.ListBackupStoragesParams{Secret: new("nope")}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			list, err := handler.ListBackupStorages(context.Background(), "prod", "ns1", tt.params)
			require.NoError(t, err)
			names := make([]string, 0, len(list.Items))
			for _, bs := range list.Items {
				names = append(names, bs.Name)
			}
			if tt.params != nil && pointer.Get(tt.params.Secret) != "" {
				// Merged lists come back sorted, like a single list, and
				// without the resourceVersion neither read has alone.
				assert.Equal(t, tt.want, names)
				assert.Empty(t, list.ResourceVersion)
				return
			}
			assert.ElementsMatch(t, tt.want, names)
		})
	}
}

// TestK8s_PatchBackupStorage pins the merge semantics. The fake client does no
// strict decoding, so fieldValidation is not observable here and is covered by
// api-tests/tests/backup-storage.spec.ts instead.
func TestK8s_PatchBackupStorage(t *testing.T) {
	t.Parallel()

	handler := newBackupStorageHandler(testBackupStorage("bs1", "s3", "AccessKey"))
	result, err := handler.PatchBackupStorage(context.Background(), "prod", "ns1", "bs1",
		[]byte(`{"spec":{"authentication":{"credentialsSecretRef":{"name":"rotated"}}}}`),
	)
	require.NoError(t, err)

	assert.Equal(t, "rotated", result.Spec.Authentication.CredentialsSecretRef.Name)
	assert.Equal(t, "AccessKey", result.Spec.Authentication.Mode)
	assert.Equal(t, "bucket-1", result.Spec.Location.ObjectStore.Bucket)
	assert.JSONEq(t, `{"region":"us-east-1"}`, string(result.Spec.Parameters.Raw))
}
