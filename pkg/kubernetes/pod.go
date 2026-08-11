package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListPods returns list of pods that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListPods(ctx context.Context, opts ...ctrlclient.ListOption) (*corev1.PodList, error) {
	result := &corev1.PodList{}
	err := listPaginated(ctx, k.k8sClient, result,
		func() *corev1.PodList { return &corev1.PodList{} },
		func(res, page *corev1.PodList) { res.Items = append(res.Items, page.Items...) },
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}
