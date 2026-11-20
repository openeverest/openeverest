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
	"strings"

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

// CurrentProviderVerdict returns the class's verdict for Instances of
// provider from status.providers. An entry computed for another generation of
// the class counts as Pending with ReasonVerdictPending, and so does a
// provider absent from a list core has not computed for the current
// generation yet. A provider absent from a current list is Incompatible with
// ReasonProviderUnsupported: the class does not run on it. Read
// status.providers only through it.
func CurrentProviderVerdict(class *BackupClass, provider string) BackupClassProviderCompatibility {
	current := func(e BackupClassProviderCompatibility) bool { return e.ClassGeneration == class.Generation }
	i := slices.IndexFunc(class.Status.Providers, func(e BackupClassProviderCompatibility) bool { return e.Name == provider })
	switch {
	case i >= 0 && current(class.Status.Providers[i]):
		return class.Status.Providers[i]
	case i >= 0:
		return pendingProviderVerdict(class, provider, fmt.Sprintf(
			"the verdict on BackupClass %q for Provider %q predates generation %d of the class",
			class.Name, provider, class.Generation,
		))
	case slices.ContainsFunc(class.Status.Providers, current):
		return BackupClassProviderCompatibility{
			Name:            provider,
			Compatibility:   CompatibilityIncompatible,
			Reason:          ReasonProviderUnsupported,
			Message:         fmt.Sprintf("BackupClass %q does not run on Provider %q", class.Name, provider),
			ClassGeneration: class.Generation,
		}
	}
	return pendingProviderVerdict(class, provider, fmt.Sprintf(
		"BackupClass %q has not been judged for Provider %q at generation %d yet", class.Name, provider, class.Generation,
	))
}

func pendingProviderVerdict(class *BackupClass, provider, message string) BackupClassProviderCompatibility {
	return BackupClassProviderCompatibility{
		Name:            provider,
		Compatibility:   CompatibilityPending,
		Reason:          ReasonVerdictPending,
		Message:         message,
		ClassGeneration: class.Generation,
	}
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
				Reason: ReasonParameterUnsupported,
				Message: fmt.Sprintf("BackupClass %q does not support the parameters of the storage: %s",
					c.Name, strings.ReplaceAll(err.Error(), "\n", ": "),
				),
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
// the prefix when it is empty. Every executor writes there, laid out in its
// BackupPathStyle (see BackupPathFor), so Instances, and storages of one
// Instance, that share a bucket never share a path.
func BackupPath(location BackupStorageLocation, namespace, instance string, uid types.UID, storage string) string {
	return path.Join(locationPrefix(location), backupsSegment, namespace, instance, string(uid), storage)
}

func locationPrefix(location BackupStorageLocation) string {
	if location.ObjectStore == nil {
		return ""
	}
	return location.ObjectStore.Prefix
}

// BackupPathStyle is how an executor lays out the path BackupPath returns in
// the bucket.
type BackupPathStyle string

const (
	// BackupPathNested lays the path out as nested directories, as
	// BackupPath returns it. It is the default, and the empty style means
	// it.
	BackupPathNested = BackupPathStyle("Nested")
	// BackupPathFlat lays the path out as one folder at the root of the
	// bucket: the nested path with every "/" replaced by "_", e.g.
	// prod_backups_team-a_orders_<uid>_s3-primary for the prefix prod. It
	// is for engines that cannot write under a nested path, such as Medusa,
	// which refuses a storage prefix containing "/". "_" cannot occur in a
	// namespace, instance, UID or BackupStorage name, so distinct entries
	// still get distinct folders. The folder is not under the storage's
	// prefix: a lifecycle or replication rule on <prefix>/backups/ does not
	// cover it, one on <prefix>_backups_ does.
	BackupPathFlat = BackupPathStyle("Flat")
)

// Apply returns p, a path in the nested layout, laid out in the style. It is
// idempotent: a path already in the style is returned as is.
func (s BackupPathStyle) Apply(p string) string {
	if s == BackupPathFlat {
		return strings.ReplaceAll(p, "/", "_")
	}
	return p
}

// BackupPathFor is BackupPath laid out in style: the path an executor of
// that style writes the backups of one storage entry to, and the one the
// runtime records for it.
func BackupPathFor(style BackupPathStyle, location BackupStorageLocation, namespace, instance string, uid types.UID, storage string) string {
	return style.Apply(BackupPath(location, namespace, instance, uid, storage))
}

// BackupKeyPrefix returns the prefix that the object keys of the paths an
// executor of style writes in location start with: <prefix>/ for the nested
// style, <prefix>_ with every "/" of the prefix replaced by "_" for the flat
// one, and nothing when the location has no prefix. An import lists the
// keys under it, and records the path of each backup it finds relative to
// it, as BackupOriginExternal.path.
func BackupKeyPrefix(style BackupPathStyle, location BackupStorageLocation) string {
	return strings.TrimSuffix(style.Apply(path.Join(locationPrefix(location), backupsSegment)), backupsSegment)
}
