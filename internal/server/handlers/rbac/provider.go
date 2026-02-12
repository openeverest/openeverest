// Package rbac provides the RBAC handler.
package rbac

import (
	"context"

	"github.com/openeverest/openeverest/v2/pkg/apis/v1alpha1"
)

// ListProviders proxies the request to the next handler.
func (h *rbacHandler) ListProviders(ctx context.Context) (*v1alpha1.ProviderList, error) {
	// Add RBAC checks here if needed in the future
	return h.next.ListProviders(ctx)
}

// GetProvider proxies the request to the next handler.
func (h *rbacHandler) GetProvider(ctx context.Context, name string) (*v1alpha1.Provider, error) {
	// Add RBAC checks here if needed in the future
	return h.next.GetProvider(ctx, name)
}
