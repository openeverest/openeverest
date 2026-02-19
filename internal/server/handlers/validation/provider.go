// Package validation provides the validation handler.
package validation

import (
	"context"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// ListProviders proxies the request to the next handler.
func (h *validateHandler) ListProviders(ctx context.Context) (*corev1alpha1.ProviderList, error) {
	return h.next.ListProviders(ctx)
}

// GetProvider proxies the request to the next handler.
func (h *validateHandler) GetProvider(ctx context.Context, name string) (*corev1alpha1.Provider, error) {
	return h.next.GetProvider(ctx, name)
}
