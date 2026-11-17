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
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListMonitoringBindings rejects a destination filter that is not a valid
// label value, since it becomes a label selector, then proxies to the next handler.
func (h *validateHandler) ListMonitoringBindings(ctx context.Context, cluster, namespace string, params *api.ListMonitoringBindingsParams) (*monitoringv1alpha1.MonitoringBindingList, error) {
	if params != nil && params.Destination != nil {
		if errs := validation.IsValidLabelValue(*params.Destination); len(errs) > 0 {
			return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("invalid destination %q: %s", *params.Destination, strings.Join(errs, "; ")))
		}
	}
	return h.next.ListMonitoringBindings(ctx, cluster, namespace, params)
}

// GetMonitoringBinding proxies the request to the next handler.
func (h *validateHandler) GetMonitoringBinding(ctx context.Context, cluster, namespace, name string) (*monitoringv1alpha1.MonitoringBinding, error) {
	return h.next.GetMonitoringBinding(ctx, cluster, namespace, name)
}
