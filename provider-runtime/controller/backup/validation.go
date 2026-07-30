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

// Package backup holds Backup/Restore reference-validation logic shared
// between the API server's validation handlers and the provider runtime's
// DataSource-seeding controller, so the existence/state/compatibility checks
// are defined once and reused rather than re-implemented per caller.
package backup

import (
	"context"
	"errors"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// Sentinel errors returned by the resolve helpers in this file. Callers use
// errors.Is to classify a failure (e.g. to pick an HTTP status or a
// condition Reason) without re-deriving the underlying check.
var (
	// ErrBackupNotFound is returned when a referenced Backup does not exist.
	ErrBackupNotFound = errors.New("backup not found")
	// ErrBackupNotSucceeded is returned when a referenced Backup exists but
	// has not reached the Succeeded state.
	ErrBackupNotSucceeded = errors.New("backup not succeeded")
	// ErrBackupClassNotFound is returned when a referenced BackupClass does
	// not exist.
	ErrBackupClassNotFound = errors.New("backup class not found")
	// ErrProviderUnsupported is returned when a BackupClass does not list a
	// provider in its SupportedProviders.
	ErrProviderUnsupported = errors.New("provider not supported by backup class")
	// ErrRestorePITRUnsupported is returned by ValidateRestorePITR when a
	// Restore requests point-in-time recovery but the resolved
	// ProviderManaged BackupClass does not advertise supportsPITR.
	ErrRestorePITRUnsupported = errors.New("point-in-time recovery is not supported")
)

// GetBackupFunc resolves a Backup by name. Callers close over their own
// client/namespace so this package stays agnostic of how the lookup is
// wired (a direct Kubernetes client vs. a reconciler Context).
type GetBackupFunc func(ctx context.Context, name string) (*backupv1alpha1.Backup, error)

// GetBackupClassFunc resolves a cluster-scoped BackupClass by name.
type GetBackupClassFunc func(ctx context.Context, name string) (*backupv1alpha1.BackupClass, error)

// ResolveSucceededBackup fetches the Backup named "name" and returns it only
// if it exists and has reached the Succeeded state. On a state failure the
// fetched Backup is still returned alongside the error so callers can
// include its current state in their own message.
//
// This is the shared check behind restore creation (a Restore may only
// reference a completed Backup) and DataSource seeding (an Instance may
// only be seeded from a completed Backup).
func ResolveSucceededBackup(ctx context.Context, name string, get GetBackupFunc) (*backupv1alpha1.Backup, error) {
	backup, err := get(ctx, name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: backup '%s' does not exist", ErrBackupNotFound, name)
		}
		return nil, fmt.Errorf("failed to get backup '%s': %w", name, err)
	}
	if backup.Status.State != backupv1alpha1.BackupStateSucceeded {
		return backup, fmt.Errorf("%w: backup '%s' is in state '%s', must be '%s' to restore from it",
			ErrBackupNotSucceeded, name, backup.Status.State, backupv1alpha1.BackupStateSucceeded)
	}
	return backup, nil
}

// ResolveBackupClass fetches the cluster-scoped BackupClass named "name".
func ResolveBackupClass(ctx context.Context, name string, get GetBackupClassFunc) (*backupv1alpha1.BackupClass, error) {
	bc, err := get(ctx, name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: backup class '%s' does not exist", ErrBackupClassNotFound, name)
		}
		return nil, fmt.Errorf("failed to get backup class '%s': %w", name, err)
	}
	return bc, nil
}

// ValidateClassSupportsProvider returns ErrProviderUnsupported unless bc
// lists provider in its SupportedProviders.
func ValidateClassSupportsProvider(bc *backupv1alpha1.BackupClass, provider string) error {
	if !bc.Spec.SupportedProviders.Has(provider) {
		return fmt.Errorf("%w: backup class '%s' does not support provider '%s'", ErrProviderUnsupported, bc.GetName(), provider)
	}
	return nil
}

// ValidateRestorePITR checks whether the point-in-time recovery options on a
// Restore are acceptable for the given BackupClass:
//   - When the resolved class is ProviderManaged, PITR may only be requested
//     if the class advertises .spec.providerManaged.supportsPITR.
//   - Job-mode classes have no PITR capability declaration; they are not
//     gated here.
//
// A nil class or a Restore without PITR options passes. The CRD's CEL rule
// already requires date when type is "date"; the check is repeated here for
// defense in depth on paths that bypass admission.
func ValidateRestorePITR(restore *backupv1alpha1.Restore, bc *backupv1alpha1.BackupClass) error {
	if restore == nil || restore.Spec.DataSource.Backup == nil || restore.Spec.DataSource.Backup.PITR == nil {
		return nil
	}
	pitr := restore.Spec.DataSource.Backup.PITR
	if pitr.Type == backupv1alpha1.PITRTypeDate && pitr.Date == nil {
		return fmt.Errorf(
			"spec.dataSource.backup.pitr.date must be set when type is %q",
			backupv1alpha1.PITRTypeDate,
		)
	}
	if bc == nil || bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeProviderManaged {
		return nil
	}
	if bc.Spec.ProviderManaged == nil || !bc.Spec.ProviderManaged.SupportsPITR {
		return fmt.Errorf(
			"%w: BackupClass %q does not declare providerManaged.supportsPITR",
			ErrRestorePITRUnsupported, bc.Name,
		)
	}
	return nil
}
