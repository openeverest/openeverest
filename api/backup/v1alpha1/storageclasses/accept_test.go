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

package storageclasses_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// builtinClass loads a generated class from config/backupstorageclasses.
func builtinClass(t *testing.T, name string) *backupv1alpha1.BackupStorageClass {
	t.Helper()
	data, err := os.ReadFile("../../../../config/backupstorageclasses/" + name + ".yaml") //nolint:gosec // a fixture of this repository
	require.NoError(t, err)
	class := &backupv1alpha1.BackupStorageClass{}
	require.NoError(t, yaml.UnmarshalStrict(data, class))
	return class
}

func selfSignedPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestAccept(t *testing.T) {
	t.Parallel()

	s3Class := builtinClass(t, storageclasses.ClassS3)
	ca := selfSignedPEM(t)

	thirdParty := &backupv1alpha1.BackupStorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "example.test.openeverest.io"},
		Spec: backupv1alpha1.BackupStorageClassSpec{
			ParametersSchema: common.ParametersSchema{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
				Type:       "object",
				Required:   []string{"endpoint"},
				Properties: map[string]apiextensionsv1.JSONSchemaProps{"endpoint": {Type: "string"}},
			}},
			AuthModes: []common.AuthMode{{Name: "Token", Credentials: &common.CredentialKeys{Required: []string{"token"}}}},
		},
	}
	withBucketParam := thirdParty.DeepCopy()
	withBucketParam.Spec.ParametersSchema.OpenAPIV3Schema.Properties["bucket"] = apiextensionsv1.JSONSchemaProps{Type: "string"}

	accessKey := common.Authentication{Mode: storageclasses.AuthModeAccessKey, CredentialsSecretRef: &common.SecretRef{Name: "creds"}}
	keys := storageclasses.SecretContent{Found: true, Data: map[string][]byte{
		storageclasses.KeyAWSAccessKeyID:     []byte("AKIA"),
		storageclasses.KeyAWSSecretAccessKey: []byte("secret"),
	}}
	caTLS := &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: "corp-ca", Key: "ca.crt"}}}
	valid := storageclasses.AcceptInput{
		Location:       backupv1alpha1.BackupStorageLocation{Bucket: "team-a-backups", Prefix: "everest"},
		Parameters:     raw(`{"region":"us-east-1","endpointURL":"https://minio.example.com","implementation":"minio"}`),
		Authentication: accessKey,
		TLS:            caTLS,
		Credentials:    keys,
		CA:             storageclasses.SecretContent{Found: true, Data: map[string][]byte{"ca.crt": ca}},
	}
	with := func(f func(*storageclasses.AcceptInput)) storageclasses.AcceptInput {
		in := valid
		f(&in)
		return in
	}

	tests := []struct {
		name       string
		class      *backupv1alpha1.BackupStorageClass
		in         storageclasses.AcceptInput
		wantReason string
	}{
		{name: "valid s3 storage", class: s3Class, in: valid, wantReason: backupv1alpha1.ReasonAccepted},
		{
			name:  "valid third-party storage, no built-in naming rules",
			class: thirdParty,
			in: storageclasses.AcceptInput{
				Location:       backupv1alpha1.BackupStorageLocation{Bucket: "Any_Container"},
				Parameters:     raw(`{"endpoint":"tcp://store"}`),
				Authentication: common.Authentication{Mode: "Token", CredentialsSecretRef: &common.SecretRef{Name: "creds"}},
				Credentials:    storageclasses.SecretContent{Found: true, Data: map[string][]byte{"token": []byte("t")}},
			},
			wantReason: backupv1alpha1.ReasonAccepted,
		},
		{name: "class declares the bucket as a parameter", class: withBucketParam, in: valid, wantReason: backupv1alpha1.ReasonInvalidClass},
		{name: "required parameter missing", class: s3Class, in: with(func(in *storageclasses.AcceptInput) { in.Parameters = raw(`{"region":"us-east-1"}`) }), wantReason: backupv1alpha1.ReasonInvalidParameters},
		{name: "no parameters at all", class: s3Class, in: with(func(in *storageclasses.AcceptInput) { in.Parameters = nil }), wantReason: backupv1alpha1.ReasonInvalidParameters},
		{name: "unknown parameter", class: s3Class, in: with(func(in *storageclasses.AcceptInput) {
			in.Parameters = raw(`{"region":"us-east-1","endpointURL":"https://s3","verifyTLS":false}`)
		}), wantReason: backupv1alpha1.ReasonInvalidParameters},
		{name: "implementation outside the enum", class: s3Class, in: with(func(in *storageclasses.AcceptInput) {
			in.Parameters = raw(`{"region":"us-east-1","endpointURL":"https://s3","implementation":"wasabi"}`)
		}), wantReason: backupv1alpha1.ReasonInvalidParameters},
		{name: "tls on a plain HTTP endpoint", class: s3Class, in: with(func(in *storageclasses.AcceptInput) {
			in.Parameters = raw(`{"region":"us-east-1","endpointURL":"http://minio.minio.svc"}`)
		}), wantReason: backupv1alpha1.ReasonInvalidParameters},
		{name: "bucket breaks the S3 naming rules", class: s3Class, in: with(func(in *storageclasses.AcceptInput) { in.Location.Bucket = "Team_A" }), wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "mode the class does not define", class: s3Class, in: with(func(in *storageclasses.AcceptInput) { in.Authentication.Mode = "SASToken" }), wantReason: backupv1alpha1.ReasonAuthModeUnsupported},
		{name: "credentials Secret missing", class: s3Class, in: with(func(in *storageclasses.AcceptInput) { in.Credentials = storageclasses.SecretContent{} }), wantReason: backupv1alpha1.ReasonCredentialsNotFound},
		{name: "credentials key missing", class: s3Class, in: with(func(in *storageclasses.AcceptInput) {
			in.Credentials = storageclasses.SecretContent{Found: true, Data: map[string][]byte{storageclasses.KeyAWSAccessKeyID: []byte("AKIA")}}
		}), wantReason: backupv1alpha1.ReasonCredentialsNotFound},
		{name: "CA Secret missing", class: s3Class, in: with(func(in *storageclasses.AcceptInput) { in.CA = storageclasses.SecretContent{} }), wantReason: backupv1alpha1.ReasonInvalidCABundle},
		{name: "CA key not PEM", class: s3Class, in: with(func(in *storageclasses.AcceptInput) {
			in.CA = storageclasses.SecretContent{Found: true, Data: map[string][]byte{"ca.crt": []byte("not pem")}}
		}), wantReason: backupv1alpha1.ReasonInvalidCABundle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := storageclasses.Accept(tt.class, tt.in)
			assert.Equal(t, tt.wantReason, v.Reason, v.Message)
			assert.Equal(t, tt.wantReason == backupv1alpha1.ReasonAccepted, v.Accepted())
			assert.NotEmpty(t, v.Message)
		})
	}
}
