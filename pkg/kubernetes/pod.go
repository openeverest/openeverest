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

	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListPods returns list of pods that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListPods(ctx context.Context, opts ...ctrlclient.ListOption) (*corev1.PodList, error) {
	result := &corev1.PodList{}
	err := listPaginated(ctx, k.k8sClient, result,
		func() *corev1.PodList { return &corev1.PodList{} },
		func(res, page *corev1.PodList) { res.Items = append(res.Items, page.Items...) },
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}
