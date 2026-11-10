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

func TestUpdateSecret(t *testing.T) {
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
			Data: map[string][]byte{"token": []byte("old"), "orgID": []byte("1")},
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

	t.Run("replaces the data and keeps the rest of the stored Secret", func(t *testing.T) {
		t.Parallel()

		handler, c := newHandler(stored("pmm-credentials", labels))
		result, err := handler.UpdateSecret(ctx, "", namespace, "pmm-credentials", &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "other-name",
				Labels:          map[string]string{"team": "a"},
				Annotations:     map[string]string{"example.com/note": "replaced"},
				OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ConfigMap", Name: "owner", UID: "1"}},
			},
			Type: corev1.SecretTypeBasicAuth,
			Data: map[string][]byte{"token": []byte("new")},
		})
		require.NoError(t, err)
		assert.Nil(t, result.Data)
		assert.Nil(t, result.StringData)

		got := get(t, c, "pmm-credentials")
		assert.Equal(t, map[string][]byte{"token": []byte("new")}, got.Data, "keys left out are removed")
		assert.Equal(t, labels, got.Labels)
		assert.Equal(t, map[string]string{"example.com/note": "kept"}, got.Annotations)
		assert.Empty(t, got.OwnerReferences)
		assert.Equal(t, corev1.SecretTypeOpaque, got.Type)
	})

	t.Run("updates with the resourceVersion the client sent", func(t *testing.T) {
		t.Parallel()

		handler, c := newHandler(stored("pmm-credentials", labels))
		current := get(t, c, "pmm-credentials").ResourceVersion

		_, err := handler.UpdateSecret(ctx, "", namespace, "pmm-credentials", &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{ResourceVersion: current},
			Data:       map[string][]byte{"token": []byte("new")},
		})
		require.NoError(t, err)

		_, err = handler.UpdateSecret(ctx, "", namespace, "pmm-credentials", &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{ResourceVersion: current},
			Data:       map[string][]byte{"token": []byte("newer")},
		})
		require.True(t, k8serrors.IsConflict(err), "a stale resourceVersion must be refused, got %v", err)
		assert.Equal(t, []byte("new"), get(t, c, "pmm-credentials").Data["token"])
	})

	t.Run("refuses a Secret the API does not manage", func(t *testing.T) {
		t.Parallel()

		handler, c := newHandler(stored("unmanaged", nil))
		_, err := handler.UpdateSecret(ctx, "", namespace, "unmanaged", &corev1.Secret{
			Data: map[string][]byte{"token": []byte("new")},
		})
		require.ErrorIs(t, err, ErrNotFound)
		assert.Equal(t, []byte("old"), get(t, c, "unmanaged").Data["token"])
	})

	t.Run("reports a missing Secret as not found", func(t *testing.T) {
		t.Parallel()

		handler, _ := newHandler()
		_, err := handler.UpdateSecret(ctx, "", namespace, "missing", &corev1.Secret{
			Data: map[string][]byte{"token": []byte("new")},
		})
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
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&secrets[0], &secrets[1], &secrets[2], &secrets[3]).
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
			want: []string{"pmm-credentials", "secret-data", "secret-stringdata"},
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
