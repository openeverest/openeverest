package validation

import (
	"context"
	"errors"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
)

func (h *validateHandler) ListDatabaseClusterBackups(ctx context.Context, namespace, clusterName string) (*everestv1alpha1.DatabaseClusterBackupList, error) {
	return h.next.ListDatabaseClusterBackups(ctx, namespace, clusterName)
}

func (h *validateHandler) CreateDatabaseClusterBackup(ctx context.Context, req *everestv1alpha1.DatabaseClusterBackup) (*everestv1alpha1.DatabaseClusterBackup, error) {
	if err := validateBackupJobResources(req); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	return h.next.CreateDatabaseClusterBackup(ctx, req)
}

func validateBackupJobResources(req *everestv1alpha1.DatabaseClusterBackup) error {
	res := req.Spec.BackupJobResources
	if res == nil {
		return nil
	}
	if !res.CPU.IsZero() && res.CPU.Sign() < 0 {
		return errors.New("backupJobResources.cpu must be a positive quantity")
	}
	if !res.Memory.IsZero() && res.Memory.Sign() < 0 {
		return errors.New("backupJobResources.memory must be a positive quantity")
	}
	return nil
}

func (h *validateHandler) DeleteDatabaseClusterBackup(ctx context.Context, namespace, name string, req *api.DeleteDatabaseClusterBackupParams) error {
	return h.next.DeleteDatabaseClusterBackup(ctx, namespace, name, req)
}

func (h *validateHandler) GetDatabaseClusterBackup(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseClusterBackup, error) {
	return h.next.GetDatabaseClusterBackup(ctx, namespace, name)
}
