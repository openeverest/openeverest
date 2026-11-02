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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
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
// the backup class or a backup storage that an Instance with backups enabled
// names is missing or incompatible.
func (r *ProviderReconciler) missingBackupReferent(ctx context.Context, in *v1alpha1.Instance) (blocker, error) {
	spec := in.Spec.Backup
	if spec == nil || !spec.Enabled || !r.isBackupProvider() {
		return blocker{}, nil
	}
	if b, err := r.unusableBackupClass(ctx, spec.ClassRef.Name, in.Spec.ProviderRef.Name); err != nil || b.reason != "" {
		return b, err
	}
	for _, s := range spec.Storages {
		if b, err := missingBackupStorage(ctx, r.Client, in.Namespace, s.StorageRef.Name); err != nil || b.reason != "" {
			return b, err
		}
	}
	return blocker{}, nil
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

// missingBackupStorage returns a blocker naming the BackupStorage, or its
// credentials Secret, when it does not exist in namespace.
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
	secretName := credentialsSecretName(storage)
	if secretName == "" {
		return blocker{}, nil
	}
	secret := &metav1.PartialObjectMetadata{}
	secret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return blocker{}, fmt.Errorf("get secret %s: %w", secretName, err)
		}
		return blocker{
			reason: v1alpha1.ReasonBackupStorageCredentialsNotFound,
			message: fmt.Sprintf(
				"secret %s with the credentials of backup storage %s not found in %s",
				secretName, name, namespace,
			),
		}, nil
	}
	return blocker{}, nil
}

func credentialsSecretName(storage *backupv1alpha1.BackupStorage) string {
	if ref := storage.Spec.Authentication.CredentialsSecretRef; ref != nil {
		return ref.Name
	}
	return ""
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
