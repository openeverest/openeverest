// Package validation provides the validation handler.
package validation

import (
	"context"

	"github.com/openeverest/openeverest/v2/pkg/apis/v1alpha1"
)

// ListProviders proxies the request to the next handler.
func (h *validateHandler) ListProviders(ctx context.Context) (*v1alpha1.ProviderList, error) {
	return h.next.ListProviders(ctx)
}

// GetProvider proxies the request to the next handler.
func (h *validateHandler) GetProvider(ctx context.Context, name string) (*v1alpha1.Provider, error) {
	return h.next.GetProvider(ctx, name)
}
