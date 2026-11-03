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

package secretdefinition

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

type fakeGetter struct {
	providers map[string]*corev1alpha1.Provider
	classes   map[string]*monitoringv1alpha1.MonitoringClass
	storages  map[string]*backupv1alpha1.BackupStorageClass
	err       error
}

func (g fakeGetter) GetProvider(_ context.Context, key ctrlclient.ObjectKey) (*corev1alpha1.Provider, error) {
	if g.err != nil {
		return nil, g.err
	}
	if p, ok := g.providers[key.Name]; ok {
		return p, nil
	}
	return nil, k8serrors.NewNotFound(schema.GroupResource{Group: "core.openeverest.io", Resource: "providers"}, key.Name)
}

func (g fakeGetter) GetMonitoringClass(_ context.Context, key ctrlclient.ObjectKey) (*monitoringv1alpha1.MonitoringClass, error) {
	if g.err != nil {
		return nil, g.err
	}
	if c, ok := g.classes[key.Name]; ok {
		return c, nil
	}
	return nil, k8serrors.NewNotFound(schema.GroupResource{Group: "monitoring.openeverest.io", Resource: "monitoringclasses"}, key.Name)
}

func (g fakeGetter) GetBackupStorageClass(_ context.Context, key ctrlclient.ObjectKey) (*backupv1alpha1.BackupStorageClass, error) {
	if g.err != nil {
		return nil, g.err
	}
	if c, ok := g.storages[key.Name]; ok {
		return c, nil
	}
	return nil, k8serrors.NewNotFound(schema.GroupResource{Group: "backup.openeverest.io", Resource: "backupstorageclasses"}, key.Name)
}

