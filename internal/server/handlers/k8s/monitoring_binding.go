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

	"github.com/AlekSi/pointer"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListMonitoringBindings returns the monitoring bindings in the namespace,
// only those to the given destination when one is set.
func (h *k8sHandler) ListMonitoringBindings(ctx context.Context, cluster, namespace string, params *api.ListMonitoringBindingsParams) (*monitoringv1alpha1.MonitoringBindingList, error) {
	kubeConnector, err := h.connector(ctx, cluster)
	if err != nil {
		return nil, err
	}
	opts := []ctrlclient.ListOption{ctrlclient.InNamespace(namespace)}
	if params != nil {
		if destination := pointer.Get(params.Destination); destination != "" {
			opts = append(opts, ctrlclient.MatchingLabels{monitoringv1alpha1.BindingDestinationLabel: destination})
		}
	}
	return kubeConnector.ListMonitoringBindings(ctx, opts...)
}

// GetMonitoringBinding returns a monitoring binding by name and namespace.
func (h *k8sHandler) GetMonitoringBinding(ctx context.Context, cluster, namespace, name string) (*monitoringv1alpha1.MonitoringBinding, error) {
	kubeConnector, err := h.connector(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return kubeConnector.GetMonitoringBinding(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}
