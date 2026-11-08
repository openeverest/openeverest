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

package v1alpha1

import (
	"fmt"
	"path"
	"slices"

	"k8s.io/apimachinery/pkg/types"
)

// StorageCompatibility is whether a BackupClass can write to a
// BackupStorage.
//
// +kubebuilder:object:generate=false
type StorageCompatibility struct {
	Compatible bool
	// Reason and Message say why the class cannot write to the storage.
	Reason  string
	Message string
}

// CanWriteTo reports whether the class can write to storage: its
// backupStorageClasses must list the storage's BackupStorageClass, and that
// entry the storage's authentication mode. It is the only place the rule
// lives.
func (c *BackupClass) CanWriteTo(storage *BackupStorage) StorageCompatibility {
	storageClass := storage.Spec.BackupStorageClassRef.Name
	entry := c.Spec.FindBackupStorageClass(storageClass)
	if entry == nil {
		return StorageCompatibility{
			Reason:  ReasonBackupStorageClassUnsupported,
			Message: fmt.Sprintf("BackupClass %q cannot write to BackupStorageClass %q", c.Name, storageClass),
		}
	}
	mode := storage.Spec.Authentication.Mode
	if !slices.Contains(entry.AuthModes, mode) {
		return StorageCompatibility{
			Reason: ReasonAuthModeUnsupported,
			Message: fmt.Sprintf("BackupClass %q supports BackupStorageClass %q only with authentication modes %v, not %q",
				c.Name, storageClass, entry.AuthModes, mode,
			),
		}
	}
	return StorageCompatibility{Compatible: true}
}

// backupsSegment separates backups from anything else kept in the bucket, so
// one lifecycle or replication rule on <prefix>/backups/ covers every backup.
const backupsSegment = "backups"

// BackupPath is the path in the storage's location where an Instance writes
// the backups of one storage entry:
// <prefix>/backups/<namespace>/<instance>/<instance-uid>/<storage>, without
// the prefix when it is empty. Every executor writes there, so Instances, and
// storages of one Instance, that share a bucket never share a path.
func BackupPath(location BackupStorageLocation, namespace, instance string, uid types.UID, storage string) string {
	var root string
	if location.ObjectStore != nil {
		root = location.ObjectStore.Prefix
	}
	return path.Join(root, backupsSegment, namespace, instance, string(uid), storage)
}
