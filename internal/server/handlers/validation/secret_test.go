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
	"fmt"
	"maps"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// testCA is a self-signed CA certificate whose private key was discarded.
const testCA = `-----BEGIN CERTIFICATE-----
MIIBrDCCAVOgAwIBAgIUejvO8eGXCMbLRlBEfeQEBiosadMwCgYIKoZIzj0EAwIw
IzEhMB8GA1UEAwwYb3BlbmV2ZXJlc3QgYXBpLXRlc3RzIENBMCAXDTI2MTEwMzE3
MDQxMloYDzIxMjYxMDEwMTcwNDEyWjAjMSEwHwYDVQQDDBhvcGVuZXZlcmVzdCBh
cGktdGVzdHMgQ0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQqq+Kp1HErWpL/
1n0yHhczVyLmSvTDwIyP1rEsT3UIs/qglJbhQV+fo8kJGGJQMS3Jv30eQuZuBdJ7
yk+IDeK3o2MwYTAdBgNVHQ4EFgQUb8ftw05oROj/5tA/eVtG+Twr6SEwHwYDVR0j
BBgwFoAUb8ftw05oROj/5tA/eVtG+Twr6SEwDwYDVR0TAQH/BAUwAwEB/zAOBgNV
HQ8BAf8EBAMCAgQwCgYIKoZIzj0EAwIDRwAwRAIgSIr4/5J7cFJB2zaDx34qdFlU
jmb44BMPZLYDV0lCycICIFsNRStO8nvLnG+ZVfaBva8Bfoj28ZJTEgQexukz41yl
-----END CERTIFICATE-----
`

