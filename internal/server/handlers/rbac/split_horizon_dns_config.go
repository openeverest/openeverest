// everest
// Copyright (C) 2025 Percona LLC
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

package rbac

import (
	"context"
	"fmt"
	"slices"

	"github.com/percona/everest-operator/api/enginefeatures.everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/pkg/rbac"
)

func (h *rbacHandler) CreateSplitHorizonDNSConfig(ctx context.Context, shdc *v1alpha1.SplitHorizonDNSConfig) (*v1alpha1.SplitHorizonDNSConfig, error) { //nolint:lll
	if err := h.enforce(ctx, rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, rbac.ActionCreate, rbac.ObjectName(shdc.GetNamespace(), shdc.GetName())); err != nil {
		return nil, err
	}
	return h.next.CreateSplitHorizonDNSConfig(ctx, shdc)
}

func (h *rbacHandler) UpdateSplitHorizonDNSConfig(ctx context.Context, namespace, name string, req *api.SplitHorizonDNSConfigUpdateParams) (*v1alpha1.SplitHorizonDNSConfig, error) { //nolint:lll
	if err := h.enforce(ctx, rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, rbac.ActionUpdate, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	return h.next.UpdateSplitHorizonDNSConfig(ctx, namespace, name, req)
}

func (h *rbacHandler) ListSplitHorizonDNSConfigs(ctx context.Context, namespace string) (*v1alpha1.SplitHorizonDNSConfigList, error) {
	shdcList, err := h.next.ListSplitHorizonDNSConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("ListSplitHorizonDNSConfigs failed: %w", err)
	}
	// filter out ListSplitHorizonDNSConfigs that the user does not have access to.
	shdcList.Items = slices.DeleteFunc(shdcList.Items, func(shdc v1alpha1.SplitHorizonDNSConfig) bool {
		return h.enforce(ctx, rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, rbac.ActionRead, rbac.ObjectName(shdc.GetNamespace(), shdc.GetName())) != nil
	})
	return shdcList, nil
}

func (h *rbacHandler) DeleteSplitHorizonDNSConfig(ctx context.Context, namespace, name string) error {
	if err := h.enforce(ctx, rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, rbac.ActionDelete, rbac.ObjectName(namespace, name)); err != nil {
		return err
	}
	return h.next.DeleteSplitHorizonDNSConfig(ctx, namespace, name)
}

func (h *rbacHandler) GetSplitHorizonDNSConfig(ctx context.Context, namespace, name string) (*v1alpha1.SplitHorizonDNSConfig, error) {
	if err := h.enforce(ctx, rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, rbac.ActionRead, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	return h.next.GetSplitHorizonDNSConfig(ctx, namespace, name)
}
