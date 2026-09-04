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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// backupImportReconciler reconciles BackupImport CRs and creates
// Backup CR deduped on (storageRef, path) so re-running BackupImport is
// idempotent.
type backupImportReconciler struct {
	client       client.Client
	importer     controller.BackupImporter
	providerName string
}

func setupBackupImportReconciler(mgr ctrl.Manager, bi controller.BackupImporter, providerName string) error {
	r := &backupImportReconciler{
		client:       mgr.GetClient(),
		importer:     bi,
		providerName: providerName,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.BackupImport{}).
		Named(providerName + "-backup-import").
		Complete(r)
}

func (r *backupImportReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	imp := &backupv1alpha1.BackupImport{}
	if err := r.client.Get(ctx, req.NamespacedName, imp); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if !imp.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	ours, err := resolveImportOwnership(ctx, r.client, imp, r.providerName)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !ours {
		return reconcile.Result{}, nil
	}

	// Succeeded and Failed are terminal states, so we don't re-run the import.
	if imp.Status.State == backupv1alpha1.BackupImportStateSucceeded ||
		imp.Status.State == backupv1alpha1.BackupImportStateFailed {
		return reconcile.Result{}, nil
	}

	storage := &backupv1alpha1.BackupStorage{}
	if err := r.client.Get(ctx, client.ObjectKey{
		Namespace: imp.Namespace,
		Name:      imp.Spec.StorageRef.Name,
	}, storage); err != nil {
		return r.updateErrorStatus(ctx, imp, fmt.Errorf("failed to get BackupStorage %q: %w", imp.Spec.StorageRef.Name, err))
	}

	result, err := r.importer.ImportBackups(ctx, imp, storage)
	if err != nil {
		return r.updateErrorStatus(ctx, imp, fmt.Errorf("failed to import backups: %w", err))
	}

	return r.applyImportResult(ctx, imp, result)
}

// applyImportResult records a terminal failure or creates the discovered
// Backups and marks the import Succeeded.
func (r *backupImportReconciler) applyImportResult(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
	result controller.BackupImportExecutionStatus,
) (reconcile.Result, error) {
	switch result.State {
	case backupv1alpha1.BackupImportStateError:
		return r.updateErrorStatus(ctx, imp, fmt.Errorf("importer returned transient error: %s", result.Message))
	case backupv1alpha1.BackupImportStateFailed:
		// The provider observed a terminal, non-retryable condition.
		// Record it and stop; the spec is immutable, so the import
		// re-runs only if the CR is recreated.
		imp.Status.State = result.State
		imp.Status.Message = result.Message
		imp.Status.LastObservedGeneration = imp.Generation
		if err := r.client.Status().Update(ctx, imp); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	case backupv1alpha1.BackupImportStateSucceeded:
		imp.Status.DiscoveredCount = int32(len(result.Backups)) //nolint:gosec // count is bounded by storage listing

		toCreate, err := r.dedupBackups(ctx, imp.Namespace, result.Backups)
		if err != nil {
			return r.updateErrorStatus(ctx, imp, fmt.Errorf("failed to dedup backups: %w", err))
		}

		created, err := r.createBackups(ctx, imp, toCreate)
		if err != nil {
			imp.Status.CreatedCount = int32(created) //nolint:gosec // count is bounded by discovered
			return r.updateErrorStatus(ctx, imp, fmt.Errorf("failed to create backups: %w", err))
		}

		imp.Status.State = result.State
		imp.Status.Message = result.Message
		imp.Status.CreatedCount = int32(created) //nolint:gosec // count is bounded by discovered
		imp.Status.LastObservedGeneration = imp.Generation
		if err := r.client.Status().Update(ctx, imp); err != nil {
			return reconcile.Result{}, err
		}

		log.FromContext(ctx).Info("backup import completed", "discovered", len(result.Backups), "created", created)
		return reconcile.Result{}, nil
	default:
		// The unknown state is reported as an error.
		return r.updateErrorStatus(ctx, imp, fmt.Errorf("importer returned unknown state %q", result.State))
	}
}

