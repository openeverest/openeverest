// everest
// Copyright (C) 2023 Percona LLC
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

// Package kubernetes ...
package kubernetes

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/apis/v1alpha1"
)

// ListProviders returns list of providers that match the criteria.
func (k *Kubernetes) ListProviders(ctx context.Context, opts ...ctrlclient.ListOption) (*v1alpha1.ProviderList, error) {
	result := &v1alpha1.ProviderList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetProvider returns provider that matches the criteria.
func (k *Kubernetes) GetProvider(ctx context.Context, key ctrlclient.ObjectKey) (*v1alpha1.Provider, error) {
	result := &v1alpha1.Provider{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}
