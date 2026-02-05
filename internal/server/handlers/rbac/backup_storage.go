package rbac

import (
	"context"
	"errors"
	"fmt"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/pkg/rbac"
)

func (h *rbacHandler) ListBackupStorages(ctx context.Context, namespace string) (*everestv1alpha1.BackupStorageList, error) {
	list, err := h.next.ListBackupStorages(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("ListBackupStorages failed: %w", err)
	}
	filtered := []everestv1alpha1.BackupStorage{}
	for _, bs := range list.Items {
		if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead,
			rbac.ObjectName(namespace, bs.GetName()),
		); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, bs)
	}
	list.Items = filtered
	return list, nil
}

func (h *rbacHandler) GetBackupStorage(ctx context.Context, namespace, name string) (*everestv1alpha1.BackupStorage, error) {
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	return h.next.GetBackupStorage(ctx, namespace, name)
}

func (h *rbacHandler) CreateBackupStorage(ctx context.Context, namespace string, req *api.CreateBackupStorageParams) (*everestv1alpha1.BackupStorage, error) {
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionCreate, rbac.ObjectName(namespace, req.Name)); err != nil {
		return nil, err
	}
	return h.next.CreateBackupStorage(ctx, namespace, req)
}

func (h *rbacHandler) UpdateBackupStorage(ctx context.Context, namespace, name string, req *api.UpdateBackupStorageParams) (*everestv1alpha1.BackupStorage, error) {
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionUpdate, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	return h.next.UpdateBackupStorage(ctx, namespace, name, req)
}

func (h *rbacHandler) DeleteBackupStorage(ctx context.Context, namespace, name string) error {
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionDelete, rbac.ObjectName(namespace, name)); err != nil {
		return err
	}
	return h.next.DeleteBackupStorage(ctx, namespace, name)
}