// updateErrorStatus records a transient error on the BackupImport status and returns
// the cause so the controller requeues with backoff.
func (r *backupImportReconciler) updateErrorStatus(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
	cause error,
) (reconcile.Result, error) {
	imp.Status.State = backupv1alpha1.BackupImportStateError
	imp.Status.Message = cause.Error()
	imp.Status.LastObservedGeneration = imp.Generation
	_ = r.client.Status().Update(ctx, imp)
	return reconcile.Result{}, cause
}

// dedupBackups filters out the discovered backups whose data has already been
// imported, keying on each backup's (storageRef, external.path) pair.
func (r *backupImportReconciler) dedupBackups(
	ctx context.Context,
	namespace string,
	backups []*backupv1alpha1.Backup,
) ([]*backupv1alpha1.Backup, error) {
	seen, err := r.existingExternalBackups(ctx, namespace)
	if err != nil {
		return nil, err
	}

	toCreate := make([]*backupv1alpha1.Backup, 0, len(backups))
	for _, backup := range backups {
		if backup == nil {
			continue
		}

		key := externalBackupKey(backup.Spec.StorageRef.Name, backup.Spec.Origin.External)
		if key != "" {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
		}

		toCreate = append(toCreate, backup)
	}
	return toCreate, nil
}

// existingExternalBackups lists the external Backups already present in the
// namespace and indexes them by their (storageRef, external.path) identity so
// dedupBackups can skip data that has already been imported.
func (r *backupImportReconciler) existingExternalBackups(ctx context.Context, namespace string) (map[string]struct{}, error) {
	list := &backupv1alpha1.BackupList{}
	if err := r.client.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("list existing Backups: %w", err)
	}

	index := make(map[string]struct{}, len(list.Items))
	for i := range list.Items {
		b := &list.Items[i]
		if key := externalBackupKey(b.Spec.StorageRef.Name, b.Spec.Origin.External); key != "" {
			index[key] = struct{}{}
		}
	}
	return index, nil
}

// externalBackupKey builds the dedup identity for an external Backup from its
// storage name and path. It returns the empty string when the backup is not
// external (nil origin), so non-external backups never collide in the index.
func externalBackupKey(storageName string, external *backupv1alpha1.BackupOriginExternal) string {
	if external == nil {
		return ""
	}
	return storageName + "\x00" + external.Path
}

// createBackups creates the given Backup CRs, adding the originating
// BackupImport name label. It returns the number of Backup CRs created.
func (r *backupImportReconciler) createBackups(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
	backups []*backupv1alpha1.Backup,
) (int, error) {
	count := 0
	for _, backup := range backups {
		if backup.Labels == nil {
			backup.Labels = map[string]string{}
		}

		// Imported Backups always live alongside the BackupImport; override
		// namespace the importer set so imports stay in-namespace.
		backup.Namespace = imp.Namespace
		backup.Labels[controller.BackupImportNameLabel] = imp.Name

		if err := r.client.Create(ctx, backup); err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}

			return count, fmt.Errorf("create Backup %q: %w", backup.Name, err)
		}

		count++
	}
	return count, nil
}

// resolveImportOwnership checks the referenced BackupClass uses executionMode
// "ProviderManaged", and lists this provider in supportedProviders.
func resolveImportOwnership(
	ctx context.Context,
	c client.Client,
	imp *backupv1alpha1.BackupImport,
	providerName string,
) (bool, error) {
	bc := &backupv1alpha1.BackupClass{}
	if err := c.Get(ctx, client.ObjectKey{Name: imp.Spec.ClassRef.Name}, bc); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get BackupClass: %w", err)
	}
	if bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeProviderManaged {
		return false, nil
	}
	return bc.Spec.SupportedProviders.Has(providerName), nil
}
