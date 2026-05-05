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
	if err := h.enforce(ctx, rbac.ResourceDataImportJobs,
		rbac.ActionRead, rbac.ObjectName(namespace, dbName),
	); errors.Is(err, ErrInsufficientPermissions) {
		list.Items = nil // No permissions, return empty list
	} else if err != nil {
		return nil, err
	}
	return list, nil
}