func testGetter() fakeGetter {
	return fakeGetter{
		providers: map[string]*corev1alpha1.Provider{
			"psmdb": {
				ObjectMeta: metav1.ObjectMeta{Name: "psmdb"},
				Spec: corev1alpha1.ProviderSpec{
					Secrets: map[string]corev1alpha1.SecretDefinition{
						"user": {
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
			},
		},
		classes: map[string]*monitoringv1alpha1.MonitoringClass{
			"pmm": {
				ObjectMeta: metav1.ObjectMeta{Name: "pmm"},
				Spec: monitoringv1alpha1.MonitoringClassSpec{
					AuthModes: []commonv1alpha1.AuthMode{
						{
							Name: "ServiceAccountToken",
							Credentials: &commonv1alpha1.CredentialKeys{
								Required: []string{"token"},
								Optional: []string{"orgID"},
							},
						},
						{Name: commonv1alpha1.AuthModeAnonymous},
					},
				},
			},
		},
		storages: map[string]*backupv1alpha1.BackupStorageClass{
			"s3": {
				ObjectMeta: metav1.ObjectMeta{Name: "s3"},
				Spec: backupv1alpha1.BackupStorageClassSpec{
					AuthModes: []commonv1alpha1.AuthMode{
						{
							Name: "AccessKey",
							Credentials: &commonv1alpha1.CredentialKeys{
								Required: []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
							},
						},
						{Name: commonv1alpha1.AuthModeWorkloadIdentity},
					},
				},
			},
		},
	}
}

func storageClassLabels(class, definition string) map[string]string {
	return map[string]string{
		common.OpenEverestBackupStorageClassLabel: class,
		common.OpenEverestDefinitionLabel:         definition,
	}
}

func classLabels(class, definition string) map[string]string {
	return map[string]string{
		common.OpenEverestMonitoringClassLabel: class,
		common.OpenEverestDefinitionLabel:      definition,
	}
}

func providerLabels(provider, definition string) map[string]string {
	return map[string]string{
		common.OpenEverestProviderLabel:   provider,
		common.OpenEverestDefinitionLabel: definition,
	}
}

func TestRefFromLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		labels  map[string]string
		want    Ref
		wantErr string
	}{
		{
			name:   "provider",
			labels: providerLabels("psmdb", "user"),
			want:   Ref{Kind: KindProvider, Source: "psmdb", Definition: "user"},
		},
		{
			name:   "monitoring class",
			labels: classLabels("pmm", "ServiceAccountToken"),
			want:   Ref{Kind: KindMonitoringClass, Source: "pmm", Definition: "ServiceAccountToken"},
		},
		{
			name:    "no labels",
			labels:  nil,
			wantErr: "label the Secret with one of openeverest.io/provider, openeverest.io/monitoring-class, openeverest.io/backup-storage-class, and with openeverest.io/definition",
		},
		{
			name:    "definition without a source",
			labels:  map[string]string{common.OpenEverestDefinitionLabel: "user"},
			wantErr: `openeverest.io/definition "user" needs one of openeverest.io/provider, openeverest.io/monitoring-class, openeverest.io/backup-storage-class; only "ca-bundle" needs none`,
		},
		{
			name:   "backup storage class",
			labels: storageClassLabels("s3", "AccessKey"),
			want:   Ref{Kind: KindBackupStorageClass, Source: "s3", Definition: "AccessKey"},
		},
		{
			name:   "core CA bundle without a source",
			labels: map[string]string{common.OpenEverestDefinitionLabel: common.OpenEverestCABundleDefinition},
			want:   Ref{Kind: KindCore, Definition: common.OpenEverestCABundleDefinition},
		},
		{
			name:   "ca-bundle with a source is the source's definition",
			labels: providerLabels("psmdb", common.OpenEverestCABundleDefinition),
			want:   Ref{Kind: KindProvider, Source: "psmdb", Definition: common.OpenEverestCABundleDefinition},
		},
		{
			name:    "source without a definition",
			labels:  map[string]string{common.OpenEverestMonitoringClassLabel: "pmm"},
			wantErr: "missing openeverest.io/definition label",
		},
		{
			name: "two sources",
			labels: map[string]string{
				common.OpenEverestProviderLabel:        "psmdb",
				common.OpenEverestMonitoringClassLabel: "pmm",
				common.OpenEverestDefinitionLabel:      "user",
			},
			wantErr: "set only one of openeverest.io/provider, openeverest.io/monitoring-class",
		},
		{
			name: "empty source label counts as absent",
			labels: map[string]string{
				common.OpenEverestProviderLabel:        "",
				common.OpenEverestMonitoringClassLabel: "pmm",
				common.OpenEverestDefinitionLabel:      "ServiceAccountToken",
			},
			want: Ref{Kind: KindMonitoringClass, Source: "pmm", Definition: "ServiceAccountToken"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := RefFromLabels(tt.labels)
			if tt.wantErr != "" {
				require.ErrorIs(t, err, ErrInvalidLabels)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		secret  *corev1.Secret
		getter  Getter
		wantIs  error
		wantErr string
	}{
		{
			name: "monitoring class credentials with the required key",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: classLabels("pmm", "ServiceAccountToken")},
				StringData: map[string]string{"token": "glsa_token"},
			},
		},
		{
			name: "monitoring class credentials in data with an optional key",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: classLabels("pmm", "ServiceAccountToken")},
				Type:       corev1.SecretTypeOpaque,
				Data:       map[string][]byte{"token": []byte("glsa_token"), "orgID": []byte("1")},
			},
		},
		{
			name: "monitoring class credentials ignore keys the mode does not list",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: classLabels("pmm", "ServiceAccountToken")},
				StringData: map[string]string{"token": "glsa_token", "note": "rotated"},
			},
		},
		{
			name: "monitoring class credentials missing a required key",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: classLabels("pmm", "ServiceAccountToken")},
				StringData: map[string]string{"orgID": "1"},
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `missing key "token" required by authentication mode "ServiceAccountToken"`,
		},
		{
			name: "monitoring class credentials with an empty required key",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: classLabels("pmm", "ServiceAccountToken")},
				StringData: map[string]string{"token": ""},
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `missing key "token"`,
		},
		{
			name: "monitoring class without the mode",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: classLabels("pmm", "BasicAuth")},
				StringData: map[string]string{"username": "u", "password": "p"},
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `MonitoringClass "pmm" has no authentication mode "BasicAuth"; it supports ServiceAccountToken, Anonymous`,
		},
		{
			name: "monitoring class reserved mode takes no Secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: classLabels("pmm", commonv1alpha1.AuthModeAnonymous)},
				StringData: map[string]string{"token": "glsa_token"},
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `authentication mode "Anonymous" of MonitoringClass "pmm" takes no Secret`,
		},
		{
			name: "monitoring class credentials must be Opaque",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: classLabels("pmm", "ServiceAccountToken")},
				Type:       corev1.SecretTypeBasicAuth,
				StringData: map[string]string{"token": "glsa_token", "username": "u"},
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `credentials of MonitoringClass "pmm" must be of type "Opaque", not "kubernetes.io/basic-auth"`,
		},
		{
			name: "unknown monitoring class",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: classLabels("datadog", "ApiKey")},
				StringData: map[string]string{"apiKey": "k"},
			},
			wantIs:  ErrSourceNotFound,
			wantErr: `MonitoringClass "datadog"`,
		},
		{
			name: "backup storage class credentials with the required keys",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: storageClassLabels("s3", "AccessKey")},
				StringData: map[string]string{"AWS_ACCESS_KEY_ID": "AKIA", "AWS_SECRET_ACCESS_KEY": "shh"},
			},
		},
		{
			name: "backup storage class credentials missing a required key",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: storageClassLabels("s3", "AccessKey")},
				StringData: map[string]string{"AWS_ACCESS_KEY_ID": "AKIA"},
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `missing key "AWS_SECRET_ACCESS_KEY" required by authentication mode "AccessKey"`,
		},
		{
			name: "backup storage class without the mode",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: storageClassLabels("s3", "AccountKey")},
				StringData: map[string]string{"AZURE_STORAGE_ACCOUNT_KEY": "k"},
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `BackupStorageClass "s3" has no authentication mode "AccountKey"; it supports AccessKey, WorkloadIdentity`,
		},
		{
			name: "backup storage class reserved mode takes no Secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: storageClassLabels("s3", commonv1alpha1.AuthModeWorkloadIdentity)},
				StringData: map[string]string{"token": "t"},
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `authentication mode "WorkloadIdentity" of BackupStorageClass "s3" takes no Secret`,
		},
		{
			name: "unknown backup storage class",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: storageClassLabels("azure", "AccountKey")},
				StringData: map[string]string{"AZURE_STORAGE_ACCOUNT_KEY": "k"},
			},
			wantIs:  ErrSourceNotFound,
			wantErr: `BackupStorageClass "azure"`,
		},
		{
			name: "provider secret matching its definition",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: providerLabels("psmdb", "user")},
				StringData: map[string]string{"username": "admin", "password": "secret"},
			},
		},
		{
			name: "provider secret not matching its schema",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: providerLabels("psmdb", "user")},
				StringData: map[string]string{"username": "admin"},
			},
			wantIs:  ErrInvalidSecret,
			wantErr: "secret does not conform to schema",
		},
		{
			name: "provider without the definition",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: providerLabels("psmdb", "splithorizon-tls")},
				StringData: map[string]string{"tls.crt": "c"},
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `secret definition "splithorizon-tls" does not exist`,
		},
		{
			name: "unknown provider",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: providerLabels("mariadb", "user")},
				StringData: map[string]string{"username": "admin", "password": "secret"},
			},
			wantIs:  ErrSourceNotFound,
			wantErr: `Provider "mariadb"`,
		},
		{
			name: "service account token Secrets are refused",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      classLabels("pmm", "ServiceAccountToken"),
					Annotations: map[string]string{corev1.ServiceAccountNameKey: "default"},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `Secrets of type "kubernetes.io/service-account-token" cannot be created`,
		},
		{
			name: "bootstrap token Secrets are refused",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: providerLabels("psmdb", "user")},
				Type:       corev1.SecretTypeBootstrapToken,
			},
			wantIs:  ErrInvalidSecret,
			wantErr: `Secrets of type "bootstrap.kubernetes.io/token" cannot be created`,
		},
		{
			name: "missing labels",
			secret: &corev1.Secret{
				StringData: map[string]string{"token": "glsa_token"},
			},
			wantIs: ErrInvalidLabels,
		},
		{
			name: "a failing lookup is not a validation error",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: classLabels("pmm", "ServiceAccountToken")},
				StringData: map[string]string{"token": "glsa_token"},
			},
			getter:  fakeGetter{err: errors.New("connection refused")},
			wantErr: `get MonitoringClass "pmm": connection refused`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			getter := tt.getter
			if getter == nil {
				getter = testGetter()
			}

			err := Validate(context.Background(), getter, tt.secret)
			if tt.wantIs == nil && tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			if tt.wantIs != nil {
				require.ErrorIs(t, err, tt.wantIs)
			} else {
				for _, sentinel := range []error{ErrInvalidLabels, ErrSourceNotFound, ErrInvalidSecret} {
					require.NotErrorIs(t, err, sentinel)
				}
			}
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateNeverEchoesValues(t *testing.T) {
	t.Parallel()

	const value = "glsa_do-not-print-me"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Labels: classLabels("pmm", "ServiceAccountToken")},
		Type:       corev1.SecretTypeBasicAuth,
		StringData: map[string]string{"username": value},
	}

	err := Validate(context.Background(), testGetter(), secret)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), value)
}

