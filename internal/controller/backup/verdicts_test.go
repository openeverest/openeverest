// everest
// Copyright (C) 2023 Percona LLC
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

package backup

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

func verdictClass(name string, gen int64, entry backupv1alpha1.BackupStorageClassSupport, accepted ...metav1.Condition) *backupv1alpha1.BackupClass {
	return &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: "uid-class", Generation: gen},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:        backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged:      &backupv1alpha1.ProviderManagedSpec{ProviderRef: common.ObjectRef{Name: "mongo"}},
			BackupStorageClasses: []backupv1alpha1.BackupStorageClassSupport{entry},
		},
		Status: backupv1alpha1.BackupClassStatus{Conditions: accepted},
	}
}

func acceptedAt(gen int64, status metav1.ConditionStatus, reason string) metav1.Condition {
	return metav1.Condition{Type: backupv1alpha1.BackupClassConditionAccepted, Status: status, Reason: reason, Message: "from the executor", ObservedGeneration: gen}
}

func verdictStorage(params string, tls *common.TLSConfig) *backupv1alpha1.BackupStorage {
	return &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: "store", Namespace: "ns", Generation: 4},
		Spec: backupv1alpha1.BackupStorageSpec{
			BackupStorageClassRef: common.ObjectRef{Name: "s3"},
			Location:              backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "backups"}},
			Parameters:            &runtime.RawExtension{Raw: []byte(params)},
			Authentication:        common.Authentication{Mode: "AccessKey", CredentialsSecretRef: &common.SecretRef{Name: "creds"}},
			TLS:                   tls,
		},
	}
}

