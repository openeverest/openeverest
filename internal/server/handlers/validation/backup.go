// Package validation provides the validation handler.
package validation

import (
	"context"

	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// GetBackup proxies the request to the next handler.
func (h *validateHandler) GetBackup(ctx context.Context, namespace, name string) (*v1alpha1.Backup, error) {
	return h.next.GetBackup(ctx, namespace, name)
}

// CreateBackup proxies the request to the next handler.
func (h *validateHandler) CreateBackup(ctx context.Context, backup *v1alpha1.Backup) (*v1alpha1.Backup, error) {
	// Add validation here if needed in the future
	return h.next.CreateBackup(ctx, backup)
}

// DeleteBackup proxies the request to the next handler.
func (h *validateHandler) DeleteBackup(ctx context.Context, namespace, name string) error {
	return h.next.DeleteBackup(ctx, namespace, name)
}
