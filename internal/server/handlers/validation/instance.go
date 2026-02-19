// Package validation provides the validation handler.
package validation

import (
	"context"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// ListInstances proxies the request to the next handler.
func (h *validateHandler) ListInstances(ctx context.Context, namespace string) (*corev1alpha1.InstanceList, error) {
	return h.next.ListInstances(ctx, namespace)
}

// GetInstance proxies the request to the next handler.
func (h *validateHandler) GetInstance(ctx context.Context, namespace, name string) (*corev1alpha1.Instance, error) {
	return h.next.GetInstance(ctx, namespace, name)
}

// CreateInstance proxies the request to the next handler.
func (h *validateHandler) CreateInstance(ctx context.Context, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	// Add validation here if needed in the future
	return h.next.CreateInstance(ctx, instance)
}

// UpdateInstance proxies the request to the next handler.
func (h *validateHandler) UpdateInstance(ctx context.Context, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	// Add validation here if needed in the future
	return h.next.UpdateInstance(ctx, instance)
}

// DeleteInstance proxies the request to the next handler.
func (h *validateHandler) DeleteInstance(ctx context.Context, namespace, name string) error {
	return h.next.DeleteInstance(ctx, namespace, name)
}
