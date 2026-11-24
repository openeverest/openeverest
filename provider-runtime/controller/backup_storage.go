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

package controller

import (
	"errors"
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/backupstorageclasses"
)

var (
	// ErrImplementationUnsupported means the engine does not support the
	// implementation a BackupStorage names.
	ErrImplementationUnsupported = errors.New("implementation not supported")
	// ErrTLSOptionUnsupported means the engine cannot honour the TLS settings
	// of a BackupStorage.
	ErrTLSOptionUnsupported = errors.New("tls option not supported")
)

// ResolvedBackupStorage is one entry of the Instance's spec.backup.storages
// with the BackupStorage it names resolved and judged.
type ResolvedBackupStorage struct {
	// Entry is the spec.backup.storages entry.
	Entry *v1alpha1.InstanceBackupStorage
	// Storage is nil when the BackupStorage does not exist.
	Storage *backupv1alpha1.BackupStorage
	// Path is where the engine writes this Instance's backups, within
	// Storage.Spec.Location.ObjectStore.Bucket, laid out in the provider's
	// BackupPathStyle. Every engine writes under it.
	Path string
	// Reason and Message say why the entry must not be rendered as
	// specified; both are empty when it must.
	Reason  string
	Message string
}

// Apply reports whether the provider MUST render the entry. When false, the
// provider MUST keep whatever the live engine CR carries for the entry (hold
// rule), so schedules and PITR continue against the last configuration that
// worked, and MUST NOT render an entry the engine does not have yet.
func (r ResolvedBackupStorage) Apply() bool {
	return r.Reason == ""
}

// Name returns the BackupStorage name the entry references.
func (r ResolvedBackupStorage) Name() string {
	return r.Entry.StorageRef.Name
}

// BackupStorages resolves and judges every entry of the Instance's
// spec.backup.storages, in order. An entry may be rendered only when its
// BackupStorage exists, is Accepted, its current verdict for the Instance's
// BackupClass in status.backupClasses is Compatible, and it renders to the
// path recorded in status.backup.storages, read in the provider's
// BackupPathStyle. Entries the provider rejected with RejectBackupStorage
// keep that verdict.
func (c *Context) BackupStorages() ([]ResolvedBackupStorage, error) {
	if c.backupStorages != nil {
		return c.backupStorages, nil
	}
	spec := c.in.Spec.Backup
	if spec == nil {
		return nil, nil
	}
	class, err := c.BackupClassForInstance()
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	recorded := make(map[string]string)
	if c.in.Status.Backup != nil {
		for _, s := range c.in.Status.Backup.Storages {
			recorded[s.Name] = s.Location
		}
	}

	storageClasses := make(map[string]*backupv1alpha1.BackupStorageClass)
	out := make([]ResolvedBackupStorage, 0, len(spec.Storages))
	for i := range spec.Storages {
		entry := &spec.Storages[i]
		rs := ResolvedBackupStorage{Entry: entry}
		name := entry.StorageRef.Name
		bs := &backupv1alpha1.BackupStorage{}
		if err := c.client.Get(c.ctx, client.ObjectKey{Namespace: c.in.Namespace, Name: name}, bs); err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("get BackupStorage %q: %w", name, err)
			}
			rs.Reason = v1alpha1.ReasonBackupStorageNotFound
			rs.Message = fmt.Sprintf("backup storage %s not found in %s", name, c.in.Namespace)
			out = append(out, rs)
			continue
		}
		rs.Storage = bs
		rs.Path = backupv1alpha1.BackupPathFor(c.BackupPathStyle(), bs.Spec.Location, c.in.Namespace, c.in.Name, c.in.UID, name)
		var storageClass *backupv1alpha1.BackupStorageClass
		if class != nil {
			if storageClass, err = c.backupStorageClass(storageClasses, bs.Spec.BackupStorageClassRef.Name); err != nil {
				return nil, err
			}
		}
		rs.Reason, rs.Message = judgeBackupStorage(class, bs, storageClass, rs.Path, c.BackupPathStyle().Apply(recorded[name]))
		out = append(out, rs)
	}
	for i := range out {
		if r, ok := c.backupStorageRejections[out[i].Name()]; ok && out[i].Apply() {
			out[i].Reason, out[i].Message = r.reason, r.message
		}
	}
	c.backupStorages = out
	return out, nil
}

