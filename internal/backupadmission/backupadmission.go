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

// Package backupadmission decides whether a Backup, Restore or BackupImport
// may be created. It reads only the verdicts core publishes, through
// backupv1alpha1.CurrentProviderVerdict and backupv1alpha1.CurrentVerdict,
// and never recomputes them. It rejects only what cannot run until an object
// it references changes: an Incompatible verdict or a terminal reference
// problem. Core's validating webhook and the HTTP API's validation handler
// both decide through it, so kubectl and the API cannot disagree.
package backupadmission

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// Reader reads the objects a decision depends on. A missing object is a
// NotFound error. kubernetes.KubernetesConnector implements it, and
// ClientReader adapts a controller-runtime client.Reader.
type Reader interface {
	GetInstance(ctx context.Context, key client.ObjectKey) (*corev1alpha1.Instance, error)
	GetBackup(ctx context.Context, key client.ObjectKey) (*backupv1alpha1.Backup, error)
	GetBackupClass(ctx context.Context, key client.ObjectKey) (*backupv1alpha1.BackupClass, error)
	GetBackupStorage(ctx context.Context, key client.ObjectKey) (*backupv1alpha1.BackupStorage, error)
	GetBackupStorageClass(ctx context.Context, key client.ObjectKey) (*backupv1alpha1.BackupStorageClass, error)
}

// Decision is whether an operation may be created.
type Decision struct {
	// Denied is why the operation cannot run until an object it references
	// changes: an Incompatible verdict or a terminal reference problem. It
	// satisfies errors.Is(err, controller.ErrInvalidReference). Nil when the
	// operation is admitted.
	Denied error
	// Missing are the objects the operation references that do not exist,
	// each satisfying errors.Is(err, controller.ErrInvalidReference).
	// Reconcilers hold the operation until they appear, so admission accepts
	// it with a warning.
	Missing []error
	// Pending are the verdicts that may still become Compatible without a
	// change of the objects, for example once a Provider claims its class.
	// Admission accepts the operation with a warning.
	Pending []string

	kind string
}

// Warnings returns the warnings an admitted operation is created with: one
// for each missing object and each pending verdict.
func (d Decision) Warnings() []string {
	w := make([]string, 0, len(d.Missing)+len(d.Pending))
	for _, err := range d.Missing {
		w = append(w, fmt.Sprintf("%v; the %s waits until it exists", err, d.kind))
	}
	return append(w, d.Pending...)
}

// Backup decides whether b may be created. Its class's verdict for the
// Provider of its Instance, or for an External backup the Provider that
// executes the class, and its storage's verdict for the class must not be
// Incompatible.
func Backup(ctx context.Context, r Reader, b *backupv1alpha1.Backup) (Decision, error) {
	j := newJudge(r, "Backup", b.GetNamespace())

	var provider string
	if ref := b.Spec.Origin.InstanceRef; ref != nil {
		in, err := j.instance(ctx, ref.Name)
		if err != nil {
			return Decision{}, err
		}
		if in != nil {
			provider = in.Spec.ProviderRef.Name
		}
	}
	storage, err := j.storage(ctx, b.Spec.StorageRef.Name)
	if err != nil {
		return Decision{}, err
	}
	class, err := j.class(ctx, b.Spec.ClassRef.Name)
	if err != nil {
		return Decision{}, err
	}
	if class == nil {
		return j.d, nil
	}

	if b.Spec.Origin.InstanceRef == nil {
		provider = class.Spec.ExecutorProvider()
	}
	if provider != "" {
		j.classFor(class, provider)
	}
	if storage != nil {
		if err := j.storageFor(ctx, storage, class); err != nil {
			return Decision{}, err
		}
	}
	return j.d, nil
}

