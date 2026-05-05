package k8s

import (
	"context"
	"slices"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListDataImporters returns a list of DataImporters that support the specified engines.
func (h *k8sHandler) ListDataImporters(ctx context.Context, supportedEngines ...string) (*everestv1alpha1.DataImporterList, error) {
	list, err := h.kubeConnector.ListDataImporters(ctx, &ctrlclient.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(supportedEngines) == 0 {
		return list, nil
	}

	// filter the list based on the engines.
	result := &everestv1alpha1.DataImporterList{}
	for _, wantEngine := range supportedEngines {
		for _, importer := range list.Items {
			if slices.Contains(importer.Spec.SupportedEngines, everestv1alpha1.EngineType(wantEngine)) {
				result.Items = append(result.Items, importer)
			}
		}
	}
	result.Items = slices.CompactFunc(result.Items, func(a, b everestv1alpha1.DataImporter) bool {
		return a.GetName() == b.GetName()
	})
	return result, nil
}
