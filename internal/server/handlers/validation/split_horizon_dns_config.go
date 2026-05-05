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

package validation

import (
	"context"
	"errors"

	enginefeaturesv1alpha1 "github.com/percona/everest-operator/api/enginefeatures.everest/v1alpha1"
	"github.com/percona/everest/api"
)

func (h *validateHandler) CreateSplitHorizonDNSConfig(ctx context.Context, shdc *enginefeaturesv1alpha1.SplitHorizonDNSConfig) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error) { //nolint:lll
	// The validation is performed in the webhook, so we just bypass the call to the next handler.
	return h.next.CreateSplitHorizonDNSConfig(ctx, shdc)
}

func (h *validateHandler) UpdateSplitHorizonDNSConfig(ctx context.Context, namespace, name string, req *api.SplitHorizonDNSConfigUpdateParams) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error) { //nolint:lll
	if req.BaseDomainNameSuffix == nil && req.Certificate == nil {
		return nil, errors.New("at least one field must be provided to update")
	}

	if req.Certificate != nil {
		var allErrs []error
		if req.Certificate.CaCrt == "" {
			allErrs = append(allErrs, errors.New("certificate.ca.crt can not be empty"))
		}

		if req.Certificate.CaKey == "" {
			allErrs = append(allErrs, errors.New("certificate.ca.key can not be empty"))
		}

		if len(allErrs) > 0 {
			return nil, errors.Join(allErrs...)
		}
	}
	return h.next.UpdateSplitHorizonDNSConfig(ctx, namespace, name, req)
}

func (h *validateHandler) ListSplitHorizonDNSConfigs(ctx context.Context, namespace string) (*enginefeaturesv1alpha1.SplitHorizonDNSConfigList, error) {
	return h.next.ListSplitHorizonDNSConfigs(ctx, namespace)
}

func (h *validateHandler) DeleteSplitHorizonDNSConfig(ctx context.Context, namespace, name string) error {
	return h.next.DeleteSplitHorizonDNSConfig(ctx, namespace, name)
}

func (h *validateHandler) GetSplitHorizonDNSConfig(ctx context.Context, namespace, name string) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error) {
	return h.next.GetSplitHorizonDNSConfig(ctx, namespace, name)
}
