package rbac

import (
	"context"
	"errors"
	"fmt"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func (h *rbacHandler) ListBackupStoragesV1(ctx context.Context, namespace string) (*everestv1alpha1.BackupStorageList, error) {
	list, err := h.next.ListBackupStoragesV1(ctx, namespace)
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

func (h *rbacHandler) GetBackupStorageV1(ctx context.Context, namespace, name string) (*everestv1alpha1.BackupStorage, error) {
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	return h.next.GetBackupStorageV1(ctx, namespace, name)
}

func (h *rbacHandler) CreateBackupStorageV1(ctx context.Context, namespace string, req *api.CreateBackupStorageParams) (*everestv1alpha1.BackupStorage, error) {
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionCreate, rbac.ObjectName(namespace, req.Name)); err != nil {
		return nil, err
	}
	return h.next.CreateBackupStorageV1(ctx, namespace, req)
}

func (h *rbacHandler) UpdateBackupStorageV1(ctx context.Context, namespace, name string, req *api.UpdateBackupStorageParams) (*everestv1alpha1.BackupStorage, error) {
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionUpdate, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	return h.next.UpdateBackupStorageV1(ctx, namespace, name, req)
}

func (h *rbacHandler) DeleteBackupStorageV1(ctx context.Context, namespace, name string) error {
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionDelete, rbac.ObjectName(namespace, name)); err != nil {
		return err
	}
	return h.next.DeleteBackupStorageV1(ctx, namespace, name)
}
