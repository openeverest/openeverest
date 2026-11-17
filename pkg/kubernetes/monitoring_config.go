// everest
// Copyright (C) 2023 Percona LLC
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

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// ListLegacyMonitoringConfigs returns the metadata of every MonitoringConfig in the cluster.
// The list is empty when the cluster no longer serves the MonitoringConfig kind.
func (k *Kubernetes) ListLegacyMonitoringConfigs(ctx context.Context) (*metav1.PartialObjectMetadataList, error) {
	result := &metav1.PartialObjectMetadataList{}
	gvk := monitoringv1alpha1.GroupVersion.WithKind("MonitoringConfig")
	if _, err := k.k8sClient.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version); err != nil {
		if meta.IsNoMatchError(err) || k8serrors.IsNotFound(err) {
			return result, nil
		}
		return nil, err
	}
	result.SetGroupVersionKind(monitoringv1alpha1.GroupVersion.WithKind("MonitoringConfigList"))
	if err := k.k8sClient.List(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}
