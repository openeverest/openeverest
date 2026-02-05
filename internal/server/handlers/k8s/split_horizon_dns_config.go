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

package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	enginefeaturesv1alpha1 "github.com/percona/everest-operator/api/enginefeatures.everest/v1alpha1"
	"github.com/percona/everest/api"
)

func (h *k8sHandler) CreateSplitHorizonDNSConfig(ctx context.Context, shdc *enginefeaturesv1alpha1.SplitHorizonDNSConfig) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error) { //nolint:lll
	return h.kubeConnector.CreateSplitHorizonDNSConfig(ctx, shdc)
}

func (h *k8sHandler) UpdateSplitHorizonDNSConfig(ctx context.Context, namespace, name string, req *api.SplitHorizonDNSConfigUpdateParams) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error) { //nolint:lll
	shdc, err := h.kubeConnector.GetSplitHorizonDNSConfig(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return nil, err
	}

	if req.BaseDomainNameSuffix != nil {
		shdc.Spec.BaseDomainNameSuffix = *req.BaseDomainNameSuffix
	}

	if req.Certificate != nil {
		shdc.Spec.TLS.Certificate = &enginefeaturesv1alpha1.SplitHorizonDNSConfigTLSCertificateSpec{
			CACert: req.Certificate.CaCrt,
			CAKey:  req.Certificate.CaKey,
		}
	}

	return h.kubeConnector.UpdateSplitHorizonDNSConfig(ctx, shdc)
}

func (h *k8sHandler) ListSplitHorizonDNSConfigs(ctx context.Context, namespace string) (*enginefeaturesv1alpha1.SplitHorizonDNSConfigList, error) {
	return h.kubeConnector.ListSplitHorizonDNSConfigs(ctx, ctrlclient.InNamespace(namespace))
}

func (h *k8sHandler) DeleteSplitHorizonDNSConfig(ctx context.Context, namespace, name string) error {
	delObj := &enginefeaturesv1alpha1.SplitHorizonDNSConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	return h.kubeConnector.DeleteSplitHorizonDNSConfig(ctx, delObj)
}

func (h *k8sHandler) GetSplitHorizonDNSConfig(ctx context.Context, namespace, name string) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error) {
	return h.kubeConnector.GetSplitHorizonDNSConfig(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	})
}
