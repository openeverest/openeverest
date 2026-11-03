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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestCreateSecret(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name   string
		secret *corev1.Secret
	}{
		{
			name: "secret with stringData",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret-stringdata",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"accessKey": "myaccesskey",
					"secretKey": "mysecretkey",
				},
			},
		},
		{
			name: "secret with data",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret-data",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("secret123"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			handler := &k8sHandler{
				kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
				log:           zap.NewNop().Sugar(),
			}

			result, err := handler.CreateSecret(ctx, "", namespace, tt.secret)

			require.NoError(t, err)
			assert.Contains(t, tt.secret.Labels, "openeverest.io/managed")
			assert.Nil(t, result.Data)
			assert.Nil(t, result.StringData)
		})
	}
}

func TestCreateSecretDropsOwnerReferencesAndFinalizers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	handler := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
		log:           zap.NewNop().Sugar(),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pmm-credentials",
			Labels: map[string]string{
				common.OpenEverestMonitoringClassLabel: "pmm",
				common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "core.openeverest.io/v1alpha1",
				Kind:       "Instance",
				Name:       "orders",
				UID:        "1234",
			}},
			Finalizers: []string{"example.com/keep"},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"token": "glsa_token"},
	}

	_, err := handler.CreateSecret(ctx, "", namespace, secret)
	require.NoError(t, err)

	stored := &corev1.Secret{}
	require.NoError(t, fakeClient.Get(ctx, ctrlclient.ObjectKey{Namespace: namespace, Name: "pmm-credentials"}, stored))
	assert.Empty(t, stored.OwnerReferences)
	assert.Empty(t, stored.Finalizers)
	assert.Equal(t, "true", stored.Labels[common.OpenEverestManagedLabel])
	assert.Equal(t, "pmm", stored.Labels[common.OpenEverestMonitoringClassLabel])
}

func TestGetSecret(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name          string
		secret        *corev1.Secret
		expectedError string
	}{
		{
			name: "secret with stringData",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-stringdata",
					Namespace: namespace,
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
						common.OpenEverestManagedLabel:    "true",
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"accessKey": "myaccesskey",
					"secretKey": "mysecretkey",
				},
			},
		},
		{
			name: "secret with data",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-data",
					Namespace: namespace,
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
						common.OpenEverestManagedLabel:    "true",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("secret123"),
				},
			},
		},
		{
			name: "secret not managed by OpenEverest",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-unmanaged",
					Namespace: namespace,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("secret123"),
				},
			},
			expectedError: "not managed by OpenEverest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.secret).Build()

			handler := &k8sHandler{
				kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
				log:           zap.NewNop().Sugar(),
			}

			result, err := handler.GetSecret(ctx, "", namespace, tt.secret.Name)

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)

				return
			}

			require.NoError(t, err)
			assert.Nil(t, result.Data)
			assert.Nil(t, result.StringData)
		})
	}
}

