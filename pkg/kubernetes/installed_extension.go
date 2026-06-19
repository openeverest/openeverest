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

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

// ListInstalledExtensions returns InstalledExtension records that match the criteria.
func (k *Kubernetes) ListInstalledExtensions(ctx context.Context, opts ...ctrlclient.ListOption) (*extensionsv1alpha1.InstalledExtensionList, error) {
	result := &extensionsv1alpha1.InstalledExtensionList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetInstalledExtension returns the InstalledExtension that matches the criteria.
func (k *Kubernetes) GetInstalledExtension(ctx context.Context, key ctrlclient.ObjectKey) (*extensionsv1alpha1.InstalledExtension, error) {
	result := &extensionsv1alpha1.InstalledExtension{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateInstalledExtension creates a new InstalledExtension.
func (k *Kubernetes) CreateInstalledExtension(ctx context.Context, ie *extensionsv1alpha1.InstalledExtension) (*extensionsv1alpha1.InstalledExtension, error) {
	if err := k.k8sClient.Create(ctx, ie); err != nil {
		return nil, err
	}
	return ie, nil
}

// UpdateInstalledExtension updates an existing InstalledExtension.
func (k *Kubernetes) UpdateInstalledExtension(ctx context.Context, ie *extensionsv1alpha1.InstalledExtension) (*extensionsv1alpha1.InstalledExtension, error) {
	if err := k.k8sClient.Update(ctx, ie); err != nil {
		return nil, err
	}
	return ie, nil
}

// DeleteInstalledExtension deletes an InstalledExtension.
func (k *Kubernetes) DeleteInstalledExtension(ctx context.Context, obj *extensionsv1alpha1.InstalledExtension) error {
	return k.k8sClient.Delete(ctx, obj)
}
