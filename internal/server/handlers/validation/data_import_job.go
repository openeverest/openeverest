package validation

import (
	"context"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListDataImportJobs returns a list of DataImportJobs for the specified database clusters.
func (h *validateHandler) ListDataImportJobs(ctx context.Context, namespace, dbName string) (*everestv1alpha1.DataImportJobList, error) {
	return h.next.ListDataImportJobs(ctx, namespace, dbName)
}
