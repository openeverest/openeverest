package validation

import (
	"context"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

func (h *validateHandler) ListDatabaseClusterRestores(ctx context.Context, namespace, clusterName string) (*everestv1alpha1.DatabaseClusterRestoreList, error) {
	return h.next.ListDatabaseClusterRestores(ctx, namespace, clusterName)
}

func (h *validateHandler) CreateDatabaseClusterRestore(ctx context.Context, req *everestv1alpha1.DatabaseClusterRestore) (*everestv1alpha1.DatabaseClusterRestore, error) {
	return h.next.CreateDatabaseClusterRestore(ctx, req)
}

func (h *validateHandler) DeleteDatabaseClusterRestore(ctx context.Context, namespace, name string) error {
	return h.next.DeleteDatabaseClusterRestore(ctx, namespace, name)
}

func (h *validateHandler) GetDatabaseClusterRestore(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseClusterRestore, error) {
	return h.next.GetDatabaseClusterRestore(ctx, namespace, name)
}

func (h *validateHandler) UpdateDatabaseClusterRestore(ctx context.Context, req *everestv1alpha1.DatabaseClusterRestore) (*everestv1alpha1.DatabaseClusterRestore, error) {
	return h.next.UpdateDatabaseClusterRestore(ctx, req)
}
