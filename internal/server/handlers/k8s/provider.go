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

package k8s

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	"github.com/openeverest/openeverest/v2/pkg/apis/v1alpha1"
)

// ListProviders returns list of providers.
func (h *k8sHandler) ListProviders(ctx context.Context) (*v1alpha1.ProviderList, error) {
	return h.kubeConnector.ListProviders(ctx)
}

// GetProvider returns provider that matches the criteria.
func (h *k8sHandler) GetProvider(ctx context.Context, name string) (*v1alpha1.Provider, error) {
	// Providers are cluster-scoped, so no namespace
	return h.kubeConnector.GetProvider(ctx, types.NamespacedName{Name: name})
}
