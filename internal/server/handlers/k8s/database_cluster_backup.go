package k8s

import (
	"context"
	"errors"
	"slices"

	"github.com/AlekSi/pointer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/pkg/common"
)

func (h *k8sHandler) ListDatabaseClusterBackups(ctx context.Context, namespace, clusterName string) (*everestv1alpha1.DatabaseClusterBackupList, error) {
	return h.kubeConnector.ListDatabaseClusterBackups(ctx,
		ctrlclient.InNamespace(namespace),
		ctrlclient.MatchingLabels{common.DatabaseClusterNameLabel: clusterName},
	)
}

func (h *k8sHandler) CreateDatabaseClusterBackup(ctx context.Context, req *everestv1alpha1.DatabaseClusterBackup) (*everestv1alpha1.DatabaseClusterBackup, error) {
	if ok, err := h.ensureNoBackupsRunningForCluster(ctx, req.Spec.DBClusterName, req.GetNamespace()); err != nil {
		return nil, errors.Join(err, errors.New("could not check if backups are running"))
	} else if !ok {
		return nil, errors.New("backup is already running for the specified cluster")
	}
	return h.kubeConnector.CreateDatabaseClusterBackup(ctx, req)
}

func (h *k8sHandler) DeleteDatabaseClusterBackup(ctx context.Context, namespace, name string, req *api.DeleteDatabaseClusterBackupParams) error {
	cleanupStorage := pointer.Get(req.CleanupBackupStorage)
	backup, err := h.kubeConnector.GetDatabaseClusterBackup(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return errors.Join(err, errors.New("could not get Database Cluster Backup"))
	}

	if !cleanupStorage {
		if err := h.ensureBackupStorageProtection(ctx, backup); err != nil {
			return errors.Join(err, errors.New("could not ensure backup storage protection"))
		}
	}
	if err := h.ensureBackupForegroundDeletion(ctx, backup); err != nil {
		return errors.Join(err, errors.New("could not ensure backup foreground deletion"))
	}
	delObj := &everestv1alpha1.DatabaseClusterBackup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	return h.kubeConnector.DeleteDatabaseClusterBackup(ctx, delObj)
}

func (h *k8sHandler) GetDatabaseClusterBackup(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseClusterBackup, error) {
	return h.kubeConnector.GetDatabaseClusterBackup(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// Returns `true` if no backups are running for the specified cluster.
func (h *k8sHandler) ensureNoBackupsRunningForCluster(ctx context.Context, dbClusterName, namespace string) (bool, error) {
	backupList, err := h.kubeConnector.ListDatabaseClusterBackups(ctx,
		ctrlclient.InNamespace(namespace),
		ctrlclient.MatchingLabels{common.DatabaseClusterNameLabel: dbClusterName},
	)
	if err != nil {
		return false, errors.Join(err, errors.New("could not list Database Cluster Backups"))
	}
	return !slices.ContainsFunc(backupList.Items, func(b everestv1alpha1.DatabaseClusterBackup) bool {
		return (b.Status.State == everestv1alpha1.BackupRunning ||
			b.Status.State == everestv1alpha1.BackupStarting ||
			b.Status.State == everestv1alpha1.BackupNew) &&
			b.DeletionTimestamp.IsZero()
	}), nil
}
