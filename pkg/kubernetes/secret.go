package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListSecrets returns list of secrets that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListSecrets(ctx context.Context, opts ...ctrlclient.ListOption) (*corev1.SecretList, error) {
	result := &corev1.SecretList{}
	err := listPaginated(ctx, k.k8sClient, result,
		func() *corev1.SecretList { return &corev1.SecretList{} },
		func(res, page *corev1.SecretList) { res.Items = append(res.Items, page.Items...) },
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetSecret returns a secret that matches the criteria.
func (k *Kubernetes) GetSecret(ctx context.Context, key ctrlclient.ObjectKey) (*corev1.Secret, error) {
	result := &corev1.Secret{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateSecret creates a secret.
func (k *Kubernetes) CreateSecret(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error) {
	if err := k.k8sClient.Create(ctx, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

// UpdateSecret updates a secret.
func (k *Kubernetes) UpdateSecret(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error) {
	if err := k.k8sClient.Update(ctx, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

// DeleteSecret deletes a secret that matches the criteria.
func (k *Kubernetes) DeleteSecret(ctx context.Context, obj *corev1.Secret) error {
	return k.k8sClient.Delete(ctx, obj)
}
