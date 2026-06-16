// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
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

// PatchSecret patches a secret using the provided patch.
func (k *Kubernetes) PatchSecret(ctx context.Context, secret *corev1.Secret, patch ctrlclient.Patch) (*corev1.Secret, error) {
	if err := k.k8sClient.Patch(ctx, secret, patch); err != nil {
		return nil, err
	}
	return secret, nil
}

// DeleteSecret deletes a secret that matches the criteria.
func (k *Kubernetes) DeleteSecret(ctx context.Context, obj *corev1.Secret) error {
	return k.k8sClient.Delete(ctx, obj)
}