func TestCreateSecret_Validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	cluster := "test-cluster"

	// Provider with secret definition.
	provider := &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-provider",
		},
		Spec: corev1alpha1.ProviderSpec{
			Secrets: map[string]corev1alpha1.SecretDefinition{
				"database-credentials": {
					ParametersSchema: &commonv1alpha1.ParametersSchema{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"username": {Type: "string"},
								"password": {Type: "string"},
							},
							Required: []string{"username", "password"},
						},
					},
				},
			},
		},
	}

	// MonitoringClass whose mode needs a token.
	class := &monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pmm",
		},
		Spec: monitoringv1alpha1.MonitoringClassSpec{
			AuthModes: []commonv1alpha1.AuthMode{{
				Name:        "ServiceAccountToken",
				Credentials: &commonv1alpha1.CredentialKeys{Required: []string{"token"}},
			}},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))
	require.NoError(t, monitoringv1alpha1.AddToScheme(scheme))

	providerLabels := map[string]string{
		common.OpenEverestDefinitionLabel: "database-credentials",
		common.OpenEverestProviderLabel:   "test-provider",
	}
	classLabels := map[string]string{
		common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
		common.OpenEverestMonitoringClassLabel: "pmm",
	}
	userCredentials := map[string]string{
		"username": "admin",
		"password": "secret123",
	}

	tests := []struct {
		name   string
		secret *corev1.Secret
		err    string
	}{
		{
			name: "provider secret matching its definition is created",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "db-credentials", Labels: providerLabels},
				Type:       corev1.SecretTypeOpaque,
				StringData: userCredentials,
			},
		},
		{
			name: "monitoring class credentials with a long name are created",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "monitoring-destination-pmm-prod-eu-west-credentials", Labels: classLabels},
				StringData: map[string]string{"token": "glsa_token"},
			},
		},
		{
			name: "missing provider label fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-provider-label",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
					},
				},
				Type:       corev1.SecretTypeOpaque,
				StringData: userCredentials,
			},
			err: `openeverest.io/definition "database-credentials" needs one of openeverest.io/provider, openeverest.io/monitoring-class`,
		},
		{
			name: "missing definition label fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "no-definition-label",
					Labels: map[string]string{common.OpenEverestProviderLabel: "test-provider"},
				},
				Type:       corev1.SecretTypeOpaque,
				StringData: map[string]string{"key": "value"},
			},
			err: "missing openeverest.io/definition label",
		},
		{
			name: "secret without labels fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "no-labels"},
				StringData: map[string]string{"token": "glsa_token"},
			},
			err: "label the Secret with one of",
		},
		{
			name: "secret not conforming to schema fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "empty-secret", Labels: providerLabels},
				Type:       corev1.SecretTypeOpaque,
			},
			err: "secret does not conform to schema",
		},
		{
			name: "monitoring class credentials missing a required key fail",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "pmm-credentials", Labels: classLabels},
				StringData: map[string]string{"apiKey": "k"},
			},
			err: `missing key "token" required by authentication mode "ServiceAccountToken"`,
		},
		{
			name: "unknown monitoring class fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "datadog-credentials",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel:      "ApiKey",
						common.OpenEverestMonitoringClassLabel: "datadog",
					},
				},
				StringData: map[string]string{"apiKey": "k"},
			},
			err: `definition source not found: MonitoringClass "datadog"`,
		},
		{
			name: "service account token secret fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pmm-credentials",
					Labels:      classLabels,
					Annotations: map[string]string{corev1.ServiceAccountNameKey: "default"},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
			err: `Secrets of type "kubernetes.io/service-account-token" cannot be created`,
		},
		{
			name: "CA bundle without a source label is created",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "backup-storage-minio-ca",
					Labels: map[string]string{common.OpenEverestDefinitionLabel: common.OpenEverestCABundleDefinition},
				},
				StringData: map[string]string{"ca.crt": testCA},
			},
		},
		{
			name: "CA bundle without a certificate fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "backup-storage-minio-ca",
					Labels: map[string]string{common.OpenEverestDefinitionLabel: common.OpenEverestCABundleDefinition},
				},
				StringData: map[string]string{"ca.crt": "not a certificate"},
			},
			err: `key "ca.crt": no PEM certificate found`,
		},
		{
			name: "CA bundle without ca.crt fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "backup-storage-minio-ca",
					Labels: map[string]string{common.OpenEverestDefinitionLabel: common.OpenEverestCABundleDefinition},
				},
				StringData: map[string]string{"bundle.pem": testCA},
			},
			err: `missing key "ca.crt"`,
		},
		{
			name: "invalid secret name fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "Invalid_Name_With_Underscore", Labels: providerLabels},
				Type:       corev1.SecretTypeOpaque,
				StringData: userCredentials,
			},
			err: `invalid name "Invalid_Name_With_Underscore"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Use DeepCopy to avoid race conditions since the fake client
			// modifies the objects' ResourceVersion during Build().
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(provider.DeepCopy(), class.DeepCopy()).Build()
			kubeConnector := kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient)

			// Create mock next handler.
			mockNext := &handlers.MockHandler{}
			mockNext.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(tt.secret, nil)

			// Create validation handler.
			handler := &validateHandler{
				log:           zap.NewNop().Sugar(),
				kubeConnector: kubeConnector,
				next:          mockNext,
			}

			_, err := handler.CreateSecret(ctx, cluster, namespace, tt.secret)
			if tt.err == "" {
				require.NoError(t, err)
				mockNext.AssertCalled(t, "CreateSecret", mock.Anything, cluster, namespace, tt.secret)
				return
			}
			require.ErrorIs(t, err, ErrInvalidRequest)
			require.ErrorContains(t, err, tt.err)
			mockNext.AssertNotCalled(t, "CreateSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestPatchSecret_Validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	cluster := "test-cluster"

	class := &monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pmm"},
		Spec: monitoringv1alpha1.MonitoringClassSpec{
			AuthModes: []commonv1alpha1.AuthMode{{
				Name:        "ServiceAccountToken",
				Credentials: &commonv1alpha1.CredentialKeys{Required: []string{"token"}},
			}},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, monitoringv1alpha1.AddToScheme(scheme))

	storedLabels := map[string]string{
		common.OpenEverestManagedLabel:         "true",
		common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
		common.OpenEverestMonitoringClassLabel: "pmm",
	}
	credentials := func() *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: namespace, Labels: maps.Clone(storedLabels)},
			Type:       corev1.SecretTypeOpaque,
			Data:       map[string][]byte{"token": []byte("glsa_old"), "orgID": []byte("1")},
		}
	}
	caBundle := func() *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: namespace, Labels: map[string]string{
				common.OpenEverestManagedLabel:    "true",
				common.OpenEverestDefinitionLabel: "ca-bundle",
			}},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"ca.crt": []byte(testCA)},
		}
	}
	errNext := errors.New("next handler")

	tests := []struct {
		name   string
		stored *corev1.Secret
		// patch gets the stored resourceVersion in place of %s.
		patch string
		err   string
	}{
		{
			name:  "a rotated key is accepted",
			patch: `{"metadata": {"resourceVersion": "%s"}, "stringData": {"token": "glsa_new"}}`,
		},
		{
			name:  "keys left out are kept, so a required one need not be sent",
			patch: `{"metadata": {"resourceVersion": "%s"}, "stringData": {"orgID": "2"}}`,
		},
		{
			name: "a Secret returned by GET is accepted",
			patch: `{"metadata": {"name": "credentials", "resourceVersion": "%s", "labels": {
				"openeverest.io/managed": "true",
				"openeverest.io/definition": "ServiceAccountToken",
				"openeverest.io/monitoring-class": "pmm"
			}}, "type": "Opaque"}`,
		},
		{
			name: "the labels the Secret was created with are accepted",
			patch: `{"metadata": {"resourceVersion": "%s", "labels": {
				"openeverest.io/definition": "ServiceAccountToken",
				"openeverest.io/monitoring-class": "pmm"
			}}, "stringData": {"token": "glsa_new"}}`,
		},
		{
			name:  "removing a required key is refused",
			patch: `{"metadata": {"resourceVersion": "%s"}, "data": {"token": null}}`,
			err:   `missing key "token"`,
		},
		{
			name:  "a changed definition label is refused",
			patch: `{"metadata": {"resourceVersion": "%s", "labels": {"openeverest.io/definition": "BasicAuth"}}}`,
			err:   `label "openeverest.io/definition" differs from the stored Secret`,
		},
		{
			name:  "a removed label is refused",
			patch: `{"metadata": {"resourceVersion": "%s", "labels": {"openeverest.io/managed": null}}}`,
			err:   `label "openeverest.io/managed" differs from the stored Secret`,
		},
		{
			name:  "an added label is refused",
			patch: `{"metadata": {"resourceVersion": "%s", "labels": {"team": "a"}}}`,
			err:   `label "team" differs from the stored Secret`,
		},
		{
			name:  "a changed type is refused",
			patch: `{"metadata": {"resourceVersion": "%s"}, "type": "kubernetes.io/basic-auth"}`,
			err:   `type can't change from "Opaque" to "kubernetes.io/basic-auth"`,
		},
		{
			name:  "making the Secret immutable is refused",
			patch: `{"metadata": {"resourceVersion": "%s"}, "immutable": true}`,
			err:   "immutable can't change",
		},
		{
			name:  "a patch without a resourceVersion is refused",
			patch: `{"stringData": {"token": "glsa_new"}}`,
			err:   "metadata.resourceVersion is required",
		},
		{
			name:   "a CA bundle is checked against the core definition",
			stored: caBundle(),
			patch:  `{"metadata": {"resourceVersion": "%s"}, "stringData": {"ca.crt": "not a certificate"}}`,
			err:    `key "ca.crt": no PEM certificate found`,
		},
		{
			name:   "a valid CA bundle is accepted",
			stored: caBundle(),
			patch:  `{"metadata": {"resourceVersion": "%s"}, "stringData": {"extra.crt": ` + strconv.Quote(testCA) + `}}`,
		},
		{
			name:   "removing ca.crt from a CA bundle is refused",
			stored: caBundle(),
			patch:  `{"metadata": {"resourceVersion": "%s"}, "data": {"ca.crt": null}, "stringData": {"bundle.pem": ` + strconv.Quote(testCA) + `}}`,
			err:    `missing key "ca.crt"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stored := tt.stored
			if stored == nil {
				stored = credentials()
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(class.DeepCopy(), stored).Build()
			current := &corev1.Secret{}
			require.NoError(t, fakeClient.Get(ctx, ctrlclient.ObjectKeyFromObject(stored), current))
			patch := []byte(tt.patch)
			if strings.Contains(tt.patch, "%s") {
				patch = fmt.Appendf(nil, tt.patch, current.ResourceVersion)
			}

			mockNext := &handlers.MockHandler{}
			mockNext.On("PatchSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(current, nil)
			handler := &validateHandler{
				log:           zap.NewNop().Sugar(),
				kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
				next:          mockNext,
			}

			_, err := handler.PatchSecret(ctx, cluster, namespace, "credentials", patch)
			if tt.err == "" {
				require.NoError(t, err)
				mockNext.AssertCalled(t, "PatchSecret", mock.Anything, cluster, namespace, "credentials", patch)
				return
			}
			require.ErrorIs(t, err, ErrInvalidRequest)
			require.ErrorContains(t, err, tt.err)
			assert.NotContains(t, err.Error(), "glsa_", "errors name keys, never values")
			mockNext.AssertNotCalled(t, "PatchSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}

	t.Run("a stale resourceVersion is a conflict", func(t *testing.T) {
		t.Parallel()

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(class.DeepCopy(), credentials()).Build()
		mockNext := &handlers.MockHandler{}
		handler := &validateHandler{
			log:           zap.NewNop().Sugar(),
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
			next:          mockNext,
		}

		// The stale patch would also remove a required key: the conflict is reported first.
		_, err := handler.PatchSecret(ctx, cluster, namespace, "credentials",
			[]byte(`{"metadata": {"resourceVersion": "1"}, "data": {"token": null}}`),
		)
		require.True(t, k8serrors.IsConflict(err), "got %v", err)
		mockNext.AssertNotCalled(t, "PatchSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("a Secret the API does not manage is left to the next handler", func(t *testing.T) {
		t.Parallel()

		unmanaged := credentials()
		unmanaged.Labels = nil
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(unmanaged).Build()
		mockNext := &handlers.MockHandler{}
		mockNext.On("PatchSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errNext)
		handler := &validateHandler{
			log:           zap.NewNop().Sugar(),
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
			next:          mockNext,
		}

		_, err := handler.PatchSecret(ctx, cluster, namespace, "credentials",
			[]byte(`{"metadata": {"resourceVersion": "1"}, "data": {"token": null}}`),
		)
		require.ErrorIs(t, err, errNext)
	})

	t.Run("a missing Secret is not found", func(t *testing.T) {
		t.Parallel()

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		mockNext := &handlers.MockHandler{}
		handler := &validateHandler{
			log:           zap.NewNop().Sugar(),
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
			next:          mockNext,
		}

		_, err := handler.PatchSecret(ctx, cluster, namespace, "missing",
			[]byte(`{"metadata": {"resourceVersion": "1"}, "stringData": {"token": "glsa_new"}}`),
		)
		require.True(t, k8serrors.IsNotFound(err), "got %v", err)
		mockNext.AssertNotCalled(t, "PatchSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}
