package k8s

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/common"
)

func (h *k8sHandler) ListDatabaseClusterRestores(ctx context.Context, namespace, clusterName string) (*everestv1alpha1.DatabaseClusterRestoreList, error) {
	return h.kubeConnector.ListDatabaseClusterRestores(ctx,
		ctrlclient.InNamespace(namespace),
		ctrlclient.MatchingLabels{common.DatabaseClusterNameLabel: clusterName},
	)
}

func (h *k8sHandler) CreateDatabaseClusterRestore(ctx context.Context, req *everestv1alpha1.DatabaseClusterRestore) (*everestv1alpha1.DatabaseClusterRestore, error) {
	dbCluster, err := h.kubeConnector.GetDatabaseCluster(ctx, types.NamespacedName{Namespace: req.GetNamespace(), Name: req.Spec.DBClusterName})
	if err != nil {
		return nil, fmt.Errorf("failed to GetDatabaseCluster: %w", err)
	}
	if dbCluster.Status.Status == everestv1alpha1.AppStateRestoring {
		return nil, errors.New("another restore is already in progress")
	}
	return h.kubeConnector.CreateDatabaseClusterRestore(ctx, req)
}

func (h *k8sHandler) DeleteDatabaseClusterRestore(ctx context.Context, namespace, name string) error {
	delObj := &everestv1alpha1.DatabaseClusterRestore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	return h.kubeConnector.DeleteDatabaseClusterRestore(ctx, delObj)
}

func (h *k8sHandler) GetDatabaseClusterRestore(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseClusterRestore, error) {
	return h.kubeConnector.GetDatabaseClusterRestore(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

func (h *k8sHandler) UpdateDatabaseClusterRestore(ctx context.Context, req *everestv1alpha1.DatabaseClusterRestore) (*everestv1alpha1.DatabaseClusterRestore, error) {
	return h.kubeConnector.UpdateDatabaseClusterRestore(ctx, req)
}
