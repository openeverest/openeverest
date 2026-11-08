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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// Compatibility is whether a class can be used with what it is judged
// against. Pending may still become Compatible without a change of either
// object, such as when a late object appears or a controller catches up;
// Incompatible needs one.
//
// +kubebuilder:validation:Enum=Compatible;Pending;Incompatible
type Compatibility string

// Compatibility levels.
const (
	CompatibilityCompatible   = Compatibility("Compatible")
	CompatibilityPending      = Compatibility("Pending")
	CompatibilityIncompatible = Compatibility("Incompatible")
)

// CurrentVerdict returns the storage's verdict for class, or a Pending one
// with ReasonVerdictPending when there is none or it was computed from other
// objects than storage, class and storageClass, the storage's
// BackupStorageClass, nil when it does not exist. Read verdicts only through
// it: a verdict is only as current as its inputs.
func CurrentVerdict(storage *BackupStorage, class *BackupClass, storageClass *BackupStorageClass) BackupClassVerdict {
	i := slices.IndexFunc(storage.Status.BackupClasses, func(v BackupClassVerdict) bool { return v.Name == class.Name })
	if i < 0 {
		return pendingVerdict(class.Name, fmt.Sprintf("BackupStorage %q has not been judged against BackupClass %q yet", storage.Name, class.Name))
	}
	v := storage.Status.BackupClasses[i]
	if storageClass == nil {
		return pendingVerdict(class.Name, fmt.Sprintf("BackupStorageClass %q of BackupStorage %q does not exist",
			storage.Spec.BackupStorageClassRef.Name, storage.Name,
		))
	}
	current := BackupClassVerdictInputs{
		StorageGeneration:      storage.Generation,
		ClassUID:               class.UID,
		ClassGeneration:        class.Generation,
		StorageClassUID:        storageClass.UID,
		StorageClassGeneration: storageClass.Generation,
	}
	if v.Observed != current {
		return pendingVerdict(class.Name, fmt.Sprintf(
			"the verdict on BackupStorage %q for BackupClass %q predates a change of the storage, the class or its BackupStorageClass",
			storage.Name, class.Name,
		))
	}
	return v
}

func pendingVerdict(class, message string) BackupClassVerdict {
	return BackupClassVerdict{Name: class, Compatibility: CompatibilityPending, Reason: ReasonVerdictPending, Message: message}
}

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
// backupStorageClasses must list the storage's BackupStorageClass, that entry
// the storage's authentication mode, the entry's parametersSchema must accept
// the storage's parameters, and the entry's tls support every setting of the
// storage's spec.tls. It is the only place the rule lives; core publishes its
// result in BackupStorage.status.backupClasses.
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
	if entry.ParametersSchema != nil && entry.ParametersSchema.OpenAPIV3Schema != nil {
		params := storage.Spec.Parameters
		if params == nil {
			params = &runtime.RawExtension{Raw: []byte("{}")}
		}
		if err := entry.ParametersSchema.Validate(params); err != nil {
			return StorageCompatibility{
				Reason:  ReasonParameterUnsupported,
				Message: fmt.Sprintf("BackupClass %q does not support the parameters of the storage: %v", c.Name, err),
			}
		}
	}
	if setting := unsupportedTLSSetting(entry.TLS, storage.Spec.TLS); setting != "" {
		return StorageCompatibility{
			Reason:  ReasonTLSOptionUnsupported,
			Message: fmt.Sprintf("BackupClass %q does not support tls.%s on BackupStorageClass %q", c.Name, setting, storageClass),
		}
	}
	return StorageCompatibility{Compatible: true}
}

// unsupportedTLSSetting returns the setting of tls that support does not
// honour, or "". An absent support block supports none.
func unsupportedTLSSetting(support *BackupStorageTLSSupport, tls *common.TLSConfig) string {
	if tls == nil {
		return ""
	}
	var s BackupStorageTLSSupport
	if support != nil {
		s = *support
	}
	switch {
	case tls.CA != nil && s.CustomCA != TLSOptionSupported:
		return "ca"
	case tls.InsecureSkipVerify && s.InsecureSkipVerify != TLSOptionSupported:
		return "insecureSkipVerify"
	}
	return ""
}

// IsAccepted reports whether the class's executor claims it at its current
// generation: an Accepted condition judged on an older generation does not
// count.
func (c *BackupClass) IsAccepted() bool {
	cond := meta.FindStatusCondition(c.Status.Conditions, BackupClassConditionAccepted)
	return cond != nil && cond.Status == metav1.ConditionTrue && cond.ObservedGeneration == c.Generation
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