func TestStorageVerdict(t *testing.T) {
	t.Parallel()

	noGCS := &common.ParametersSchema{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensionsv1.JSONSchemaProps{
			"region":         {Type: "string"},
			"implementation": {Type: "string", Enum: []apiextensionsv1.JSON{{Raw: []byte(`"aws"`)}, {Raw: []byte(`"minio"`)}}},
		},
	}}
	s3Entry := backupv1alpha1.BackupStorageClassSupport{Name: "s3", AuthModes: []string{"AccessKey"}}
	narrowed := s3Entry
	narrowed.ParametersSchema = noGCS
	caOnly := s3Entry
	caOnly.TLS = &backupv1alpha1.BackupStorageTLSSupport{CustomCA: backupv1alpha1.TLSOptionSupported, InsecureSkipVerify: backupv1alpha1.TLSOptionNone}
	ca := &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: "ca", Key: "ca.crt"}}}
	accepted := metav1.Condition{Type: backupv1alpha1.BackupStorageConditionAccepted, Status: metav1.ConditionTrue, Reason: backupv1alpha1.ReasonAccepted}
	notAccepted := metav1.Condition{Type: backupv1alpha1.BackupStorageConditionAccepted, Status: metav1.ConditionFalse, Reason: backupv1alpha1.ReasonCredentialsNotFound}

	tests := []struct {
		name       string
		entry      backupv1alpha1.BackupStorageClassSupport
		params     string
		tls        *common.TLSConfig
		accepted   metav1.Condition
		want       backupv1alpha1.Compatibility
		wantReason string
	}{
		{name: "compatible", entry: s3Entry, params: `{"region":"r"}`, accepted: accepted, want: backupv1alpha1.CompatibilityCompatible},
		{name: "storage not accepted", entry: s3Entry, params: `{}`, accepted: notAccepted, want: backupv1alpha1.CompatibilityPending, wantReason: backupv1alpha1.ReasonStorageNotAccepted},
		{
			name: "storage class not listed", entry: backupv1alpha1.BackupStorageClassSupport{Name: "swift.example.com", AuthModes: []string{"Token"}},
			params: `{}`, accepted: accepted, want: backupv1alpha1.CompatibilityIncompatible, wantReason: backupv1alpha1.ReasonBackupStorageClassUnsupported,
		},
		{
			name: "auth mode not listed", entry: backupv1alpha1.BackupStorageClassSupport{Name: "s3", AuthModes: []string{"WorkloadIdentity"}},
			params: `{}`, accepted: accepted, want: backupv1alpha1.CompatibilityIncompatible, wantReason: backupv1alpha1.ReasonAuthModeUnsupported,
		},
		{name: "narrowed parameter supported", entry: narrowed, params: `{"region":"r","implementation":"minio"}`, accepted: accepted, want: backupv1alpha1.CompatibilityCompatible},
		{
			name: "narrowed parameter value", entry: narrowed, params: `{"region":"r","implementation":"gcs"}`, accepted: accepted,
			want: backupv1alpha1.CompatibilityIncompatible, wantReason: backupv1alpha1.ReasonParameterUnsupported,
		},
		{
			name: "parameter the executor does not know", entry: narrowed, params: `{"region":"r","checksum":"crc32"}`, accepted: accepted,
			want: backupv1alpha1.CompatibilityIncompatible, wantReason: backupv1alpha1.ReasonParameterUnsupported,
		},
		{name: "custom CA supported", entry: caOnly, params: `{}`, tls: ca, accepted: accepted, want: backupv1alpha1.CompatibilityCompatible},
		{
			name: "insecure not supported", entry: caOnly, params: `{}`, tls: &common.TLSConfig{InsecureSkipVerify: true}, accepted: accepted,
			want: backupv1alpha1.CompatibilityIncompatible, wantReason: backupv1alpha1.ReasonTLSOptionUnsupported,
		},
		{
			name: "no tls block supports no setting", entry: s3Entry, params: `{}`, tls: ca, accepted: accepted,
			want: backupv1alpha1.CompatibilityIncompatible, wantReason: backupv1alpha1.ReasonTLSOptionUnsupported,
		},
		{name: "empty tls needs no support", entry: s3Entry, params: `{}`, tls: &common.TLSConfig{}, accepted: accepted, want: backupv1alpha1.CompatibilityCompatible},
	}
	storageClass := &backupv1alpha1.BackupStorageClass{ObjectMeta: metav1.ObjectMeta{Name: "s3", UID: "uid-s3", Generation: 2}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			class := verdictClass("pbm", 3, tt.entry)
			got := storageVerdict(verdictStorage(tt.params, tt.tls), tt.accepted, class, storageClass)
			assert.Equal(t, tt.want, got.Compatibility, got.Message)
			assert.Equal(t, tt.wantReason, got.Reason)
			want := backupv1alpha1.BackupClassVerdictInputs{
				StorageGeneration: 4, ClassUID: class.UID, ClassGeneration: 3, StorageClassUID: "uid-s3", StorageClassGeneration: 2,
			}
			assert.Equal(t, want, got.Observed)
		})
	}
}

func TestStorageVerdicts(t *testing.T) {
	t.Parallel()

	entry := backupv1alpha1.BackupStorageClassSupport{Name: "s3", AuthModes: []string{"AccessKey"}}
	accepted := metav1.Condition{Type: backupv1alpha1.BackupStorageConditionAccepted, Status: metav1.ConditionTrue}
	classes := []backupv1alpha1.BackupClass{
		*verdictClass("zeta", 1, entry, acceptedAt(1, metav1.ConditionTrue, "Accepted")),
		*verdictClass("alpha", 2, entry, acceptedAt(2, metav1.ConditionTrue, "Accepted")),
		*verdictClass("stale", 2, entry, acceptedAt(1, metav1.ConditionTrue, "Accepted")),
		*verdictClass("pending", 1, entry, acceptedAt(1, metav1.ConditionUnknown, "Pending")),
		*verdictClass("unclaimed", 1, entry),
	}
	got := storageVerdicts(verdictStorage(`{}`, nil), accepted, classes, nil)
	require.Len(t, got, 2)
	assert.Equal(t, "alpha", got[0].Name)
	assert.Equal(t, "zeta", got[1].Name)

	many := make([]backupv1alpha1.BackupClass, 0, maxVerdicts+5)
	for i := range maxVerdicts + 5 {
		many = append(many, *verdictClass(fmt.Sprintf("class-%03d", i), 1, entry, acceptedAt(1, metav1.ConditionTrue, "Accepted")))
	}
	assert.Len(t, storageVerdicts(verdictStorage(`{}`, nil), accepted, many, nil), maxVerdicts)
}