func TestPatchSecret(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	labels := map[string]string{
		common.OpenEverestManagedLabel:         "true",
		common.OpenEverestMonitoringClassLabel: "pmm",
		common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
	}
	stored := func(name string, labels map[string]string) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   namespace,
				Labels:      labels,
				Annotations: map[string]string{"example.com/note": "kept"},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"token": []byte("old"), "orgID": []byte("1"), "region": []byte("eu")},
		}
	}
	newHandler := func(objs ...ctrlclient.Object) (*k8sHandler, ctrlclient.Client) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
		return &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}, fakeClient
	}
	get := func(t *testing.T, c ctrlclient.Client, name string) *corev1.Secret {
		t.Helper()
		secret := &corev1.Secret{}
		require.NoError(t, c.Get(ctx, ctrlclient.ObjectKey{Namespace: namespace, Name: name}, secret))
		return secret
	}

	t.Run("merges the data and changes nothing else", func(t *testing.T) {
		t.Parallel()

		handler, c := newHandler(stored("pmm-credentials", labels))
		current := get(t, c, "pmm-credentials").ResourceVersion

		patch := []byte(`{
			"metadata": {
				"resourceVersion": "` + current + `",
				"name": "other-name",
				"annotations": {"example.com/note": "replaced"},
				"ownerReferences": [{"apiVersion": "v1", "kind": "ConfigMap", "name": "owner", "uid": "1"}]
			},
			"type": "Opaque",
			"data": {"orgID": null},
			"stringData": {"token": "new"}
		}`)
		result, err := handler.PatchSecret(ctx, "", namespace, "pmm-credentials", patch)
		require.NoError(t, err)
		assert.Nil(t, result.Data)
		assert.Nil(t, result.StringData)

		got := get(t, c, "pmm-credentials")
		assert.Equal(t, map[string][]byte{"token": []byte("new"), "region": []byte("eu")}, got.Data,
			"keys sent are set, keys left out are kept, a null removes a key",
		)
		assert.Equal(t, labels, got.Labels)
		assert.Equal(t, map[string]string{"example.com/note": "kept"}, got.Annotations)
		assert.Empty(t, got.OwnerReferences)
		assert.Equal(t, corev1.SecretTypeOpaque, got.Type)
	})

	t.Run("refuses a stale resourceVersion", func(t *testing.T) {
		t.Parallel()

		handler, c := newHandler(stored("pmm-credentials", labels))
		current := get(t, c, "pmm-credentials").ResourceVersion
		patch := func(token string) []byte {
			return []byte(`{"metadata": {"resourceVersion": "` + current + `"}, "stringData": {"token": "` + token + `"}}`)
		}

		_, err := handler.PatchSecret(ctx, "", namespace, "pmm-credentials", patch("new"))
		require.NoError(t, err)

		_, err = handler.PatchSecret(ctx, "", namespace, "pmm-credentials", patch("newer"))
		require.True(t, k8serrors.IsConflict(err), "a stale resourceVersion must be refused, got %v", err)
		assert.Equal(t, []byte("new"), get(t, c, "pmm-credentials").Data["token"])
	})

	t.Run("refuses a patch without a resourceVersion", func(t *testing.T) {
		t.Parallel()

		handler, c := newHandler(stored("pmm-credentials", labels))
		_, err := handler.PatchSecret(ctx, "", namespace, "pmm-credentials", []byte(`{"stringData": {"token": "new"}}`))
		require.ErrorIs(t, err, ErrInvalidRequest)
		require.ErrorContains(t, err, "metadata.resourceVersion is required")
		assert.Equal(t, []byte("old"), get(t, c, "pmm-credentials").Data["token"])
	})

	t.Run("refuses a Secret the API does not manage", func(t *testing.T) {
		t.Parallel()

		handler, c := newHandler(stored("unmanaged", nil))
		current := get(t, c, "unmanaged").ResourceVersion
		_, err := handler.PatchSecret(ctx, "", namespace, "unmanaged",
			[]byte(`{"metadata": {"resourceVersion": "`+current+`"}, "stringData": {"token": "new"}}`),
		)
		require.ErrorIs(t, err, ErrNotFound)
		assert.Equal(t, []byte("old"), get(t, c, "unmanaged").Data["token"])
	})

	t.Run("reports a missing Secret as not found", func(t *testing.T) {
		t.Parallel()

		handler, _ := newHandler()
		_, err := handler.PatchSecret(ctx, "", namespace, "missing",
			[]byte(`{"metadata": {"resourceVersion": "1"}, "stringData": {"token": "new"}}`),
		)
		require.True(t, k8serrors.IsNotFound(err), "got %v", err)
	})
}

