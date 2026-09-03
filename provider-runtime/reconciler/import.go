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

	corev1 "k8s.io/api/core/v1"
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
	logger := log.FromContext(ctx).WithValues("provider", r.providerName, "backupImport", req.NamespacedName)

	imp := &backupv1alpha1.BackupImport{}
	if err := r.client.Get(ctx, req.NamespacedName, imp); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if !imp.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	_, ours, err := resolveImportOwnership(ctx, r.client, imp, r.providerName)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !ours {
		return reconcile.Result{}, nil
	}

	// Already finished successfully: nothing to do until the spec changes.
	// The spec is immutable, so a Succeeded import is terminal.
	if imp.Status.State == backupv1alpha1.BackupImportStateSucceeded {
		return reconcile.Result{}, nil
	}

	storage, accessKeyID, secretAccessKey, err := resolveImportStorage(ctx, r.client, imp)
	if err != nil {
		return r.fail(ctx, imp, fmt.Sprintf("resolve storage: %v", err))
	}

	s3c, err := controller.NewS3Client(storage.Spec.S3, accessKeyID, secretAccessKey)
	if err != nil {
		return r.fail(ctx, imp, fmt.Sprintf("build S3 client: %v", err))
	}

	backups, err := r.importer.ImportBackups(ctx, imp, storage, s3c)
	if err != nil {
		return r.fail(ctx, imp, fmt.Sprintf("import backups: %v", err))
	}

	toCreate, err := r.dedupBackups(ctx, imp.Namespace, backups)
	if err != nil {
		return r.fail(ctx, imp, fmt.Sprintf("dedup backups: %v", err))
	}

	created, err := r.createBackups(ctx, imp, toCreate)
	if err != nil {
		return r.fail(ctx, imp, fmt.Sprintf("create backups: %v", err))
	}

	imp.Status.State = backupv1alpha1.BackupImportStateSucceeded
	imp.Status.Message = ""
	imp.Status.DiscoveredCount = int32(len(backups)) //nolint:gosec // count is bounded by storage listing
	imp.Status.CreatedCount = int32(created)         //nolint:gosec // count is bounded by discovered
	imp.Status.LastObservedGeneration = imp.Generation
	if err := r.client.Status().Update(ctx, imp); err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("backup import completed", "discovered", len(backups), "created", created)
	return reconcile.Result{}, nil
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

// createBackups creates the given Backup CRs, defaulting their namespace and
// stamping the originating BackupImport name label. Callers are expected to
// have already removed duplicates via dedupBackups; AlreadyExists is still
// tolerated as a safety net against concurrent creates. It returns the number
// of Backup CRs that exist after the run.
func (r *backupImportReconciler) createBackups(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
	backups []*backupv1alpha1.Backup,
) (int, error) {
	count := 0
	for _, backup := range backups {
		if backup.Namespace == "" {
			backup.Namespace = imp.Namespace
		}

		if backup.Labels == nil {
			backup.Labels = map[string]string{}
		}

		backup.Labels[controller.BackupImportNameLabel] = imp.Name

		if err := r.client.Create(ctx, backup); err != nil {
			if apierrors.IsAlreadyExists(err) {
				count++
				continue
			}

			return count, fmt.Errorf("create Backup %q: %w", backup.Name, err)
		}

		count++
	}
	return count, nil
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

// fail records a terminal failure on the BackupImport status.
func (r *backupImportReconciler) fail(ctx context.Context, imp *backupv1alpha1.BackupImport, message string) (reconcile.Result, error) {
	imp.Status.State = backupv1alpha1.BackupImportStateFailed
	imp.Status.Message = message
	imp.Status.LastObservedGeneration = imp.Generation
	if err := r.client.Status().Update(ctx, imp); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// resolveImportOwnership checks the referenced BackupClass uses executionMode
// "ProviderManaged", and lists this provider in supportedProviders.
func resolveImportOwnership(
	ctx context.Context,
	c client.Client,
	imp *backupv1alpha1.BackupImport,
	providerName string,
) (*backupv1alpha1.BackupClass, bool, error) {
	bc := &backupv1alpha1.BackupClass{}
	if err := c.Get(ctx, client.ObjectKey{Name: imp.Spec.ClassRef.Name}, bc); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to get BackupClass: %w", err)
	}
	if bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeProviderManaged {
		return bc, false, nil
	}
	return bc, bc.Spec.SupportedProviders.Has(providerName), nil
}

// resolveImportStorage fetches the BackupStorage referenced by the import and
// reads its S3 credentials from the referenced Secret.
func resolveImportStorage(
	ctx context.Context,
	c client.Client,
	imp *backupv1alpha1.BackupImport,
) (*backupv1alpha1.BackupStorage, string, string, error) {
	bs := &backupv1alpha1.BackupStorage{}
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: imp.Namespace,
		Name:      imp.Spec.StorageRef.Name,
	}, bs); err != nil {
		return nil, "", "", fmt.Errorf("get BackupStorage %q: %w", imp.Spec.StorageRef.Name, err)
	}
	if bs.Spec.S3 == nil {
		return nil, "", "", fmt.Errorf("BackupStorage %q is not an S3 storage", bs.Name)
	}

	var accessKeyID, secretAccessKey string
	if ref := bs.Spec.S3.CredentialsSecretRef.Name; ref != "" {
		secret := &corev1.Secret{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: bs.Namespace, Name: ref}, secret); err != nil {
			return nil, "", "", fmt.Errorf("get credentials secret %q: %w", ref, err)
		}
		accessKeyID = string(secret.Data["AWS_ACCESS_KEY_ID"])
		secretAccessKey = string(secret.Data["AWS_SECRET_ACCESS_KEY"])
	}
	return bs, accessKeyID, secretAccessKey, nil
}
