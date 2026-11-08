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

package backup_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/backupstorageclasses/s3"
	"github.com/openeverest/openeverest/v2/internal/classvalidation"
	"github.com/openeverest/openeverest/v2/internal/controller/backup"
)

// storageSpecSchema is the schema of BackupStorage.spec in the CRD, with
// parameters typed by the class's parametersSchema: what self is in the
// class's validations.
func storageSpecSchema(t *testing.T, class *backupv1alpha1.BackupStorageClass) *apiextensionsv1.JSONSchemaProps {
	t.Helper()
	data, err := os.ReadFile("../../../config/crd/bases/backup.openeverest.io_backupstorages.yaml")
	require.NoError(t, err)
	crd := &apiextensionsv1.CustomResourceDefinition{}
	require.NoError(t, yaml.UnmarshalStrict(data, crd))
	require.Len(t, crd.Spec.Versions, 1)
	spec := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	require.NotNil(t, class.Spec.ParametersSchema.OpenAPIV3Schema)
	spec.Properties["parameters"] = *class.Spec.ParametersSchema.OpenAPIV3Schema
	return &spec
}

// TestS3ValidationsMatchAccept evaluates the validations generated from
// s3.StorageView against BackupStorage specs, typed by the BackupStorage CRD,
// so a view that drifts from BackupStorageSpec fails here. They must agree
// with the rules Accept applies through the s3 decoder.
func TestS3ValidationsMatchAccept(t *testing.T) {
	t.Parallel()

	class := builtinClass(t, s3.ClassName)
	require.Len(t, class.Spec.Validations, 2)
	rules, err := classvalidation.Compile(storageSpecSchema(t, class), class.Spec.Validations)
	require.NoError(t, err)

	pem := selfSignedPEM(t)
	ca := &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: "ca", Key: "ca.crt"}}}
	tests := []struct {
		name       string
		bucket     string
		endpoint   string
		tls        *common.TLSConfig
		wantReason string
	}{
		{name: "valid", bucket: "team-a-backups", endpoint: "https://s3.example.com"},
		{name: "underscore and dots", bucket: "team_a.backups", endpoint: "https://s3.example.com"},
		{name: "63 characters", bucket: strings.Repeat("a", 63), endpoint: "https://s3.example.com"},
		{name: "64 characters", bucket: strings.Repeat("a", 64), endpoint: "https://s3.example.com", wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "too short", bucket: "ab", endpoint: "https://s3.example.com", wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "uppercase", bucket: "Backups", endpoint: "https://s3.example.com", wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "double dot", bucket: "team..a", endpoint: "https://s3.example.com", wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "starts with a dash", bucket: "-backups", endpoint: "https://s3.example.com", wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "CA on https", bucket: "backups", endpoint: "https://s3.example.com", tls: ca},
		{name: "insecure on https", bucket: "backups", endpoint: "https://s3.example.com", tls: &common.TLSConfig{InsecureSkipVerify: true}},
		{name: "plain http", bucket: "backups", endpoint: "http://minio.minio.svc:9000"},
		{name: "empty tls on http", bucket: "backups", endpoint: "http://minio.minio.svc:9000", tls: &common.TLSConfig{}},
		{name: "CA on http", bucket: "backups", endpoint: "http://minio.minio.svc:9000", tls: ca, wantReason: backupv1alpha1.ReasonInvalidParameters},
		{
			name: "insecure on http", bucket: "backups", endpoint: "http://minio.minio.svc:9000",
			tls: &common.TLSConfig{InsecureSkipVerify: true}, wantReason: backupv1alpha1.ReasonInvalidParameters,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec := backupv1alpha1.BackupStorageSpec{
				BackupStorageClassRef: common.ObjectRef{Name: s3.ClassName},
				Location:              backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: tt.bucket}},
				Parameters:            raw(`{"region":"us-east-1","endpointURL":"` + tt.endpoint + `"}`),
				Authentication:        common.Authentication{Mode: s3.AuthModeAccessKey, CredentialsSecretRef: &common.SecretRef{Name: "creds"}},
				TLS:                   tt.tls,
			}
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&spec)
			require.NoError(t, err)
			var got string
			if v := rules.Evaluate(t.Context(), obj); len(v) > 0 {
				got = v[0].Reason
			}
			assert.Equal(t, tt.wantReason, got, "validations")

			verdict := backup.Accept(class, backup.AcceptInput{
				Location:       spec.Location,
				Parameters:     spec.Parameters,
				Authentication: spec.Authentication,
				TLS:            spec.TLS,
				Credentials: backup.SecretContent{Found: true, Data: map[string][]byte{
					s3.KeyAWSAccessKeyID: []byte("AKIA"), s3.KeyAWSSecretAccessKey: []byte("secret"),
				}},
				CA: backup.SecretContent{Found: true, Data: map[string][]byte{"ca.crt": pem}},
			})
			want := tt.wantReason
			if want == "" {
				want = backupv1alpha1.ReasonAccepted
			}
			assert.Equal(t, want, verdict.Reason, "Accept: %s", verdict.Message)
		})
	}
}
