// Package validation provides the validation handler.
package validation

import (
	"context"

	"github.com/openeverest/openeverest/v2/pkg/apis/v1alpha1"
)

// ListBackupClasses proxies the request to the next handler.
func (h *validateHandler) ListBackupClasses(ctx context.Context) (*v1alpha1.BackupClassList, error) {
	return h.next.ListBackupClasses(ctx)
}

// GetBackupClass proxies the request to the next handler.
func (h *validateHandler) GetBackupClass(ctx context.Context, name string) (*v1alpha1.BackupClass, error) {
	return h.next.GetBackupClass(ctx, name)
}
