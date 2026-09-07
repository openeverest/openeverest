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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// TestK8s_PatchBackupStorage pins the merge semantics and the write-only
// credential flow. The fake client does no strict decoding, so fieldValidation
// is not observable here and is covered by api-tests/tests/backup-storage.spec.ts
// instead.
func TestK8s_PatchBackupStorage(t *testing.T) {
	t.Parallel()

	verifyTLS := false
	seed := func() *backupv1alpha1.BackupStorage {
		return &backupv1alpha1.BackupStorage{
			ObjectMeta: metav1.ObjectMeta{Name: "bs1", Namespace: "ns1"},
			Spec: backupv1alpha1.BackupStorageSpec{
				Type: backupv1alpha1.BackupStorageTypeS3,
				S3: &backupv1alpha1.BackupStorageS3Spec{
					Bucket:               "bucket-1",
					Region:               "us-east-1",
					EndpointURL:          "https://minio.minio.svc",
					VerifyTLS:            &verifyTLS,
					CredentialsSecretRef: common.SecretRef{Name: "bs1-creds"},
				},
			},
		}
	}

	newHandler := func() *k8sHandler {
		fakeClient := fake.NewClientBuilder().
			WithScheme(kubernetes.CreateScheme()).
			WithObjects(seed()).
			Build()
		return &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}
	}

	ctx := context.Background()

	t.Run("named member changes and the rest survives", func(t *testing.T) {
		t.Parallel()

		result, err := newHandler().PatchBackupStorage(ctx, "prod", "ns1", "bs1",
			[]byte(`{"spec":{"s3":{"bucket":"bucket-2"}}}`))
		require.NoError(t, err)

		assert.Equal(t, "bucket-2", result.Spec.S3.Bucket)
		assert.Equal(t, "us-east-1", result.Spec.S3.Region)
		assert.Equal(t, "bs1-creds", result.Spec.S3.CredentialsSecretRef.Name)
	})

	t.Run("credentials land in the stored object's secret and not on the object", func(t *testing.T) {
		t.Parallel()

		handler := newHandler()
		result, err := handler.PatchBackupStorage(ctx, "prod", "ns1", "bs1",
			[]byte(`{"spec":{"s3":{"accessKeyId":"key","secretAccessKey":"secret"}}}`))
		require.NoError(t, err)

		assert.Empty(t, result.Spec.S3.AccessKeyID)
		assert.Empty(t, result.Spec.S3.SecretAccessKey)

		secret, err := handler.kubeConnector.GetSecret(ctx, types.NamespacedName{Namespace: "ns1", Name: "bs1-creds"})
		require.NoError(t, err)
		assert.Equal(t, "key", secret.StringData["AWS_ACCESS_KEY_ID"])
		assert.Equal(t, "secret", secret.StringData["AWS_SECRET_ACCESS_KEY"])
	})

	t.Run("a secret reference in the patch wins over the stored one", func(t *testing.T) {
		t.Parallel()

		handler := newHandler()
		result, err := handler.PatchBackupStorage(ctx, "prod", "ns1", "bs1",
			[]byte(`{"spec":{"s3":{"credentialsSecretRef":{"name":"rotated"},"accessKeyId":"key","secretAccessKey":"secret"}}}`))
		require.NoError(t, err)

		assert.Equal(t, "rotated", result.Spec.S3.CredentialsSecretRef.Name)

		secret, err := handler.kubeConnector.GetSecret(ctx, types.NamespacedName{Namespace: "ns1", Name: "rotated"})
		require.NoError(t, err)
		assert.Equal(t, "key", secret.StringData["AWS_ACCESS_KEY_ID"])
	})

	t.Run("half a credential pair is a bad request and changes nothing", func(t *testing.T) {
		t.Parallel()

		handler := newHandler()
		_, err := handler.PatchBackupStorage(ctx, "prod", "ns1", "bs1",
			[]byte(`{"spec":{"s3":{"bucket":"bucket-2","accessKeyId":"key"}}}`))
		require.ErrorIs(t, err, ErrInvalidRequest)
		require.ErrorContains(t, err, "secretAccessKey is not provided")

		stored, err := handler.kubeConnector.GetBackupStorage(ctx, types.NamespacedName{Namespace: "ns1", Name: "bs1"})
		require.NoError(t, err)
		assert.Equal(t, "bucket-1", stored.Spec.S3.Bucket, "the patch must not be applied when the pair is rejected")
	})

	t.Run("a rejected patch does not rotate the Secret", func(t *testing.T) {
		t.Parallel()

		handler := newHandler()
		// The bucket's type is wrong, so the write fails after the credentials
		// were taken out of the document.
		_, err := handler.PatchBackupStorage(ctx, "prod", "ns1", "bs1",
			[]byte(`{"spec":{"s3":{"bucket":7,"accessKeyId":"key","secretAccessKey":"secret"}}}`))
		require.Error(t, err)

		_, err = handler.kubeConnector.GetSecret(ctx, types.NamespacedName{Namespace: "ns1", Name: "bs1-creds"})
		require.Error(t, err, "no Secret should be written for a patch the API server refused")
	})

	t.Run("a credential that is not a string is left for the API server", func(t *testing.T) {
		t.Parallel()

		handler := newHandler()
		_, err := handler.PatchBackupStorage(ctx, "prod", "ns1", "bs1",
			[]byte(`{"spec":{"s3":{"accessKeyId":7,"secretAccessKey":"secret"}}}`))
		require.ErrorContains(t, err, "accessKeyId", "the type complaint comes from Kubernetes, not from this handler")

		_, err = handler.kubeConnector.GetSecret(ctx, types.NamespacedName{Namespace: "ns1", Name: "bs1-creds"})
		require.Error(t, err, "no Secret should be written for a credential this handler cannot read")
	})
}