func TestProviderCompatibilities(t *testing.T) {
	t.Parallel()

	entry := backupv1alpha1.BackupStorageClassSupport{Name: "s3", AuthModes: []string{"AccessKey"}}
	tests := []struct {
		name         string
		class        *backupv1alpha1.BackupClass
		found        bool
		want         backupv1alpha1.Compatibility
		wantReason   string
		wantInDetail string
	}{
		{name: "accepted", class: verdictClass("pbm", 2, entry, acceptedAt(2, metav1.ConditionTrue, "Accepted")), found: true, want: backupv1alpha1.CompatibilityCompatible},
		{
			name: "provider missing", class: verdictClass("pbm", 2, entry, acceptedAt(2, metav1.ConditionTrue, "Accepted")),
			want: backupv1alpha1.CompatibilityPending, wantReason: backupv1alpha1.ReasonProviderNotFound,
		},
		{name: "no claim yet", class: verdictClass("pbm", 1, entry), found: true, want: backupv1alpha1.CompatibilityPending, wantReason: backupv1alpha1.ReasonClassNotAccepted},
		{
			name: "claim of an older generation", class: verdictClass("pbm", 3, entry, acceptedAt(2, metav1.ConditionTrue, "Accepted")), found: true,
			want: backupv1alpha1.CompatibilityPending, wantReason: backupv1alpha1.ReasonClassNotAccepted,
		},
		{
			name: "drift of an older generation", class: verdictClass("pbm", 3, entry, acceptedAt(2, metav1.ConditionFalse, backupv1alpha1.ReasonSpecDrift)), found: true,
			want: backupv1alpha1.CompatibilityPending, wantReason: backupv1alpha1.ReasonClassNotAccepted,
		},
		{
			name: "newer definition", class: verdictClass("pbm", 1, entry, acceptedAt(1, metav1.ConditionUnknown, "NewerDefinition")), found: true,
			want: backupv1alpha1.CompatibilityPending, wantReason: backupv1alpha1.ReasonClassNotAccepted, wantInDetail: "NewerDefinition",
		},
		{
			name: "spec drift", class: verdictClass("pbm", 1, entry, acceptedAt(1, metav1.ConditionFalse, backupv1alpha1.ReasonSpecDrift)), found: true,
			want: backupv1alpha1.CompatibilityIncompatible, wantReason: backupv1alpha1.ReasonSpecDrift,
		},
		{
			name: "unknown class", class: verdictClass("pbm", 1, entry, acceptedAt(1, metav1.ConditionFalse, backupv1alpha1.ReasonUnknownClass)), found: true,
			want: backupv1alpha1.CompatibilityIncompatible, wantReason: backupv1alpha1.ReasonUnknownClass,
		},
		{
			name: "rejected for another reason", class: verdictClass("pbm", 1, entry, acceptedAt(1, metav1.ConditionFalse, "Starting")), found: true,
			want: backupv1alpha1.CompatibilityPending, wantReason: backupv1alpha1.ReasonClassNotAccepted, wantInDetail: "Starting",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := providerCompatibilities(tt.class, tt.found)
			require.Len(t, got, 1)
			assert.Equal(t, "mongo", got[0].Name)
			assert.Equal(t, tt.want, got[0].Compatibility, got[0].Message)
			assert.Equal(t, tt.wantReason, got[0].Reason)
			assert.Equal(t, tt.class.Generation, got[0].ClassGeneration)
			assert.Contains(t, got[0].Message, tt.wantInDetail)
		})
	}

	other := verdictClass("job", 1, entry)
	other.Spec.ExecutionMode = "Job"
	assert.Empty(t, providerCompatibilities(other, true), "a mode without an executor Provider has no entry")
}
