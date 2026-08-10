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

package rbac

import (
	"context"
	"errors"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/rbac"
)

// ListDataImportJobs returns a list of DataImportJobs for the specified database clusters.
func (h *rbacHandler) ListDataImportJobs(ctx context.Context, namespace, dbName string) (*everestv1alpha1.DataImportJobList, error) {
	list, err := h.next.ListDataImportJobs(ctx, namespace, dbName)
	if err != nil {
		return nil, err
	}
	if err := h.enforce(
		ctx, rbac.ResourceDataImportJobs,
		rbac.ActionRead, rbac.ObjectName(namespace, dbName),
	); errors.Is(err, ErrInsufficientPermissions) {
		list.Items = nil // No permissions, return empty list
	} else if err != nil {
		return nil, err
	}
	return list, nil
}
