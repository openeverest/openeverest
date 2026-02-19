// Package rbac provides the RBAC handler.
package rbac

import (
	"context"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupClasses proxies the request to the next handler.
func (h *rbacHandler) ListBackupClasses(ctx context.Context) (*backupv1alpha1.BackupClassList, error) {
	// Add RBAC checks here if needed in the future
	return h.next.ListBackupClasses(ctx)
}

// GetBackupClass proxies the request to the next handler.
func (h *rbacHandler) GetBackupClass(ctx context.Context, name string) (*backupv1alpha1.BackupClass, error) {
	// Add RBAC checks here if needed in the future
	return h.next.GetBackupClass(ctx, name)
}