// Restore decides whether rs may be created. Its class is the one the data
// was written with: the source Backup's or, for a point-in-time restore, that
// of the Instance owning the stream. The class's verdict for the Provider of
// the target Instance and, for a point-in-time restore, the source storage's
// verdict for the class must not be Incompatible. The source must be
// restorable: a Succeeded Backup, or a stream the Instance archives to the
// named storage with a class that supports point-in-time recovery.
func Restore(ctx context.Context, r Reader, rs *backupv1alpha1.Restore) (Decision, error) {
	j := newJudge(r, "Restore", rs.GetNamespace())

	className, err := j.restoreSource(ctx, rs)
	if err != nil {
		return Decision{}, err
	}
	var class *backupv1alpha1.BackupClass
	if className != "" {
		if class, err = j.class(ctx, className); err != nil {
			return Decision{}, err
		}
	}
	target, err := j.instance(ctx, rs.Spec.InstanceRef.Name)
	if err != nil {
		return Decision{}, err
	}
	var storage *backupv1alpha1.BackupStorage
	if pitr := rs.Spec.DataSource.PointInTime; rs.Spec.DataSource.Type == backupv1alpha1.DataSourceTypePointInTime && pitr != nil {
		if storage, err = j.storage(ctx, pitr.Source.StorageRef.Name); err != nil {
			return Decision{}, err
		}
	}

	if err := controller.ValidateRestorePITR(rs, class); err != nil {
		j.deny(err)
	}
	if class == nil {
		return j.d, nil
	}
	if target != nil {
		j.classFor(class, target.Spec.ProviderRef.Name)
	}
	if storage != nil {
		if err := j.storageFor(ctx, storage, class); err != nil {
			return Decision{}, err
		}
	}
	return j.d, nil
}

// Import decides whether imp may be created. It has no Instance: the class's
// verdict for the Provider that executes it and the storage's verdict for the
// class must not be Incompatible, and the class must support import.
func Import(ctx context.Context, r Reader, imp *backupv1alpha1.BackupImport) (Decision, error) {
	j := newJudge(r, "BackupImport", imp.GetNamespace())

	class, err := j.class(ctx, imp.Spec.ClassRef.Name)
	if err != nil {
		return Decision{}, err
	}
	storage, err := j.storage(ctx, imp.Spec.StorageRef.Name)
	if err != nil {
		return Decision{}, err
	}
	if class == nil {
		return j.d, nil
	}

	if err := controller.ValidateBackupImportSupported(class); err != nil {
		j.deny(err)
	}
	if provider := class.Spec.ExecutorProvider(); provider != "" {
		j.classFor(class, provider)
	}
	if storage != nil {
		if err := j.storageFor(ctx, storage, class); err != nil {
			return Decision{}, err
		}
	}
	return j.d, nil
}

// judge reads the objects of one operation and records its decision.
type judge struct {
	r         Reader
	namespace string
	// instances caches the Instances read so far, nil for missing ones, so
	// a Restore of an Instance's own stream reads and reports it once.
	instances map[string]*corev1alpha1.Instance
	d         Decision
}

func newJudge(r Reader, kind, namespace string) *judge {
	return &judge{
		r:         r,
		namespace: namespace,
		instances: make(map[string]*corev1alpha1.Instance),
		d:         Decision{kind: kind},
	}
}

// deny records why the operation is denied, keeping the first reason.
func (j *judge) deny(err error) {
	if j.d.Denied == nil {
		j.d.Denied = err
	}
}

