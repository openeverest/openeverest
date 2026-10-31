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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
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
	// Storage.Spec.Location.Bucket. Every engine writes under it.
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
// BackupStorage exists, is Accepted, the Instance's BackupClass can write to
// it, and it renders to the path recorded in status.backup.storages. Entries
// the provider rejected with RejectBackupStorage keep that verdict.
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
		rs.Path = storageclasses.BackupPath(bs.Spec.Location, c.in.Namespace, c.in.Name, c.in.UID, name)
		rs.Reason, rs.Message = judgeBackupStorage(class, bs, rs.Path, recorded[name])
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

func judgeBackupStorage(class *backupv1alpha1.BackupClass, bs *backupv1alpha1.BackupStorage, path, recorded string) (string, string) {
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
		if c := storageclasses.Compatible(class, bs); !c.Compatible {
			return v1alpha1.ReasonBackupStorageIncompatible,
				fmt.Sprintf("backup class %s cannot write to backup storage %s: %s", class.Name, bs.Name, c.Message)
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

// RejectBackupStorage records that the provider cannot render the entry
// naming the BackupStorage name, because of err. The reason comes from the
// sentinel err wraps: storageclasses.ErrUnsupportedParameter,
// ErrImplementationUnsupported or ErrTLSOptionUnsupported, and
// BackupStorageNotAccepted for anything else. The entry is reported on
// BackupConfigured and in status.backup.storages, and the provider applies
// the hold rule to it as for any entry that is not Apply.
func (c *Context) RejectBackupStorage(name string, err error) {
	reason := v1alpha1.ReasonBackupStorageNotAccepted
	switch {
	case errors.Is(err, storageclasses.ErrUnsupportedParameter):
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
