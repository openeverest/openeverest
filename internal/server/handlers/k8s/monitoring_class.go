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

package k8s

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// ListMonitoringClasses returns list of monitoring classes.
func (h *k8sHandler) ListMonitoringClasses(ctx context.Context, _ string) (*monitoringv1alpha1.MonitoringClassList, error) {
	return h.kubeConnector.ListMonitoringClasses(ctx)
}

// GetMonitoringClass returns monitoring class that matches the criteria.
func (h *k8sHandler) GetMonitoringClass(ctx context.Context, _, name string) (*monitoringv1alpha1.MonitoringClass, error) {
	return h.kubeConnector.GetMonitoringClass(ctx, types.NamespacedName{Name: name})
}