// restoreSource checks the source of rs and returns the name of the class
// its data was written with, or "" when it cannot be resolved.
func (j *judge) restoreSource(ctx context.Context, rs *backupv1alpha1.Restore) (string, error) {
	ds := rs.Spec.DataSource
	switch ds.Type {
	case backupv1alpha1.DataSourceTypeBackup:
		if ds.Backup == nil {
			j.deny(fmt.Errorf("%w: spec.dataSource.backup is required when type is 'Backup'", controller.ErrInvalidReference))
			return "", nil
		}
		source, err := j.backup(ctx, ds.Backup.BackupRef.Name)
		if err != nil || source == nil {
			return "", err
		}
		// Nothing holds a Restore until its source Backup succeeds: its
		// executor would run it at once and fail.
		if err := controller.ValidateBackupSucceeded(source); err != nil {
			j.deny(err)
		}
		return source.Spec.ClassRef.Name, nil

	case backupv1alpha1.DataSourceTypePointInTime:
		if ds.PointInTime == nil {
			j.deny(fmt.Errorf("%w: spec.dataSource.pointInTime is required when type is 'PointInTime'", controller.ErrInvalidReference))
			return "", nil
		}
		name := controller.RestoreStreamInstanceName(rs)
		in, err := j.instance(ctx, name)
		if err != nil || in == nil {
			return "", err
		}
		if in.Spec.Backup == nil || in.Spec.Backup.ClassRef.Name == "" {
			j.deny(fmt.Errorf(
				"%w: instance '%s' has no backup class configured, so it has no stream to recover from",
				controller.ErrInvalidReference, name,
			))
			return "", nil
		}
		if err := controller.ValidatePITRStorage(ds.PointInTime, in); err != nil {
			j.deny(err)
		}
		return in.Spec.Backup.ClassRef.Name, nil
	}
	j.deny(fmt.Errorf("%w: unsupported dataSource type '%s'", controller.ErrInvalidReference, ds.Type))
	return "", nil
}

// classFor checks the class's verdict for provider, the Provider that would
// execute the operation.
func (j *judge) classFor(class *backupv1alpha1.BackupClass, provider string) {
	v := backupv1alpha1.CurrentProviderVerdict(class, provider)
	switch v.Compatibility {
	case backupv1alpha1.CompatibilityCompatible:
		return
	case backupv1alpha1.CompatibilityIncompatible:
		j.deny(fmt.Errorf("%w: BackupClass %q is Incompatible with Provider %q (%s): %s",
			controller.ErrBackupClassIncompatible, class.Name, provider, v.Reason, v.Message,
		))
		return
	case backupv1alpha1.CompatibilityPending:
	}
	// Pending, or a level this build does not know: only what is known to be
	// Incompatible is rejected.
	j.d.Pending = append(j.d.Pending, fmt.Sprintf("BackupClass %q is %s for Provider %q (%s): %s",
		class.Name, v.Compatibility, provider, v.Reason, v.Message,
	))
}

// storageFor checks the storage's verdict for the class.
func (j *judge) storageFor(ctx context.Context, storage *backupv1alpha1.BackupStorage, class *backupv1alpha1.BackupClass) error {
	name := storage.Spec.BackupStorageClassRef.Name
	storageClass, err := j.r.GetBackupStorageClass(ctx, client.ObjectKey{Name: name})
	switch {
	case apierrors.IsNotFound(err):
		storageClass = nil
	case err != nil:
		return fmt.Errorf("failed to get backup storage class '%s': %w", name, err)
	}
	v := backupv1alpha1.CurrentVerdict(storage, class, storageClass)
	switch v.Compatibility {
	case backupv1alpha1.CompatibilityCompatible:
		return nil
	case backupv1alpha1.CompatibilityIncompatible:
		j.deny(fmt.Errorf("%w: BackupStorage %q is Incompatible with BackupClass %q (%s): %s",
			controller.ErrBackupStorageIncompatible, storage.Name, class.Name, v.Reason, v.Message,
		))
		return nil
	case backupv1alpha1.CompatibilityPending:
	}
	j.d.Pending = append(j.d.Pending, fmt.Sprintf("BackupStorage %q is %s for BackupClass %q (%s): %s",
		storage.Name, v.Compatibility, class.Name, v.Reason, v.Message,
	))
	return nil
}