// backupStorageClass returns the BackupStorageClass named name, nil when it
// does not exist, through cache.
func (c *Context) backupStorageClass(
	cache map[string]*backupv1alpha1.BackupStorageClass, name string,
) (*backupv1alpha1.BackupStorageClass, error) {
	if sc, ok := cache[name]; ok {
		return sc, nil
	}
	var found *backupv1alpha1.BackupStorageClass
	sc := &backupv1alpha1.BackupStorageClass{}
	switch err := c.client.Get(c.ctx, client.ObjectKey{Name: name}, sc); {
	case err == nil:
		found = sc
	case !apierrors.IsNotFound(err):
		return nil, fmt.Errorf("get BackupStorageClass %q: %w", name, err)
	}
	cache[name] = found
	return found, nil
}

// judgeBackupStorage returns why the entry naming bs must not be rendered,
// or nothing. Whether class can write to bs is the verdict core publishes in
// bs's status.backupClasses, read through CurrentVerdict with storageClass,
// the storage's BackupStorageClass: an Incompatible verdict is
// BackupStorageIncompatible, any other that is not Compatible holds the
// entry as BackupStorageVerdictPending until core publishes a current one.
func judgeBackupStorage(
	class *backupv1alpha1.BackupClass, bs *backupv1alpha1.BackupStorage, storageClass *backupv1alpha1.BackupStorageClass,
	path, recorded string,
) (string, string) {
	accepted := meta.FindStatusCondition(bs.Status.Conditions, backupv1alpha1.BackupStorageConditionAccepted)
	switch {
	case accepted == nil || accepted.ObservedGeneration != bs.Generation:
		return v1alpha1.ReasonBackupStorageNotAccepted,
			fmt.Sprintf("backup storage %s has not been validated yet", bs.Name)
	case accepted.Status != metav1.ConditionTrue:
		return v1alpha1.ReasonBackupStorageNotAccepted,
			fmt.Sprintf("backup storage %s is not accepted: %s: %s", bs.Name, accepted.Reason, accepted.Message)
	}
	if class != nil {
		v := backupv1alpha1.CurrentVerdict(bs, class, storageClass)
		if v.Compatibility == backupv1alpha1.CompatibilityIncompatible {
			return v1alpha1.ReasonBackupStorageIncompatible,
				fmt.Sprintf("backup class %s cannot write to backup storage %s: %s: %s", class.Name, bs.Name, v.Reason, v.Message)
		}
		if v.Compatibility != backupv1alpha1.CompatibilityCompatible {
			return v1alpha1.ReasonBackupStorageVerdictPending,
				fmt.Sprintf("backup storage %s is %s for backup class %s: %s: %s", bs.Name, v.Compatibility, class.Name, v.Reason, v.Message)
		}
	}
	if recorded != "" && recorded != path {
		return v1alpha1.ReasonBackupStorageLocationChanged, fmt.Sprintf(
			"backup storage %s now renders to %q, not %q where this instance's backups are: "+
				"remove the entry, or point it at a storage with the original location",
			bs.Name, path, recorded,
		)
	}
	return "", ""
}

// SetBackupPathStyle sets how the provider lays out backup paths, from
// BackupPathStyleProvider. It is set by the runtime; providers do not call
// it.
func (c *Context) SetBackupPathStyle(style backupv1alpha1.BackupPathStyle) {
	c.backupPathStyle = style
}

// BackupPathStyle returns how the provider lays out backup paths:
// BackupPathNested unless it declares another style through
// BackupPathStyleProvider. Lay a path out with backupv1alpha1.BackupPathFor
// and this style, never by hand.
func (c *Context) BackupPathStyle() backupv1alpha1.BackupPathStyle {
	if c.backupPathStyle == "" {
		return backupv1alpha1.BackupPathNested
	}
	return c.backupPathStyle
}

// BackupLocation returns the directory in bs's bucket that holds backup's
// data, laid out in the provider's BackupPathStyle: the path recorded when
// the backup started, or for an imported backup its external path under
// BackupKeyPrefix of the storage's location. It is "" for a backup taken
// before the path was recorded.
func (c *Context) BackupLocation(backup *backupv1alpha1.Backup, bs *backupv1alpha1.BackupStorage) string {
	style := c.BackupPathStyle()
	if ext := backup.Spec.Origin.External; ext != nil && style == backupv1alpha1.BackupPathFlat {
		return backupv1alpha1.BackupKeyPrefix(style, bs.Spec.Location) + ext.Path
	}
	return style.Apply(BackupLocation(backup, bs))
}

