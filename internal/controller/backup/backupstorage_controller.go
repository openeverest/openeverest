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

package backup

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

const (
	backupStorageFinalizer = "backupstorage.backup.openeverest.io/in-use-protection"

	// instanceBackupStorageRefField is the field path used for indexing Instances
	// by their scheduled backup storage reference names.
	instanceBackupStorageRefField = ".spec.backup.storages.storageRef.name"

	// backupStorageNameField is the field path used for indexing Backups
	// by their storage name.
	backupStorageNameField = ".spec.storageName"
)

// BackupStorageReconciler reconciles a BackupStorage object
type BackupStorageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorages/finalizers,verbs=update
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backups,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=core.openeverest.io,resources=instances,verbs=get;list;watch

// Reconcile reconciles a BackupStorage by adding a finalizer to protect against
// deletion when in use by Instances or Backups, and adopting the credentials Secret as
// a child resource (setting a controller owner reference on it) so that the
// Secret is garbage-collected when the BackupStorage is deleted.
func (r *BackupStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	bs := &backupv1alpha1.BackupStorage{}
	if err := r.Get(ctx, req.NamespacedName, bs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !bs.GetDeletionTimestamp().IsZero() {
		err := r.handleFinalizer(ctx, bs)

		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(bs, backupStorageFinalizer) {
		controllerutil.AddFinalizer(bs, backupStorageFinalizer)
		if err := r.Update(ctx, bs); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	if bs.Spec.S3 == nil || bs.Spec.S3.CredentialsSecretName == "" {
		return ctrl.Result{}, nil
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      bs.Spec.S3.CredentialsSecretName,
		Namespace: req.Namespace,
	}, secret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Adopt the secret if it has no controller yet.
	if metav1.GetControllerOf(secret) == nil {
		logger.Info("Setting controller reference for the credentials secret", "secret", secret.Name)
		if err := controllerutil.SetControllerReference(bs, secret, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set controller reference on secret: %w", err)
		}
		if err := r.Update(ctx, secret); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update secret with controller reference: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

// handleFinalizer handles the deletion logic for BackupStorage.
// It checks if any Instances or Backups are using this BackupStorage.
// If in use, it returns successfully, keeping the BackupStorage in Terminating
// state. If not in use, it removes the finalizer to allow deletion.
func (r *BackupStorageReconciler) handleFinalizer(ctx context.Context, bs *backupv1alpha1.BackupStorage) error {
	logger := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(bs, backupStorageFinalizer) {
		return nil
	}

	// Check if any Instance is using this BackupStorage
	instances := &corev1alpha1.InstanceList{}
	if err := r.List(ctx, instances, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(instanceBackupStorageRefField, bs.Name),
		Namespace:     bs.Namespace,
	}); err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	if len(instances.Items) > 0 {
		logger.Info("Cannot delete BackupStorage: still in use by Instances",
			"backupStorage", bs.Name)
		return nil
	}

	// Check if any Backup is using this BackupStorage
	backups := &backupv1alpha1.BackupList{}
	if err := r.List(ctx, backups, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(backupStorageNameField, bs.Name),
		Namespace:     bs.Namespace,
	}); err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups.Items) > 0 {
		logger.Info("Cannot delete BackupStorage: still in use by Backups",
			"backupStorage", bs.Name)
		return nil
	}

	// No Instances or Backups are using this BackupStorage, safe to remove finalizer
	controllerutil.RemoveFinalizer(bs, backupStorageFinalizer)
	if err := r.Update(ctx, bs); err != nil {
		return fmt.Errorf("failed to remove finalizer: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.initIndexers(context.Background(), mgr); err != nil {
		return fmt.Errorf("init field indexers: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.BackupStorage{}).
		Named("backup-backupstorage").
		Owns(&corev1.Secret{}).
		Watches(&corev1alpha1.Instance{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueBackupStoragesFromInstance),
			builder.WithPredicates(
				instanceBackupChangePredicate(),
				predicate.GenerationChangedPredicate{},
			)).
		Watches(&backupv1alpha1.Backup{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueBackupStorageFromBackup),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// initIndexers registers the field indexers required by this controller.
func (r *BackupStorageReconciler) initIndexers(ctx context.Context, mgr ctrl.Manager) error {
	// Index Instances by backup storage reference name
	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1alpha1.Instance{},
		instanceBackupStorageRefField,
		func(obj client.Object) []string {
			instance, ok := obj.(*corev1alpha1.Instance)
			if !ok {
				return nil
			}
			if instance.Spec.Backup == nil || !instance.Spec.Backup.Enabled {
				return nil
			}

			// Return all storage reference names from the instance
			var storageNames []string
			for _, storage := range instance.Spec.Backup.Storages {
				if storage.StorageRef.Name != "" {
					storageNames = append(storageNames, storage.StorageRef.Name)
				}
			}
			return storageNames
		},
	); err != nil {
		return fmt.Errorf("indexing instance by backup storage ref: %w", err)
	}

	// Index Backups by storage name
	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&backupv1alpha1.Backup{},
		backupStorageNameField,
		func(obj client.Object) []string {
			backup, ok := obj.(*backupv1alpha1.Backup)
			if !ok {
				return nil
			}
			if backup.Spec.StorageName == "" {
				return nil
			}
			return []string{backup.Spec.StorageName}
		},
	); err != nil {
		return fmt.Errorf("indexing backup by storage name: %w", err)
	}

	return nil
}

// enqueueBackupStoragesFromInstance maps an Instance change to reconcile requests
// for all BackupStorage resources referenced by the Instance's backup config.
func (r *BackupStorageReconciler) enqueueBackupStoragesFromInstance(ctx context.Context, obj client.Object) []reconcile.Request {
	instance, ok := obj.(*corev1alpha1.Instance)
	if !ok {
		return nil
	}

	if instance.Spec.Backup == nil || !instance.Spec.Backup.Enabled {
		return nil
	}

	var requests []reconcile.Request
	for _, storage := range instance.Spec.Backup.Storages {
		if storage.StorageRef.Name != "" {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      storage.StorageRef.Name,
					Namespace: instance.Namespace,
				},
			})
		}
	}

	return requests
}

// enqueueBackupStorageFromBackup maps a Backup change to a reconcile request
// for the BackupStorage it references.
func (r *BackupStorageReconciler) enqueueBackupStorageFromBackup(ctx context.Context, obj client.Object) []reconcile.Request {
	backup, ok := obj.(*backupv1alpha1.Backup)
	if !ok {
		return nil
	}

	if backup.Spec.StorageName == "" {
		return nil
	}

	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      backup.Spec.StorageName,
				Namespace: backup.Namespace,
			},
		},
	}
}

// instanceBackupChangePredicate filters Instance events to only reconcile
// BackupStorages when backup configuration is present.
func instanceBackupChangePredicate() predicate.Funcs {
	hasBackup := func(obj client.Object) bool {
		instance, ok := obj.(*corev1alpha1.Instance)
		return ok && instance.Spec.Backup != nil && instance.Spec.Backup.Enabled
	}

	return predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return hasBackup(e.Object) },
		UpdateFunc:  func(e event.UpdateEvent) bool { return hasBackup(e.ObjectOld) || hasBackup(e.ObjectNew) },
		DeleteFunc:  func(e event.DeleteEvent) bool { return hasBackup(e.Object) },
		GenericFunc: func(e event.GenericEvent) bool { return hasBackup(e.Object) },
	}
}
