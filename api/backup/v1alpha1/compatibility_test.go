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

package v1alpha1_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/backupstorageclasses/s3"
)

func TestCanWriteTo(t *testing.T) {
	t.Parallel()

	storage := func(class, mode string) *backupv1alpha1.BackupStorage {
		auth := common.Authentication{Mode: mode}
		if !common.IsReservedAuthMode(mode) {
			auth.CredentialsSecretRef = &common.SecretRef{Name: "creds"}
		}
		return &backupv1alpha1.BackupStorage{
			ObjectMeta: metav1.ObjectMeta{Name: "store", Namespace: "ns"},
			Spec: backupv1alpha1.BackupStorageSpec{
				BackupStorageClassRef: common.ObjectRef{Name: class},
				Location:              backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "backups"}},
				Authentication:        auth,
			},
		}
	}
	backupClass := func(supports ...backupv1alpha1.BackupStorageClassSupport) *backupv1alpha1.BackupClass {
		return &backupv1alpha1.BackupClass{
			ObjectMeta: metav1.ObjectMeta{Name: "engine-native"},
			Spec: backupv1alpha1.BackupClassSpec{
				ExecutionMode:        backupv1alpha1.BackupExecutionModeProviderManaged,
				BackupStorageClasses: supports,
			},
		}
	}
	s3AccessKey := backupv1alpha1.BackupStorageClassSupport{
		Name:      s3.ClassName,
		AuthModes: []string{s3.AuthModeAccessKey},
	}
	s3Both := backupv1alpha1.BackupStorageClassSupport{
		Name:      s3.ClassName,
		AuthModes: []string{s3.AuthModeAccessKey, common.AuthModeWorkloadIdentity},
	}
	thirdParty := backupv1alpha1.BackupStorageClassSupport{
		Name:      "example.test.openeverest.io",
		AuthModes: []string{"Token"},
	}

	tests := []struct {
		name        string
		class       *backupv1alpha1.BackupClass
		storage     *backupv1alpha1.BackupStorage
		want        backupv1alpha1.StorageCompatibility
		wantMessage string
	}{
		{
			name:    "class and auth mode listed",
			class:   backupClass(s3AccessKey),
			storage: storage(s3.ClassName, s3.AuthModeAccessKey),
			want:    backupv1alpha1.StorageCompatibility{Compatible: true},
		},
		{
			name:    "auth mode among several listed",
			class:   backupClass(thirdParty, s3Both),
			storage: storage(s3.ClassName, common.AuthModeWorkloadIdentity),
			want:    backupv1alpha1.StorageCompatibility{Compatible: true},
		},
		{
			name:        "storage class not listed",
			class:       backupClass(s3AccessKey),
			storage:     storage("example.test.openeverest.io", "Token"),
			want:        backupv1alpha1.StorageCompatibility{Reason: backupv1alpha1.ReasonBackupStorageClassUnsupported},
			wantMessage: `BackupClass "engine-native" cannot write to BackupStorageClass "example.test.openeverest.io"`,
		},
		{
			name:        "auth mode not listed for the storage class",
			class:       backupClass(s3AccessKey),
			storage:     storage(s3.ClassName, common.AuthModeWorkloadIdentity),
			want:        backupv1alpha1.StorageCompatibility{Reason: backupv1alpha1.ReasonAuthModeUnsupported},
			wantMessage: `BackupClass "engine-native" supports BackupStorageClass "s3" only with authentication modes [AccessKey], not "WorkloadIdentity"`,
		},
		{
			name:        "auth mode listed only for another storage class",
			class:       backupClass(s3AccessKey, thirdParty),
			storage:     storage(s3.ClassName, "Token"),
			want:        backupv1alpha1.StorageCompatibility{Reason: backupv1alpha1.ReasonAuthModeUnsupported},
			wantMessage: `BackupClass "engine-native" supports BackupStorageClass "s3" only with authentication modes [AccessKey], not "Token"`,
		},
		{
			name:        "class lists no storage class",
			class:       backupClass(),
			storage:     storage(s3.ClassName, s3.AuthModeAccessKey),
			want:        backupv1alpha1.StorageCompatibility{Reason: backupv1alpha1.ReasonBackupStorageClassUnsupported},
			wantMessage: `BackupClass "engine-native" cannot write to BackupStorageClass "s3"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.class.CanWriteTo(tt.storage)
			assert.Equal(t, tt.want.Compatible, got.Compatible)
			assert.Equal(t, tt.want.Reason, got.Reason)
			assert.Equal(t, tt.wantMessage, got.Message)
		})
	}
}

func TestBackupPath(t *testing.T) {
	t.Parallel()

	const uid = types.UID("0f8fad5b-d9cb-469f-a165-70867728950e")
	tests := []struct {
		name     string
		location backupv1alpha1.BackupStorageLocation
		want     string
	}{
		{
			name:     "without prefix",
			location: backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "team-a-backups"}},
			want:     "backups/team-a/orders/0f8fad5b-d9cb-469f-a165-70867728950e/s3-primary",
		},
		{
			name:     "with single-segment prefix",
			location: backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "team-a-backups", Prefix: "everest"}},
			want:     "everest/backups/team-a/orders/0f8fad5b-d9cb-469f-a165-70867728950e/s3-primary",
		},
		{
			name:     "with nested prefix",
			location: backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "team-a-backups", Prefix: "prod/eu-west"}},
			want:     "prod/eu-west/backups/team-a/orders/0f8fad5b-d9cb-469f-a165-70867728950e/s3-primary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, backupv1alpha1.BackupPath(tt.location, "team-a", "orders", uid, "s3-primary"))
		})
	}
}

func TestBackupPathFor(t *testing.T) {
	t.Parallel()

	const uid = types.UID("0f8fad5b-d9cb-469f-a165-70867728950e")
	location := func(prefix string) backupv1alpha1.BackupStorageLocation {
		return backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "team-a-backups", Prefix: prefix}}
	}
	tests := []struct {
		name          string
		style         backupv1alpha1.BackupPathStyle
		prefix        string
		wantPath      string
		wantKeyPrefix string
	}{
		{
			name:          "nested without prefix",
			style:         backupv1alpha1.BackupPathNested,
			wantPath:      "backups/team-a/orders/0f8fad5b-d9cb-469f-a165-70867728950e/s3-primary",
			wantKeyPrefix: "",
		},
		{
			name:          "nested with nested prefix",
			style:         backupv1alpha1.BackupPathNested,
			prefix:        "prod/eu-west/",
			wantPath:      "prod/eu-west/backups/team-a/orders/0f8fad5b-d9cb-469f-a165-70867728950e/s3-primary",
			wantKeyPrefix: "prod/eu-west/",
		},
		{
			name:          "the empty style is nested",
			prefix:        "prod",
			wantPath:      "prod/backups/team-a/orders/0f8fad5b-d9cb-469f-a165-70867728950e/s3-primary",
			wantKeyPrefix: "prod/",
		},
		{
			name:          "flat without prefix",
			style:         backupv1alpha1.BackupPathFlat,
			wantPath:      "backups_team-a_orders_0f8fad5b-d9cb-469f-a165-70867728950e_s3-primary",
			wantKeyPrefix: "",
		},
		{
			name:          "flat with nested prefix",
			style:         backupv1alpha1.BackupPathFlat,
			prefix:        "prod/eu-west/",
			wantPath:      "prod_eu-west_backups_team-a_orders_0f8fad5b-d9cb-469f-a165-70867728950e_s3-primary",
			wantKeyPrefix: "prod_eu-west_",
		},
		{
			name:          "flat with a prefix holding underscores",
			style:         backupv1alpha1.BackupPathFlat,
			prefix:        "team_a",
			wantPath:      "team_a_backups_team-a_orders_0f8fad5b-d9cb-469f-a165-70867728950e_s3-primary",
			wantKeyPrefix: "team_a_",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			loc := location(tt.prefix)
			got := backupv1alpha1.BackupPathFor(tt.style, loc, "team-a", "orders", uid, "s3-primary")
			assert.Equal(t, tt.wantPath, got)
			assert.Equal(t, got, tt.style.Apply(got), "Apply is idempotent")
			assert.Equal(t, tt.wantKeyPrefix, backupv1alpha1.BackupKeyPrefix(tt.style, loc))
			assert.True(t, strings.HasPrefix(got, backupv1alpha1.BackupKeyPrefix(tt.style, loc)))
			if tt.style == backupv1alpha1.BackupPathFlat {
				// The folder provider-cassandra renders as the Medusa prefix
				// since provider-cassandra#40.
				assert.Equal(t, strings.ReplaceAll(backupv1alpha1.BackupPath(loc, "team-a", "orders", uid, "s3-primary"), "/", "_"), got)
				assert.NotContains(t, got, "/")
			}
		})
	}
}

func TestCurrentVerdict(t *testing.T) {
	t.Parallel()

	class := &backupv1alpha1.BackupClass{ObjectMeta: metav1.ObjectMeta{Name: "pbm", UID: "class-1", Generation: 3}}
	storageClass := &backupv1alpha1.BackupStorageClass{ObjectMeta: metav1.ObjectMeta{Name: s3.ClassName, UID: "s3-1", Generation: 2}}
	current := backupv1alpha1.BackupClassVerdictInputs{
		StorageGeneration: 5, ClassUID: "class-1", ClassGeneration: 3, StorageClassUID: "s3-1", StorageClassGeneration: 2,
	}
	verdict := backupv1alpha1.BackupClassVerdict{
		Name: "pbm", Compatibility: backupv1alpha1.CompatibilityIncompatible,
		Reason: backupv1alpha1.ReasonTLSOptionUnsupported, Message: "no custom CA", Observed: current,
	}
	storage := func(verdicts ...backupv1alpha1.BackupClassVerdict) *backupv1alpha1.BackupStorage {
		return &backupv1alpha1.BackupStorage{
			ObjectMeta: metav1.ObjectMeta{Name: "store", Generation: 5},
			Spec:       backupv1alpha1.BackupStorageSpec{BackupStorageClassRef: common.ObjectRef{Name: s3.ClassName}},
			Status:     backupv1alpha1.BackupStorageStatus{BackupClasses: verdicts},
		}
	}
	stale := func(mutate func(*backupv1alpha1.BackupClassVerdictInputs)) backupv1alpha1.BackupClassVerdict {
		v := verdict
		mutate(&v.Observed)
		return v
	}

	assert.Equal(t, verdict, backupv1alpha1.CurrentVerdict(storage(verdict), class, storageClass), "current")

	tests := []struct {
		name         string
		storage      *backupv1alpha1.BackupStorage
		storageClass *backupv1alpha1.BackupStorageClass
	}{
		{name: "no verdict", storage: storage(), storageClass: storageClass},
		{name: "verdict for another class", storage: storage(backupv1alpha1.BackupClassVerdict{Name: "other", Observed: current}), storageClass: storageClass},
		{name: "storage changed", storage: storage(stale(func(o *backupv1alpha1.BackupClassVerdictInputs) { o.StorageGeneration = 4 })), storageClass: storageClass},
		{name: "class changed", storage: storage(stale(func(o *backupv1alpha1.BackupClassVerdictInputs) { o.ClassGeneration = 2 })), storageClass: storageClass},
		{name: "class recreated", storage: storage(stale(func(o *backupv1alpha1.BackupClassVerdictInputs) { o.ClassUID = "class-0" })), storageClass: storageClass},
		{name: "storage class changed", storage: storage(stale(func(o *backupv1alpha1.BackupClassVerdictInputs) { o.StorageClassGeneration = 1 })), storageClass: storageClass},
		{name: "storage class recreated", storage: storage(stale(func(o *backupv1alpha1.BackupClassVerdictInputs) { o.StorageClassUID = "s3-0" })), storageClass: storageClass},
		{name: "storage class missing", storage: storage(verdict)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := backupv1alpha1.CurrentVerdict(tt.storage, class, tt.storageClass)
			assert.Equal(t, "pbm", got.Name)
			assert.Equal(t, backupv1alpha1.CompatibilityPending, got.Compatibility)
			assert.Equal(t, backupv1alpha1.ReasonVerdictPending, got.Reason)
			assert.NotEmpty(t, got.Message)
		})
	}
}

func TestExecutorProvider(t *testing.T) {
	t.Parallel()

	pm := &backupv1alpha1.ProviderManagedSpec{ProviderRef: common.ObjectRef{Name: "percona-server-mongodb"}}
	for name, tt := range map[string]struct {
		spec backupv1alpha1.BackupClassSpec
		want string
	}{
		"provider managed":        {spec: backupv1alpha1.BackupClassSpec{ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged, ProviderManaged: pm}, want: "percona-server-mongodb"},
		"provider managed, empty": {spec: backupv1alpha1.BackupClassSpec{ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged}},
		"another mode":            {spec: backupv1alpha1.BackupClassSpec{ExecutionMode: "Other", ProviderManaged: pm}},
	} {
		assert.Equal(t, tt.want, tt.spec.ExecutorProvider(), name)
	}
}

func TestCurrentProviderVerdict(t *testing.T) {
	t.Parallel()

	class := func(generation int64, entries ...backupv1alpha1.BackupClassProviderCompatibility) *backupv1alpha1.BackupClass {
		return &backupv1alpha1.BackupClass{
			ObjectMeta: metav1.ObjectMeta{Name: "pbm", Generation: generation},
			Status:     backupv1alpha1.BackupClassStatus{Providers: entries},
		}
	}
	drift := backupv1alpha1.BackupClassProviderCompatibility{
		Name: "psmdb", Compatibility: backupv1alpha1.CompatibilityIncompatible,
		Reason: backupv1alpha1.ReasonSpecDrift, Message: "edited", ClassGeneration: 2,
	}

	assert.Equal(t, drift, backupv1alpha1.CurrentProviderVerdict(class(2, drift), "psmdb"), "the current entry")

	tests := []struct {
		name       string
		class      *backupv1alpha1.BackupClass
		want       backupv1alpha1.Compatibility
		wantReason string
	}{
		{name: "entry of an older generation", class: class(3, drift), want: backupv1alpha1.CompatibilityPending, wantReason: backupv1alpha1.ReasonVerdictPending},
		{name: "no entries yet", class: class(2), want: backupv1alpha1.CompatibilityPending, wantReason: backupv1alpha1.ReasonVerdictPending},
		{
			name: "only stale entries for others", class: class(3, backupv1alpha1.BackupClassProviderCompatibility{Name: "pxc", ClassGeneration: 2}),
			want: backupv1alpha1.CompatibilityPending, wantReason: backupv1alpha1.ReasonVerdictPending,
		},
		{
			name: "current entries for others only", class: class(2, backupv1alpha1.BackupClassProviderCompatibility{Name: "pxc", ClassGeneration: 2}),
			want: backupv1alpha1.CompatibilityIncompatible, wantReason: backupv1alpha1.ReasonProviderUnsupported,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := backupv1alpha1.CurrentProviderVerdict(tt.class, "psmdb")
			assert.Equal(t, "psmdb", got.Name)
			assert.Equal(t, tt.want, got.Compatibility)
			assert.Equal(t, tt.wantReason, got.Reason)
			assert.Equal(t, tt.class.Generation, got.ClassGeneration)
			assert.NotEmpty(t, got.Message)
		})
	}
}

func TestCanWriteToNamesTheUnsupportedParameter(t *testing.T) {
	t.Parallel()

	class := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pbm"},
		Spec: backupv1alpha1.BackupClassSpec{BackupStorageClasses: []backupv1alpha1.BackupStorageClassSupport{{
			Name: s3.ClassName, AuthModes: []string{s3.AuthModeAccessKey},
			ParametersSchema: &common.ParametersSchema{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
				Type: "object",
				Properties: map[string]apiextensionsv1.JSONSchemaProps{
					"implementation": {Type: "string", Enum: []apiextensionsv1.JSON{{Raw: []byte(`"aws"`)}}},
				},
			}},
		}}},
	}
	storage := &backupv1alpha1.BackupStorage{Spec: backupv1alpha1.BackupStorageSpec{
		BackupStorageClassRef: common.ObjectRef{Name: s3.ClassName},
		Parameters:            &runtime.RawExtension{Raw: []byte(`{"implementation":"gcs"}`)},
		Authentication:        common.Authentication{Mode: s3.AuthModeAccessKey},
	}}
	got := class.CanWriteTo(storage)
	assert.Equal(t, backupv1alpha1.ReasonParameterUnsupported, got.Reason)
	assert.Contains(t, got.Message, "implementation")
	assert.NotContains(t, got.Message, "\n")
}
