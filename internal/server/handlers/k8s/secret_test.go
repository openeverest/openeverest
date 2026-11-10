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
	"errors"
	"maps"
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

	"github.com/openeverest/openeverest/v2/internal/secretdefinition"
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
						common.OpenEverestProviderLabel:   "test-provider",
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
						common.OpenEverestProviderLabel:   "test-provider",
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
			name: "secret without definition labels",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-unlabelled",
					Namespace: namespace,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("secret123"),
				},
			},
			expectedError: `secrets "secret-unlabelled" not found`,
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

	t.Run("does not find a Secret without definition labels", func(t *testing.T) {
		t.Parallel()

		handler, c := newHandler(stored("unmanaged", map[string]string{common.OpenEverestManagedLabel: "true"}))
		current := get(t, c, "unmanaged").ResourceVersion
		_, err := handler.PatchSecret(ctx, "", namespace, "unmanaged",
			[]byte(`{"metadata": {"resourceVersion": "`+current+`"}, "stringData": {"token": "new"}}`),
		)
		require.True(t, k8serrors.IsNotFound(err), "got %v", err)
		require.EqualError(t, err, `secrets "unmanaged" not found`)
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
					common.OpenEverestProviderLabel:   "test-provider",
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
					common.OpenEverestProviderLabel:   "test-provider",
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
			// Applied with kubectl or by Git: labelled, but not created through the API.
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-credentials-from-git",
				Namespace: namespace,
				Labels: map[string]string{
					common.OpenEverestMonitoringClassLabel: "pmm",
					common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"token": []byte("glsa_token"),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "corp-ca",
				Namespace: namespace,
				Labels: map[string]string{
					common.OpenEverestDefinitionLabel: common.OpenEverestCABundleDefinition,
				},
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "external-secrets.io/v1",
					Kind:       "ExternalSecret",
					Name:       "corp-ca",
					UID:        "1111",
					Controller: new(true),
				}},
			},
			Type: corev1.SecretTypeOpaque,
		},
		{
			// A definition other than ca-bundle needs a source label.
			ObjectMeta: metav1.ObjectMeta{
				Name:      "definition-without-source",
				Namespace: namespace,
				Labels: map[string]string{
					common.OpenEverestDefinitionLabel: "database-credentials",
					common.OpenEverestManagedLabel:    "true",
				},
			},
			Type: corev1.SecretTypeOpaque,
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "two-sources",
				Namespace: namespace,
				Labels: map[string]string{
					common.OpenEverestProviderLabel:        "test-provider",
					common.OpenEverestMonitoringClassLabel: "pmm",
					common.OpenEverestDefinitionLabel:      "database-credentials",
				},
			},
			Type: corev1.SecretTypeOpaque,
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "managed-without-definition",
				Namespace: namespace,
				Labels: map[string]string{
					common.OpenEverestMonitoringClassLabel: "pmm",
					common.OpenEverestManagedLabel:         "true",
				},
			},
			Type: corev1.SecretTypeOpaque,
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-namespace",
				Namespace: "other-namespace",
				Labels: map[string]string{
					common.OpenEverestMonitoringClassLabel: "pmm",
					common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
				},
			},
			Type: corev1.SecretTypeOpaque,
		},
	}

	builder := fake.NewClientBuilder().WithScheme(scheme)
	for i := range secrets {
		builder = builder.WithObjects(&secrets[i])
	}
	fakeClient := builder.Build()

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
			name: "no filter lists labelled secrets, whoever created them",
			want: []string{"corp-ca", "pmm-credentials", "pmm-credentials-from-git", "secret-data", "secret-stringdata"},
		},
		{
			name:   "by definition",
			filter: handlers.SecretFilter{Definition: "database-credentials"},
			want:   []string{"secret-data", "secret-stringdata"},
		},
		{
			name:   "by provider",
			filter: handlers.SecretFilter{Provider: "test-provider"},
			want:   []string{"secret-data", "secret-stringdata"},
		},
		{
			name:   "CA bundles",
			filter: handlers.SecretFilter{Definition: common.OpenEverestCABundleDefinition},
			want:   []string{"corp-ca"},
		},
		{
			name:   "by monitoring class and definition",
			filter: handlers.SecretFilter{MonitoringClass: "pmm", Definition: "ServiceAccountToken"},
			want:   []string{"pmm-credentials", "pmm-credentials-from-git"},
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

// TestSecretVerbsByManager runs every verb against Secrets created through the
// API, applied by another writer, owned by a controller, and unlabelled (spec
// 020 §4.3): the API shows every labelled Secret, patches only those it
// created and no controller owns, and deletes any labelled Secret no
// controller owns.
func TestSecretVerbsByManager(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	labels := map[string]string{
		common.OpenEverestMonitoringClassLabel: "pmm",
		common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
	}
	managed := maps.Clone(labels)
	managed[common.OpenEverestManagedLabel] = "true"
	externalSecret := metav1.OwnerReference{
		APIVersion: "external-secrets.io/v1",
		Kind:       "ExternalSecret",
		Name:       "pmm-source",
		UID:        "1111",
		Controller: new(true),
	}
	notController := metav1.OwnerReference{APIVersion: "v1", Kind: "ConfigMap", Name: "inventory", UID: "2222"}
	notFound := k8serrors.NewNotFound(corev1.Resource("secrets"), "pmm-credentials")

	errManagedByController := &secretdefinition.ManagedByControllerError{Kind: "ExternalSecret", Name: "pmm-source"}

	tests := []struct {
		name    string
		labels  map[string]string
		owners  []metav1.OwnerReference
		get     error
		patch   error
		deleted error
	}{
		{
			name:   "created through the API",
			labels: managed,
		},
		{
			name:   "created through the API, with an owner that is not its controller",
			labels: managed,
			owners: []metav1.OwnerReference{notController},
		},
		{
			name:   "applied with kubectl or by Git",
			labels: labels,
			patch:  secretdefinition.ErrManagedOutside,
		},
		{
			name:    "owned by a controller",
			labels:  labels,
			owners:  []metav1.OwnerReference{externalSecret},
			patch:   errManagedByController,
			deleted: errManagedByController,
		},
		{
			name:    "created through the API, then owned by a controller",
			labels:  managed,
			owners:  []metav1.OwnerReference{notController, externalSecret},
			patch:   errManagedByController,
			deleted: errManagedByController,
		},
		{
			name:    "unlabelled",
			get:     notFound,
			patch:   notFound,
			deleted: notFound,
		},
		{
			name:    "created through the API, then stripped of its definition",
			labels:  map[string]string{common.OpenEverestManagedLabel: "true", common.OpenEverestMonitoringClassLabel: "pmm"},
			get:     notFound,
			patch:   notFound,
			deleted: notFound,
		},
	}

	assertErr := func(t *testing.T, want, err error) {
		t.Helper()
		var wantByController *secretdefinition.ManagedByControllerError
		switch {
		case want == nil:
			require.NoError(t, err)
		case errors.As(want, &wantByController):
			var byController *secretdefinition.ManagedByControllerError
			require.ErrorAs(t, err, &byController)
			assert.Equal(t, wantByController, byController)
		case k8serrors.IsNotFound(want):
			assert.Equal(t, want, err)
		default:
			require.ErrorIs(t, err, want)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pmm-credentials",
					Namespace:       namespace,
					Labels:          tt.labels,
					OwnerReferences: tt.owners,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{"token": []byte("glsa_old")},
			}
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
			handler := &k8sHandler{
				kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(c),
				log:           zap.NewNop().Sugar(),
			}
			stored := func(t *testing.T) *corev1.Secret {
				t.Helper()
				s := &corev1.Secret{}
				require.NoError(t, c.Get(ctx, ctrlclient.ObjectKeyFromObject(secret), s))
				return s
			}

			got, err := handler.GetSecret(ctx, "", namespace, secret.Name)
			assertErr(t, tt.get, err)
			if tt.get == nil {
				assert.Equal(t, tt.owners, got.OwnerReferences, "owner references are shown")
			}

			list, err := handler.ListSecrets(ctx, "", namespace, handlers.SecretFilter{})
			require.NoError(t, err)
			if tt.get == nil {
				require.Len(t, list.Items, 1, "listed, since it can be read")
				assert.Equal(t, secret.Name, list.Items[0].Name)
			} else {
				assert.Empty(t, list.Items, "not listed, since it can't be read")
			}

			version := stored(t).ResourceVersion
			_, err = handler.PatchSecret(ctx, "", namespace, secret.Name,
				[]byte(`{"metadata": {"resourceVersion": "`+version+`"}, "stringData": {"token": "glsa_new"}}`),
			)
			assertErr(t, tt.patch, err)
			want := "glsa_new"
			if tt.patch != nil {
				want = "glsa_old"
			}
			assert.Equal(t, want, string(stored(t).Data["token"]))

			err = handler.DeleteSecret(ctx, "", namespace, secret.Name)
			assertErr(t, tt.deleted, err)
			err = c.Get(ctx, ctrlclient.ObjectKeyFromObject(secret), &corev1.Secret{})
			if tt.deleted == nil {
				assert.True(t, k8serrors.IsNotFound(err), "deleted, got %v", err)
			} else {
				require.NoError(t, err, "kept")
			}
		})
	}
}

