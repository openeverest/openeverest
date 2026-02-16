// Package rbac provides the RBAC handler.
package rbac

import (
	"context"

	"github.com/openeverest/openeverest/v2/pkg/apis/v1alpha1"
)

// GetBackup proxies the request to the next handler.
func (h *rbacHandler) GetBackup(ctx context.Context, namespace, name string) (*v1alpha1.Backup, error) {
	// Add RBAC checks here if needed in the future
	return h.next.GetBackup(ctx, namespace, name)
}

// CreateBackup proxies the request to the next handler.
func (h *rbacHandler) CreateBackup(ctx context.Context, backup *v1alpha1.Backup) (*v1alpha1.Backup, error) {
	// Add RBAC checks here if needed in the future
	return h.next.CreateBackup(ctx, backup)
}

// DeleteBackup proxies the request to the next handler.
func (h *rbacHandler) DeleteBackup(ctx context.Context, namespace, name string) error {
	// Add RBAC checks here if needed in the future
	return h.next.DeleteBackup(ctx, namespace, name)
}
