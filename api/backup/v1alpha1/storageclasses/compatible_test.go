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
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

func TestCompatible(t *testing.T) {
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
				Location:              backupv1alpha1.BackupStorageLocation{Bucket: "backups"},
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
		Name:      storageclasses.ClassS3,
		AuthModes: []string{storageclasses.AuthModeAccessKey},
	}
	s3Both := backupv1alpha1.BackupStorageClassSupport{
		Name:      storageclasses.ClassS3,
		AuthModes: []string{storageclasses.AuthModeAccessKey, common.AuthModeWorkloadIdentity},
	}
	thirdParty := backupv1alpha1.BackupStorageClassSupport{
		Name:      "example.test.openeverest.io",
		AuthModes: []string{"Token"},
	}

	tests := []struct {
		name        string
		class       *backupv1alpha1.BackupClass
		storage     *backupv1alpha1.BackupStorage
		want        storageclasses.Compatibility
		wantMessage string
	}{
		{
			name:    "class and auth mode listed",
			class:   backupClass(s3AccessKey),
			storage: storage(storageclasses.ClassS3, storageclasses.AuthModeAccessKey),
			want:    storageclasses.Compatibility{Compatible: true},
		},
		{
			name:    "auth mode among several listed",
			class:   backupClass(thirdParty, s3Both),
			storage: storage(storageclasses.ClassS3, common.AuthModeWorkloadIdentity),
			want:    storageclasses.Compatibility{Compatible: true},
		},
		{
			name:        "storage class not listed",
			class:       backupClass(s3AccessKey),
			storage:     storage("example.test.openeverest.io", "Token"),
			want:        storageclasses.Compatibility{Reason: backupv1alpha1.ReasonBackupStorageClassUnsupported},
			wantMessage: `BackupClass "engine-native" cannot write to BackupStorageClass "example.test.openeverest.io"`,
		},
		{
			name:        "auth mode not listed for the storage class",
			class:       backupClass(s3AccessKey),
			storage:     storage(storageclasses.ClassS3, common.AuthModeWorkloadIdentity),
			want:        storageclasses.Compatibility{Reason: backupv1alpha1.ReasonAuthModeUnsupported},
			wantMessage: `BackupClass "engine-native" supports BackupStorageClass "s3" only with authentication modes [AccessKey], not "WorkloadIdentity"`,
		},
		{
			name:        "auth mode listed only for another storage class",
			class:       backupClass(s3AccessKey, thirdParty),
			storage:     storage(storageclasses.ClassS3, "Token"),
			want:        storageclasses.Compatibility{Reason: backupv1alpha1.ReasonAuthModeUnsupported},
			wantMessage: `BackupClass "engine-native" supports BackupStorageClass "s3" only with authentication modes [AccessKey], not "Token"`,
		},
		{
			name:        "class lists no storage class",
			class:       backupClass(),
			storage:     storage(storageclasses.ClassS3, storageclasses.AuthModeAccessKey),
			want:        storageclasses.Compatibility{Reason: backupv1alpha1.ReasonBackupStorageClassUnsupported},
			wantMessage: `BackupClass "engine-native" cannot write to BackupStorageClass "s3"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := storageclasses.Compatible(tt.class, tt.storage)
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
			location: backupv1alpha1.BackupStorageLocation{Bucket: "team-a-backups"},
			want:     "backups/team-a/orders/0f8fad5b-d9cb-469f-a165-70867728950e/s3-primary",
		},
		{
			name:     "with single-segment prefix",
			location: backupv1alpha1.BackupStorageLocation{Bucket: "team-a-backups", Prefix: "everest"},
			want:     "everest/backups/team-a/orders/0f8fad5b-d9cb-469f-a165-70867728950e/s3-primary",
		},
		{
			name:     "with nested prefix",
			location: backupv1alpha1.BackupStorageLocation{Bucket: "team-a-backups", Prefix: "prod/eu-west"},
			want:     "prod/eu-west/backups/team-a/orders/0f8fad5b-d9cb-469f-a165-70867728950e/s3-primary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, storageclasses.BackupPath(tt.location, "team-a", "orders", uid, "s3-primary"))
		})
	}
}
