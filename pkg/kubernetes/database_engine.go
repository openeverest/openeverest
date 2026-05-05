// everest
// Copyright (C) 2023 Percona LLC
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

// Package kubernetes ...
package kubernetes

import (
	"context"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListDatabaseEngines returns list of managed database engines that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListDatabaseEngines(ctx context.Context, opts ...ctrlclient.ListOption) (*everestv1alpha1.DatabaseEngineList, error) {
	result := &everestv1alpha1.DatabaseEngineList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetDatabaseEngine returns database engine that matches the criteria.
func (k *Kubernetes) GetDatabaseEngine(ctx context.Context, key ctrlclient.ObjectKey) (*everestv1alpha1.DatabaseEngine, error) {
	result := &everestv1alpha1.DatabaseEngine{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateDatabaseEngine updates the provided database engine.
func (k *Kubernetes) UpdateDatabaseEngine(ctx context.Context, engine *everestv1alpha1.DatabaseEngine) (*everestv1alpha1.DatabaseEngine, error) {
	if err := k.k8sClient.Update(ctx, engine); err != nil {
		return nil, err
	}
	return engine, nil
}

// SetDatabaseEngineLock sets the lock on the database engine that matches the criteria.
// The lock is automatically set to false once everest-operator completes its upgrade.
func (k *Kubernetes) SetDatabaseEngineLock(ctx context.Context, key ctrlclient.ObjectKey, locked bool) error {
	// We wrap this logic into a retry block to reduce the chances of conflicts.
	var b backoff.BackOff
	b = backoff.NewConstantBackOff(backoffInterval)
	b = backoff.WithMaxRetries(b, backoffMaxRetries)
	b = backoff.WithContext(b, ctx)
	return backoff.Retry(func() error {
		engine, err := k.GetDatabaseEngine(ctx, key)
		if err != nil {
			return err
		}
		annotations := engine.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[everestv1alpha1.DatabaseOperatorUpgradeLockAnnotation] = time.Now().Format(time.RFC3339)
		if !locked {
			delete(annotations, everestv1alpha1.DatabaseOperatorUpgradeLockAnnotation)
		}
		engine.SetAnnotations(annotations)
		_, err = k.UpdateDatabaseEngine(ctx, engine)
		return err
	},
		b,
	)
}