func TestListSecrets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	secrets := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-stringdata",
				Namespace: namespace,
				Labels: map[string]string{
					common.OpenEverestDefinitionLabel: "database-credentials",
					common.OpenEverestManagedLabel:    "true",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"accessKey": []byte("myaccesskey"),
				"secretKey": []byte("mysecretkey"),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-data",
				Namespace: namespace,
				Labels: map[string]string{
					common.OpenEverestDefinitionLabel: "database-credentials",
					common.OpenEverestManagedLabel:    "true",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret123"),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "not-managed-by-openeverest",
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret123"),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-credentials",
				Namespace: namespace,
				Labels: map[string]string{
					common.OpenEverestMonitoringClassLabel: "pmm",
					common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
					common.OpenEverestManagedLabel:         "true",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"token": []byte("glsa_token"),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backup-storage-prod-credentials",
				Namespace: namespace,
				Labels: map[string]string{
					common.OpenEverestBackupStorageClassLabel: "s3",
					common.OpenEverestDefinitionLabel:         "AccessKey",
					common.OpenEverestManagedLabel:            "true",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"AWS_ACCESS_KEY_ID":     []byte("AKIA"),
				"AWS_SECRET_ACCESS_KEY": []byte("shh"),
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&secrets[0], &secrets[1], &secrets[2], &secrets[3], &secrets[4]).
		Build()

	handler := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
		log:           zap.NewNop().Sugar(),
	}

	tests := []struct {
		name   string
		filter handlers.SecretFilter
		want   []string
	}{
		{
			name: "no filter lists managed secrets only",
			want: []string{"backup-storage-prod-credentials", "pmm-credentials", "secret-data", "secret-stringdata"},
		},
		{
			name:   "by definition",
			filter: handlers.SecretFilter{Definition: "database-credentials"},
			want:   []string{"secret-data", "secret-stringdata"},
		},
		{
			name:   "by monitoring class and definition",
			filter: handlers.SecretFilter{MonitoringClass: "pmm", Definition: "ServiceAccountToken"},
			want:   []string{"pmm-credentials"},
		},
		{
			name:   "by backup storage class and definition",
			filter: handlers.SecretFilter{BackupStorageClass: "s3", Definition: "AccessKey"},
			want:   []string{"backup-storage-prod-credentials"},
		},
		{
			name:   "by another monitoring class",
			filter: handlers.SecretFilter{MonitoringClass: "prometheus"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := handler.ListSecrets(ctx, "", namespace, tt.filter)
			require.NoError(t, err)

			names := make([]string, 0, len(result.Items))
			for _, item := range result.Items {
				assert.Nil(t, item.Data)
				assert.Nil(t, item.StringData)
				names = append(names, item.Name)
			}
			assert.ElementsMatch(t, tt.want, names)
		})
	}
}

func TestSecretView(t *testing.T) {
	t.Parallel()

	created := metav1.NewTime(time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC))
	deleted := metav1.NewTime(time.Date(2026, 11, 10, 11, 0, 0, 0, time.UTC))
	labels := map[string]string{
		common.OpenEverestManagedLabel:         "true",
		common.OpenEverestMonitoringClassLabel: "pmm",
		common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
	}
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "pmm-credentials",
			GenerateName:               "pmm-",
			Namespace:                  "team-a",
			UID:                        "1111",
			ResourceVersion:            "42",
			Generation:                 3,
			CreationTimestamp:          created,
			DeletionTimestamp:          &deleted,
			DeletionGracePeriodSeconds: new(int64(0)),
			Labels:                     labels,
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": `{"stringData":{"token":"glsa_plaintext"}}`,
				"reconcile.external-secrets.io/data-hash":          "hash-of-the-data",
				"evilopeneverest.io/note":                          "not ours",
				"openeverest.io":                                   "no prefix",
				"openeverest.io/note":                              "kept",
				"monitoring.openeverest.io/attached-destinations":  "2",
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "monitoring.openeverest.io/v1alpha1",
				Kind:               "MonitoringDestination",
				Name:               "pmm-prod",
				UID:                "2222",
				Controller:         new(true),
				BlockOwnerDeletion: new(true),
			}},
			Finalizers: []string{"monitoring.openeverest.io/secret-protection"},
			ManagedFields: []metav1.ManagedFieldsEntry{{
				Manager:    "kubectl-client-side-apply",
				Operation:  metav1.ManagedFieldsOperationUpdate,
				FieldsType: "FieldsV1",
				FieldsV1:   metav1.NewFieldsV1(`{"f:data":{"f:token":{}}}`),
			}},
		},
		Immutable:  new(true),
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"token": []byte("glsa_plaintext")},
		StringData: map[string]string{"token": "glsa_plaintext"}, //nolint:gosec // test usage only
	}

	want := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pmm-credentials",
			Namespace:         "team-a",
			UID:               "1111",
			ResourceVersion:   "42",
			CreationTimestamp: created,
			DeletionTimestamp: &deleted,
			Labels:            labels,
			Annotations: map[string]string{
				"openeverest.io/note":                             "kept",
				"monitoring.openeverest.io/attached-destinations": "2",
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "monitoring.openeverest.io/v1alpha1",
				Kind:       "MonitoringDestination",
				Name:       "pmm-prod",
				UID:        "2222",
				Controller: new(true),
			}},
		},
		Immutable: new(true),
		Type:      corev1.SecretTypeOpaque,
	}

	view := secretView(secret)
	assert.Equal(t, want, view)

	encoded, err := json.Marshal(view)
	require.NoError(t, err)
	for _, leak := range []string{"glsa_plaintext", "f:token", "data-hash", "managedFields", "finalizers"} {
		assert.NotContains(t, string(encoded), leak)
	}

	view.Labels["team"] = "a"
	assert.NotContains(t, secret.Labels, "team", "the view does not share maps with the Secret")
}

