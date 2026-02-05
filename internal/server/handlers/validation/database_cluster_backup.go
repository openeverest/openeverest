package validation

import (
	"context"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
)

func (h *validateHandler) ListDatabaseClusterBackups(ctx context.Context, namespace, clusterName string) (*everestv1alpha1.DatabaseClusterBackupList, error) {
	return h.next.ListDatabaseClusterBackups(ctx, namespace, clusterName)
}

func (h *validateHandler) CreateDatabaseClusterBackup(ctx context.Context, req *everestv1alpha1.DatabaseClusterBackup) (*everestv1alpha1.DatabaseClusterBackup, error) {
	return h.next.CreateDatabaseClusterBackup(ctx, req)
}

func (h *validateHandler) DeleteDatabaseClusterBackup(ctx context.Context, namespace, name string, req *api.DeleteDatabaseClusterBackupParams) error {
	return h.next.DeleteDatabaseClusterBackup(ctx, namespace, name, req)
}

func (h *validateHandler) GetDatabaseClusterBackup(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseClusterBackup, error) {
	return h.next.GetDatabaseClusterBackup(ctx, namespace, name)
}
