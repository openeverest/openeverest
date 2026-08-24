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

// Package validation provides the validation handler.
package validation

import (
	"context"
	"errors"
	"fmt"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListInstances proxies the request to the next handler.
func (h *validateHandler) ListInstances(ctx context.Context, cluster, namespace string) (*corev1alpha1.InstanceList, error) {
	return h.next.ListInstances(ctx, cluster, namespace)
}

// GetInstance proxies the request to the next handler.
func (h *validateHandler) GetInstance(ctx context.Context, cluster, namespace, name string) (*corev1alpha1.Instance, error) {
	return h.next.GetInstance(ctx, cluster, namespace, name)
}

// CreateInstance proxies the request to the next handler.
func (h *validateHandler) CreateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	if err := validateBackupScheduleNames(instance); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	return h.next.CreateInstance(ctx, cluster, instance)
}

// UpdateInstance proxies the request to the next handler.
func (h *validateHandler) UpdateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	if err := validateBackupScheduleNames(instance); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	return h.next.UpdateInstance(ctx, cluster, instance)
}

// validateBackupScheduleNames rejects Instances whose backup schedule names
// are not unique across all storages. Schedule names double as the schedule
// key on the engine, so a duplicate silently overwrites (drops) one of the
// schedules on reconcile.
func validateBackupScheduleNames(instance *corev1alpha1.Instance) error {
	if instance == nil || instance.Spec.Backup == nil {
		return nil
	}
	seen := make(map[string]struct{})
	for _, storage := range instance.Spec.Backup.Storages {
		for _, schedule := range storage.Schedules {
			if _, ok := seen[schedule.Name]; ok {
				return fmt.Errorf("%w: schedule name '%s' is used more than once", errDuplicatedSchedules, schedule.Name)
			}
			seen[schedule.Name] = struct{}{}
		}
	}
	return nil
}

// DeleteInstance proxies the request to the next handler.
func (h *validateHandler) DeleteInstance(ctx context.Context, cluster, namespace, name string, params *api.DeleteInstanceParams) error {
	return h.next.DeleteInstance(ctx, cluster, namespace, name, params)
}

// GetInstanceConnection proxies the request to the next handler.
func (h *validateHandler) GetInstanceConnection(ctx context.Context, cluster, namespace, name string) (*api.InstanceConnectionDetails, error) {
	return h.next.GetInstanceConnection(ctx, cluster, namespace, name)
}