func TestSecretResponsesShowOnlyTheView(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	// A Secret written with client-side kubectl apply carries its data in an annotation.
	applied := func() *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-credentials",
				Namespace: namespace,
				Labels: map[string]string{
					common.OpenEverestManagedLabel:         "true",
					common.OpenEverestMonitoringClassLabel: "pmm",
					common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
				},
				Annotations: map[string]string{
					"kubectl.kubernetes.io/last-applied-configuration": `{"stringData":{"token":"glsa_plaintext"}}`,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"token": []byte("glsa_plaintext")},
		}
	}
	newHandler := func(t *testing.T, objs ...ctrlclient.Object) (*k8sHandler, string) {
		t.Helper()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
		version := ""
		if len(objs) > 0 {
			stored := &corev1.Secret{}
			require.NoError(t, fakeClient.Get(ctx, ctrlclient.ObjectKeyFromObject(objs[0]), stored))
			version = stored.ResourceVersion
		}
		return &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}, version
	}

	responses := map[string]func(t *testing.T) any{
		"create": func(t *testing.T) any {
			t.Helper()
			h, _ := newHandler(t)
			res, err := h.CreateSecret(ctx, "", namespace, applied())
			require.NoError(t, err)
			return res
		},
		"get": func(t *testing.T) any {
			t.Helper()
			h, _ := newHandler(t, applied())
			res, err := h.GetSecret(ctx, "", namespace, "pmm-credentials")
			require.NoError(t, err)
			return res
		},
		"list": func(t *testing.T) any {
			t.Helper()
			h, _ := newHandler(t, applied())
			res, err := h.ListSecrets(ctx, "", namespace, handlers.SecretFilter{})
			require.NoError(t, err)
			require.Len(t, res.Items, 1)
			return res
		},
		"patch": func(t *testing.T) any {
			t.Helper()
			h, version := newHandler(t, applied())
			res, err := h.PatchSecret(ctx, "", namespace, "pmm-credentials",
				[]byte(`{"metadata": {"resourceVersion": "`+version+`"}, "stringData": {"token": "glsa_rotated"}}`),
			)
			require.NoError(t, err)
			return res
		},
	}

	for name, respond := range responses {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(respond(t))
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), "glsa_")
			assert.NotContains(t, string(encoded), "last-applied-configuration")
			assert.Contains(t, string(encoded), `"openeverest.io/definition":"ServiceAccountToken"`)
		})
	}
}