func TestDeleteSecretNotFound(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	handler := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "test-namespace").
			WithKubernetesClient(fake.NewClientBuilder().WithScheme(scheme).Build()),
		log: zap.NewNop().Sugar(),
	}

	err := handler.DeleteSecret(context.Background(), "", "test-namespace", "missing")
	require.True(t, k8serrors.IsNotFound(err), "got %v", err)
}

// TestUnlabelledSecretAnswersAsMissing checks that the API reveals nothing
// about a Secret it does not serve: every verb answers for an unlabelled
// Secret exactly as for a missing one.
func TestUnlabelledSecretAnswersAsMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	newHandler := func(objs ...ctrlclient.Object) *k8sHandler {
		return &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).
				WithKubernetesClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()),
			log: zap.NewNop().Sugar(),
		}
	}
	unlabelled := newHandler(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db-credentials",
			Namespace: namespace,
			Labels:    map[string]string{common.OpenEverestManagedLabel: "true"},
		},
		Data: map[string][]byte{"password": []byte("s3cr3t")},
	})
	missing := newHandler()

	verbs := map[string]func(h *k8sHandler) error{
		"get": func(h *k8sHandler) error {
			_, err := h.GetSecret(ctx, "", namespace, "db-credentials")
			return err
		},
		"patch": func(h *k8sHandler) error {
			_, err := h.PatchSecret(ctx, "", namespace, "db-credentials",
				[]byte(`{"metadata": {"resourceVersion": "1"}, "stringData": {"password": "new"}}`),
			)
			return err
		},
		"delete": func(h *k8sHandler) error {
			return h.DeleteSecret(ctx, "", namespace, "db-credentials")
		},
	}

	for name, verb := range verbs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			want := verb(missing)
			require.True(t, k8serrors.IsNotFound(want), "got %v", want)
			assert.Equal(t, want, verb(unlabelled))
		})
	}
}
