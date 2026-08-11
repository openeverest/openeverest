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

package kubernetes

import (
	"context"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetSubscription returns OLM subscription that matches the criteria.
func (k *Kubernetes) GetSubscription(ctx context.Context, key ctrlclient.ObjectKey) (*olmv1alpha1.Subscription, error) {
	result := &olmv1alpha1.Subscription{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListSubscriptions lists OLM subscriptions that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListSubscriptions(ctx context.Context, opts ...ctrlclient.ListOption) (*olmv1alpha1.SubscriptionList, error) {
	result := &olmv1alpha1.SubscriptionList{}
	err := listPaginated(ctx, k.k8sClient, result,
		func() *olmv1alpha1.SubscriptionList { return &olmv1alpha1.SubscriptionList{} },
		func(res, page *olmv1alpha1.SubscriptionList) { res.Items = append(res.Items, page.Items...) },
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteSubscription deletes OLM subscription that matches the criteria.
func (k *Kubernetes) DeleteSubscription(ctx context.Context, obj *olmv1alpha1.Subscription) error {
	return k.k8sClient.Delete(ctx, obj)
}
