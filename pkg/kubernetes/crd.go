// everest
// Copyright (C) 2025 Percona LLC
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

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListCRDs lists all CRDs that match the criteria.
func (k *Kubernetes) ListCRDs(ctx context.Context, opts ...ctrlclient.ListOption) (*apiextv1.CustomResourceDefinitionList, error) {
	result := &apiextv1.CustomResourceDefinitionList{}
	//nolint:gofumpt
	err := listPaginated(ctx, k.k8sClient, result,
		func() *apiextv1.CustomResourceDefinitionList {
			return &apiextv1.CustomResourceDefinitionList{}
		},
		func(res, page *apiextv1.CustomResourceDefinitionList) {
			res.Items = append(res.Items, page.Items...)
		},
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteCRD deletes a CRD that matches the criteria.
func (k *Kubernetes) DeleteCRD(ctx context.Context, obj *apiextv1.CustomResourceDefinition) error {
	return k.k8sClient.Delete(ctx, obj)
}
