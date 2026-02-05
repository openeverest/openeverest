package validation

import (
	"context"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListDataImporters returns a list of DataImporters that support the specified engines.
func (h *validateHandler) ListDataImporters(ctx context.Context, supportedEngines ...string) (*everestv1alpha1.DataImporterList, error) {
	return h.next.ListDataImporters(ctx, supportedEngines...)
}
