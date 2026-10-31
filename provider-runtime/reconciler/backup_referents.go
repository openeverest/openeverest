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

package reconciler

import (
	"context"
	"fmt"
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// isBackupProvider reports whether the provider handles backups, which is
// also when its RBAC lets the runtime read backup classes and storages.
func (r *ProviderReconciler) isBackupProvider() bool {
	_, ok := r.provider.(controller.BackupProvider)
	return ok
}

// missingBackupReferent returns the BackupConfigured reason and message when
// the backup class an Instance with backups enabled names is missing or
// incompatible. Backup storages are judged per entry instead, so a bad
// entry never blocks the rest of the Instance.
func (r *ProviderReconciler) missingBackupReferent(ctx context.Context, in *v1alpha1.Instance) (blocker, error) {
	spec := in.Spec.Backup
	if spec == nil || !spec.Enabled || !r.isBackupProvider() {
		return blocker{}, nil
	}
	return r.unusableBackupClass(ctx, spec.ClassRef.Name, in.Spec.ProviderRef.Name)
}

func (r *ProviderReconciler) unusableBackupClass(ctx context.Context, className, providerName string) (blocker, error) {
	class := &backupv1alpha1.BackupClass{}
	if err := r.Get(ctx, client.ObjectKey{Name: className}, class); err != nil {
		if !apierrors.IsNotFound(err) {
			return blocker{}, fmt.Errorf("get backup class %s: %w", className, err)
		}
		return blocker{
			reason:  v1alpha1.ReasonBackupClassNotFound,
			message: fmt.Sprintf("backup class %s not found", className),
		}, nil
	}
	if class.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeProviderManaged {
		return blocker{
			reason:  v1alpha1.ReasonBackupClassIncompatible,
			message: fmt.Sprintf("backup class %s runs backups as Jobs and can't be an instance's backup class", className),
		}, nil
	}
	if !class.Spec.SupportedProviders.Has(providerName) {
		return blocker{
			reason:  v1alpha1.ReasonBackupClassIncompatible,
			message: fmt.Sprintf("backup class %s does not support provider %s", className, providerName),
		}, nil
	}
	return blocker{}, nil
}

// missingBackupStorage returns a blocker naming the BackupStorage when it
// does not exist in namespace or is not Accepted.
func missingBackupStorage(ctx context.Context, c client.Reader, namespace, name string) (blocker, error) {
	storage := &backupv1alpha1.BackupStorage{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, storage); err != nil {
		if !apierrors.IsNotFound(err) {
			return blocker{}, fmt.Errorf("get backup storage %s: %w", name, err)
		}
		return blocker{
			reason:  v1alpha1.ReasonBackupStorageNotFound,
			message: fmt.Sprintf("backup storage %s not found in %s", name, namespace),
		}, nil
	}
	accepted := meta.FindStatusCondition(storage.Status.Conditions, backupv1alpha1.BackupStorageConditionAccepted)
	if accepted == nil || accepted.Status != metav1.ConditionTrue {
		message := fmt.Sprintf("backup storage %s has not been validated yet", name)
		if accepted != nil {
			message = fmt.Sprintf("backup storage %s is not accepted: %s: %s", name, accepted.Reason, accepted.Message)
		}
		return blocker{reason: v1alpha1.ReasonBackupStorageNotAccepted, message: message}, nil
	}
	return blocker{}, nil
}

// instanceBackupPath returns the directory in the bucket of the BackupStorage
// name that the Instance writes its backups to, or "" while the storage does
// not exist.
func instanceBackupPath(ctx context.Context, c client.Reader, in *v1alpha1.Instance, name string) (string, error) {
	storage := &backupv1alpha1.BackupStorage{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: in.Namespace, Name: name}, storage); err != nil {
		return "", client.IgnoreNotFound(err)
	}
	return storageclasses.BackupPath(storage.Spec.Location, in.Namespace, in.Name, in.UID, name), nil
}

func backupStorageNames(in *v1alpha1.Instance) []string {
	if in.Spec.Backup == nil {
		return nil
	}
	names := make([]string, 0, len(in.Spec.Backup.Storages))
	for _, s := range in.Spec.Backup.Storages {
		names = append(names, s.StorageRef.Name)
	}
	return names
}

// instancesForBackupClass enqueues the Instances of this provider whose
// spec.backup.classRef names the BackupClass.
func (r *ProviderReconciler) instancesForBackupClass(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.instancesMatching(ctx, "", func(in *v1alpha1.Instance) bool {
		return in.Spec.Backup != nil && in.Spec.Backup.ClassRef.Name == obj.GetName()
	})
}

// instancesForBackupStorage enqueues the Instances of this provider whose
// spec.backup.storages names the BackupStorage.
func (r *ProviderReconciler) instancesForBackupStorage(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.instancesMatching(ctx, obj.GetNamespace(), func(in *v1alpha1.Instance) bool {
		return slices.Contains(backupStorageNames(in), obj.GetName())
	})
}

// storagesUsingSecret returns the names of the BackupStorages in namespace
// that reference the Secret, for credentials or a CA bundle.
func storagesUsingSecret(ctx context.Context, c client.Reader, namespace, secret string) []string {
	list := &backupv1alpha1.BackupStorageList{}
	if err := c.List(ctx, list, client.InNamespace(namespace)); err != nil {
		log.FromContext(ctx).Error(err, "Listing backup storages to enqueue failed")
		return nil
	}
	var names []string
	for i := range list.Items {
		if slices.Contains(list.Items[i].Spec.SecretNames(), secret) {
			names = append(names, list.Items[i].Name)
		}
	}
	return names
}

// acceptanceChanged passes BackupStorage events that can block or unblock a
// referrer: creation, deletion and a change of the Accepted condition.
func acceptanceChanged() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return storageAccepted(e.ObjectOld) != storageAccepted(e.ObjectNew)
		},
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

func storageAccepted(obj client.Object) metav1.ConditionStatus {
	bs, ok := obj.(*backupv1alpha1.BackupStorage)
	if !ok {
		return metav1.ConditionUnknown
	}
	if c := meta.FindStatusCondition(bs.Status.Conditions, backupv1alpha1.BackupStorageConditionAccepted); c != nil {
		return c.Status
	}
	return metav1.ConditionUnknown
}
