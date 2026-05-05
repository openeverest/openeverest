package rbac

import (
	"context"
	"errors"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/rbac"
)

// ListDataImporters returns a list of DataImporters that support the specified engines.
func (h *rbacHandler) ListDataImporters(ctx context.Context, supportedEngines ...string) (*everestv1alpha1.DataImporterList, error) {
	result, err := h.next.ListDataImporters(ctx, supportedEngines...)
	if err != nil {
		return nil, err
	}
	filtered := make([]everestv1alpha1.DataImporter, 0, len(result.Items))
	for _, di := range result.Items {
		if err := h.enforce(ctx, rbac.ResourceDataImporters, rbac.ActionRead, rbac.ObjectName(di.GetName())); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, err
		}
		filtered = append(filtered, di)
	}
	result.Items = filtered
	return result, nil
}
