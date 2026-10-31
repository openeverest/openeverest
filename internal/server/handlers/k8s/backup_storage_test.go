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
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func testBackupStorage(name, class string, backupClasses ...backupv1alpha1.BackupClassCompatibility) *backupv1alpha1.BackupStorage {
	return &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns1"},
		Spec: backupv1alpha1.BackupStorageSpec{
			BackupStorageClassRef: common.ObjectRef{Name: class},
			Location:              backupv1alpha1.BackupStorageLocation{Bucket: "bucket-1"},
			Parameters:            &runtime.RawExtension{Raw: []byte(`{"region":"us-east-1"}`)},
			Authentication: common.Authentication{
				Mode:                 "AccessKey",
				CredentialsSecretRef: &common.SecretRef{Name: name + "-creds"},
			},
		},
		Status: backupv1alpha1.BackupStorageStatus{BackupClasses: backupClasses},
	}
}

func newBackupStorageHandler(objs ...ctrlclient.Object) *k8sHandler {
	fakeClient := fake.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithObjects(objs...).
		Build()
	return &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
		log:           zap.NewNop().Sugar(),
	}
}

func TestK8s_ListBackupStorages(t *testing.T) {
	t.Parallel()

	handler := newBackupStorageHandler(
		testBackupStorage("writable", "s3", backupv1alpha1.BackupClassCompatibility{Name: "pbm", Compatible: true}),
		testBackupStorage("refused", "s3", backupv1alpha1.BackupClassCompatibility{Name: "pbm", Reason: "AuthModeUnsupported"}),
		testBackupStorage("unevaluated", "s3"),
		testBackupStorage("other-class", "gcs", backupv1alpha1.BackupClassCompatibility{Name: "pbm", Compatible: true}),
	)

	tests := []struct {
		name               string
		backupClass        string
		backupStorageClass string
		want               []string
	}{
		{name: "no filter", want: []string{"other-class", "refused", "unevaluated", "writable"}},
		{name: "by backup class", backupClass: "pbm", want: []string{"other-class", "writable"}},
		{name: "by backup storage class", backupStorageClass: "s3", want: []string{"refused", "unevaluated", "writable"}},
		{name: "by both", backupClass: "pbm", backupStorageClass: "s3", want: []string{"writable"}},
		{name: "unknown backup class", backupClass: "nope", want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			list, err := handler.ListBackupStorages(context.Background(), "prod", "ns1", tt.backupClass, tt.backupStorageClass)
			require.NoError(t, err)
			names := make([]string, 0, len(list.Items))
			for _, bs := range list.Items {
				names = append(names, bs.Name)
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

	handler := newBackupStorageHandler(testBackupStorage("bs1", "s3"))
	result, err := handler.PatchBackupStorage(context.Background(), "prod", "ns1", "bs1",
		[]byte(`{"spec":{"authentication":{"credentialsSecretRef":{"name":"rotated"}}}}`),
	)
	require.NoError(t, err)

	assert.Equal(t, "rotated", result.Spec.Authentication.CredentialsSecretRef.Name)
	assert.Equal(t, "AccessKey", result.Spec.Authentication.Mode)
	assert.Equal(t, "bucket-1", result.Spec.Location.Bucket)
	assert.JSONEq(t, `{"region":"us-east-1"}`, string(result.Spec.Parameters.Raw))
}
