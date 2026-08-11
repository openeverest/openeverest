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
	"errors"
	"fmt"

	goversion "github.com/hashicorp/go-version"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrOperatorNotInstalled is returned when an operator is not installed.
var ErrOperatorNotInstalled = fmt.Errorf("operatorNotInstalled")

// GetInstalledOperatorVersion returns the version of installed operator that matches the criteria.
func (k *Kubernetes) GetInstalledOperatorVersion(ctx context.Context, key ctrlclient.ObjectKey) (*goversion.Version, error) {
	sub := &olmv1alpha1.Subscription{}
	if err := k.k8sClient.Get(ctx, key, sub); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.Join(ErrOperatorNotInstalled, errors.New("could not retrieve subscription"))
		}
		return nil, errors.Join(err, errors.New("could not retrieve subscription"))
	}

	if sub.Status.InstalledCSV == "" {
		return nil, ErrOperatorNotInstalled
	}

	csv, err := k.GetClusterServiceVersion(ctx, types.NamespacedName{
		Name:      sub.Status.InstalledCSV,
		Namespace: sub.GetNamespace(),
	})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not retrieve cluster service version"))
	}

	return goversion.NewVersion(csv.Spec.Version.FinalizeVersion())
}

// ListInstalledOperators returns the list of installed operators that match the criteria.
func (k *Kubernetes) ListInstalledOperators(ctx context.Context, opts ...ctrlclient.ListOption) (*olmv1alpha1.SubscriptionList, error) {
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