// BackupLocation returns the directory in bs's bucket that holds backup's
// data: the path recorded when the backup started, or for an imported backup
// its external path under the storage's prefix. It is "" for a backup taken
// before the path was recorded.
//
// Deprecated: use Context.BackupLocation, which honours the provider's
// BackupPathStyle. BackupLocation assumes the nested one.
func BackupLocation(backup *backupv1alpha1.Backup, bs *backupv1alpha1.BackupStorage) string {
	if ext := backup.Spec.Origin.External; ext != nil {
		var prefix string
		if store := bs.Spec.Location.ObjectStore; store != nil {
			prefix = store.Prefix
		}
		return path.Join(prefix, ext.Path)
	}
	return backup.Status.Path
}

// RejectBackupStorage records that the provider cannot render the entry
// naming the BackupStorage name, because of err. The reason comes from the
// sentinel err wraps: backupstorageclasses.ErrUnsupportedParameter,
// ErrImplementationUnsupported or ErrTLSOptionUnsupported, and
// BackupStorageNotAccepted for anything else. The entry is reported on
// BackupConfigured and in status.backup.storages, and the provider applies
// the hold rule to it as for any entry that is not Apply.
func (c *Context) RejectBackupStorage(name string, err error) {
	reason := v1alpha1.ReasonBackupStorageNotAccepted
	switch {
	case errors.Is(err, backupstorageclasses.ErrUnsupportedParameter):
		reason = v1alpha1.ReasonUnsupportedParameter
	case errors.Is(err, ErrImplementationUnsupported):
		reason = v1alpha1.ReasonImplementationUnsupported
	case errors.Is(err, ErrTLSOptionUnsupported):
		reason = v1alpha1.ReasonTLSOptionUnsupported
	}
	message := fmt.Sprintf("backup storage %s: %v", name, err)
	if c.backupStorageRejections == nil {
		c.backupStorageRejections = make(map[string]backupStorageRejection)
	}
	c.backupStorageRejections[name] = backupStorageRejection{reason: reason, message: message}
	for i := range c.backupStorages {
		if c.backupStorages[i].Name() == name && c.backupStorages[i].Apply() {
			c.backupStorages[i].Reason, c.backupStorages[i].Message = reason, message
		}
	}
}

// backupStorageRejection is a verdict staged with RejectBackupStorage.
type backupStorageRejection struct {
	reason  string
	message string
}

// TLSSupport is what an engine can do with a BackupStorage's spec.tls.
type TLSSupport struct {
	// CustomCA is true when the engine can verify the server against a CA
	// bundle.
	CustomCA bool
	// SkipVerify is true when the engine can turn certificate verification
	// off.
	SkipVerify bool
}

// StorageTLS is the effective TLS of a BackupStorage.
type StorageTLS struct {
	// CA is the PEM bundle the server certificate is verified against; nil
	// means the system trust store.
	CA []byte
	// CASecretKeyRef is the Secret key CA was read from, for engines that
	// read the bundle themselves.
	CASecretKeyRef *common.SecretKeyRef
	// InsecureSkipVerify turns certificate verification off.
	InsecureSkipVerify bool
}

// BackupStorageTLS returns the effective TLS settings of bs, and an error
// wrapping ErrTLSOptionUnsupported when supported cannot honour them.
func (c *Context) BackupStorageTLS(bs *backupv1alpha1.BackupStorage, supported TLSSupport) (StorageTLS, error) {
	tls := bs.Spec.TLS
	if tls == nil {
		return StorageTLS{}, nil
	}
	if tls.InsecureSkipVerify {
		if !supported.SkipVerify {
			return StorageTLS{}, fmt.Errorf("%w: the engine cannot skip certificate verification", ErrTLSOptionUnsupported)
		}
		return StorageTLS{InsecureSkipVerify: true}, nil
	}
	if tls.CA == nil || tls.CA.SecretKeyRef == nil {
		return StorageTLS{}, nil
	}
	if !supported.CustomCA {
		return StorageTLS{}, fmt.Errorf("%w: the engine cannot verify against a custom CA", ErrTLSOptionUnsupported)
	}
	ref := tls.CA.SecretKeyRef
	secret := &corev1.Secret{}
	if err := c.client.Get(c.ctx, client.ObjectKey{Namespace: bs.Namespace, Name: ref.Name}, secret); err != nil {
		return StorageTLS{}, fmt.Errorf("get CA Secret %q of BackupStorage %q: %w", ref.Name, bs.Name, err)
	}
	pem, ok := secret.Data[ref.Key]
	if !ok || len(pem) == 0 {
		return StorageTLS{}, fmt.Errorf("CA Secret %q of BackupStorage %q has no key %q", ref.Name, bs.Name, ref.Key)
	}
	return StorageTLS{CA: pem, CASecretKeyRef: ref}, nil
}