// testCertificatePEM returns a self-signed CA certificate in PEM.
func testCertificatePEM(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestValidateCABundle(t *testing.T) {
	t.Parallel()

	ca := testCertificatePEM(t)
	privateKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("do-not-print-me")})
	caBundle := func(data map[string][]byte) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
				common.OpenEverestDefinitionLabel: common.OpenEverestCABundleDefinition,
			}},
			Data: data,
		}
	}

	tests := []struct {
		name    string
		secret  *corev1.Secret
		wantErr string
	}{
		{
			name:   "one key with a certificate",
			secret: caBundle(map[string][]byte{"ca.crt": ca}),
		},
		{
			name:   "several keys, one with two certificates",
			secret: caBundle(map[string][]byte{"ca.crt": ca, "bundle.pem": append(append([]byte{}, ca...), ca...)}),
		},
		{
			name: "certificate in stringData",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					common.OpenEverestDefinitionLabel: common.OpenEverestCABundleDefinition,
				}},
				Type:       corev1.SecretTypeOpaque,
				StringData: map[string]string{"ca.crt": string(ca)},
			},
		},
		{
			name:    "no keys",
			secret:  caBundle(nil),
			wantErr: "a CA bundle needs at least one key",
		},
		{
			name:    "a key without a certificate",
			secret:  caBundle(map[string][]byte{"ca.crt": ca, "notes": []byte("rotated in May")}),
			wantErr: `key "notes": no PEM certificate found`,
		},
		{
			name:    "a private key next to the certificate",
			secret:  caBundle(map[string][]byte{"ca.crt": append(append([]byte{}, ca...), privateKey...)}),
			wantErr: `key "ca.crt" holds a "PRIVATE KEY" PEM block; a CA bundle holds only certificates`,
		},
		{
			name: "not Opaque",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					common.OpenEverestDefinitionLabel: common.OpenEverestCABundleDefinition,
				}},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{"tls.crt": ca, "tls.key": privateKey},
			},
			wantErr: `a CA bundle must be of type "Opaque", not "kubernetes.io/tls"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// A core definition reads no Provider or class.
			err := Validate(context.Background(), fakeGetter{err: errors.New("unexpected lookup")}, tt.secret)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, ErrInvalidSecret)
			require.ErrorContains(t, err, tt.wantErr)
			assert.NotContains(t, err.Error(), "do-not-print-me")
		})
	}
}