func (j *judge) instance(ctx context.Context, name string) (*corev1alpha1.Instance, error) {
	if in, ok := j.instances[name]; ok {
		return in, nil
	}
	in, err := j.r.GetInstance(ctx, client.ObjectKey{Namespace: j.namespace, Name: name})
	in, err = found(j, in, err, fmt.Sprintf("instance '%s'", name),
		fmt.Errorf("%w: instance '%s' does not exist", controller.ErrInstanceNotFound, name),
	)
	if err != nil {
		return nil, err
	}
	j.instances[name] = in
	return in, nil
}

func (j *judge) backup(ctx context.Context, name string) (*backupv1alpha1.Backup, error) {
	b, err := j.r.GetBackup(ctx, client.ObjectKey{Namespace: j.namespace, Name: name})
	return found(j, b, err, fmt.Sprintf("backup '%s'", name),
		fmt.Errorf("%w: '%s' in namespace '%s'", controller.ErrBackupNotFound, name, j.namespace),
	)
}

func (j *judge) class(ctx context.Context, name string) (*backupv1alpha1.BackupClass, error) {
	class, err := j.r.GetBackupClass(ctx, client.ObjectKey{Name: name})
	return found(j, class, err, fmt.Sprintf("backup class '%s'", name),
		fmt.Errorf("%w: '%s'", controller.ErrBackupClassNotFound, name),
	)
}

func (j *judge) storage(ctx context.Context, name string) (*backupv1alpha1.BackupStorage, error) {
	storage, err := j.r.GetBackupStorage(ctx, client.ObjectKey{Namespace: j.namespace, Name: name})
	return found(j, storage, err, fmt.Sprintf("backup storage '%s'", name),
		fmt.Errorf("%w: backup storage '%s' does not exist", controller.ErrBackupStorageNotFound, name),
	)
}

// found returns obj as read with err, or nil after recording notFound as
// missing when it does not exist. Any other error is returned, naming what
// was read.
func found[T any](j *judge, obj *T, err error, what string, notFound error) (*T, error) {
	switch {
	case err == nil:
		return obj, nil
	case apierrors.IsNotFound(err):
		j.d.Missing = append(j.d.Missing, notFound)
		return nil, nil //nolint:nilnil // a missing object is recorded, not an error
	}
	return nil, fmt.Errorf("failed to get %s: %w", what, err)
}

// ClientReader is a Reader over a controller-runtime client.Reader.
type ClientReader struct {
	client.Reader
}

// GetInstance implements Reader.
func (r ClientReader) GetInstance(ctx context.Context, key client.ObjectKey) (*corev1alpha1.Instance, error) {
	return get[corev1alpha1.Instance](ctx, r.Reader, key)
}

// GetBackup implements Reader.
func (r ClientReader) GetBackup(ctx context.Context, key client.ObjectKey) (*backupv1alpha1.Backup, error) {
	return get[backupv1alpha1.Backup](ctx, r.Reader, key)
}

// GetBackupClass implements Reader.
func (r ClientReader) GetBackupClass(ctx context.Context, key client.ObjectKey) (*backupv1alpha1.BackupClass, error) {
	return get[backupv1alpha1.BackupClass](ctx, r.Reader, key)
}

// GetBackupStorage implements Reader.
func (r ClientReader) GetBackupStorage(ctx context.Context, key client.ObjectKey) (*backupv1alpha1.BackupStorage, error) {
	return get[backupv1alpha1.BackupStorage](ctx, r.Reader, key)
}

// GetBackupStorageClass implements Reader.
func (r ClientReader) GetBackupStorageClass(ctx context.Context, key client.ObjectKey) (*backupv1alpha1.BackupStorageClass, error) {
	return get[backupv1alpha1.BackupStorageClass](ctx, r.Reader, key)
}

func get[T any, PT interface {
	*T
	client.Object
}](ctx context.Context, r client.Reader, key client.ObjectKey) (*T, error) {
	var o T
	obj := PT(&o)
	if err := r.Get(ctx, key, obj); err != nil {
		return nil, err
	}
	return obj, nil
}
