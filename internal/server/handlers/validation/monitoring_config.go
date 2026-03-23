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

package validation

import (
	"context"
	"errors"
	"fmt"

	"github.com/percona/everest-operator/utils"
	operatorUtils "github.com/percona/everest-operator/utils"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListMonitoringConfigs proxies the request to the next handler.
func (h *validateHandler) ListMonitoringConfigs(ctx context.Context, namespace string) (*monitoringv1alpha1.MonitoringConfigList, error) {
	return h.next.ListMonitoringConfigs(ctx, namespace)
}

// CreateMonitoringConfig proxies the request to the next handler.
func (h *validateHandler) CreateMonitoringConfig(ctx context.Context, namespace string, req *api.MonitoringConfigCreateParams) (*monitoringv1alpha1.MonitoringConfig, error) {
	if err := utils.ValidateEverestResourceName(req.Name, "name"); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	if ok := utils.ValidateURL(req.Url); !ok {
		return nil, errors.Join(ErrInvalidRequest, ErrInvalidURL("url"))
	}

	switch req.Type {
	case api.MonitoringConfigCreateParamsTypePmm:
		if req.Pmm == nil {
			return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("pmm key is required for type %s", req.Type))
		}

		if req.Pmm.ApiKey == "" && (req.Pmm.User == "" || req.Pmm.Password == "") {
			return nil, errors.Join(ErrInvalidRequest, errors.New("pmm.apiKey or pmm.user with pmm.password fields are required"))
		}

	default:
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("monitoring type %s is not supported", req.Type))
	}

	return h.next.CreateMonitoringConfig(ctx, namespace, req)
}

// DeleteMonitoringConfig proxies the request to the next handler.
func (h *validateHandler) DeleteMonitoringConfig(ctx context.Context, namespace, name string) error {
	// TODO: check if the monitoring config is used by any instance.

	return h.next.DeleteMonitoringConfig(ctx, namespace, name)
}

// GetMonitoringConfig proxies the request to the next handler.
func (h *validateHandler) GetMonitoringConfig(ctx context.Context, namespace, name string) (*monitoringv1alpha1.MonitoringConfig, error) {
	return h.next.GetMonitoringConfig(ctx, namespace, name)
}

// UpdateMonitoringConfig proxies the request to the next handler.
func (h *validateHandler) UpdateMonitoringConfig(ctx context.Context, namespace, name string, req *api.MonitoringConfigUpdateParams) (*monitoringv1alpha1.MonitoringConfig, error) {
	if req.Url != "" {
		if ok := operatorUtils.ValidateURL(req.Url); !ok {
			err := ErrInvalidURL("url")
			return nil, errors.Join(ErrInvalidRequest, err)
		}
	}

	switch req.Type {
	case "": // nothing to do.
	case api.MonitoringConfigUpdateParamsTypePmm:
		if req.Pmm == nil {
			return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("pmm key is required for type %s", req.Type))
		}
	default:
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("monitoring type %s is not supported", req.Type))
	}

	return h.next.UpdateMonitoringConfig(ctx, namespace, name, req)
}
