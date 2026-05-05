package k8s

import (
	"context"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListDataImportJobs returns a list of DataImportJobs for the specified database clusters.
func (h *k8sHandler) ListDataImportJobs(ctx context.Context, namespace, dbName string) (*everestv1alpha1.DataImportJobList, error) {
	result, err := h.kubeConnector.ListDataImportJobs(ctx, namespace, dbName)
	if err != nil {
		return nil, err
	}
	return result, nil
}
