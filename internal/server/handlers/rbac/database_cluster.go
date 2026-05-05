package rbac

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/AlekSi/pointer"
	corev1 "k8s.io/api/core/v1"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/rbac"
)

func (h *rbacHandler) CreateDatabaseCluster(ctx context.Context, db *everestv1alpha1.DatabaseCluster) (*everestv1alpha1.DatabaseCluster, error) {
	name := db.GetName()
	namespace := db.GetNamespace()
	object := rbac.ObjectName(namespace, name)
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusters, rbac.ActionCreate, object); err != nil {
		return nil, err
	}

	engineName := common.OperatorTypeToName[db.Spec.Engine.Type]
	if err := h.enforce(ctx, rbac.ResourceDatabaseEngines, rbac.ActionRead, rbac.ObjectName(namespace, engineName)); err != nil {
		return nil, err
	}

	schedules := db.Spec.Backup.Schedules
	if len(schedules) > 0 {
		// To create a cluster with backup schedules, the user needs to explicitly have permissions to take backups for this cluster.
		if err := h.enforce(ctx, rbac.ResourceDatabaseClusterBackups, rbac.ActionCreate,
			rbac.ObjectName(namespace, db.GetName()),
		); err != nil {
			return nil, err
		}
		// User should be able to read a backup storage to use it in a backup schedule.
		for _, sched := range schedules {
			if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead,
				rbac.ObjectName(namespace, sched.BackupStorageName),
			); err != nil {
				return nil, err
			}
		}
	}

	// Check permissions for creating a cluster from a backup.
	if dataSrc := db.Spec.DataSource; dataSrc != nil && dataSrc.DBClusterBackupName != "" {
		sourceBackup := dataSrc.DBClusterBackupName
		if err := h.enforce(ctx, rbac.ResourceDatabaseClusterRestores,
			rbac.ActionCreate, rbac.ObjectName(namespace, db.GetName()),
		); err != nil {
			return nil, err
		}
		// Get the name of the source database cluster.
		bkp, err := h.next.GetDatabaseClusterBackup(ctx, namespace, sourceBackup)
		if err != nil {
			return nil, errors.Join(err, errors.New("failed to get database cluster backup"))
		}
		sourceDB := bkp.Spec.DBClusterName

		if err := h.enforceDBRestore(ctx, namespace, sourceDB); err != nil {
			return nil, err
		}
	}

	// Check permissions for engine features used in the database cluster.
	if err := h.enforceEngineFeaturesRead(ctx, db); err != nil {
		return nil, err
	}
	return h.next.CreateDatabaseCluster(ctx, db)
}

func (h *rbacHandler) ListDatabaseClusters(ctx context.Context, namespace string) (*everestv1alpha1.DatabaseClusterList, error) {
	clusterList, err := h.next.ListDatabaseClusters(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("ListDatabaseClusters failed: %w", err)
	}

	result := make([]everestv1alpha1.DatabaseCluster, 0, len(clusterList.Items))
	for _, db := range clusterList.Items {
		if err := h.enforceDBClusterRead(ctx, &db); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		result = append(result, db)
	}
	clusterList.Items = result
	return clusterList, nil
}

func (h *rbacHandler) DeleteDatabaseCluster(ctx context.Context, namespace, name string, req *api.DeleteDatabaseClusterParams) error {
	db, err := h.next.GetDatabaseCluster(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("GetDatabaseCluster failed: %w", err)
	}
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusters, rbac.ActionDelete, rbac.ObjectName(namespace, name)); err != nil {
		return err
	}
	engineName := common.OperatorTypeToName[db.Spec.Engine.Type]
	if err := h.enforce(ctx, rbac.ResourceDatabaseEngines, rbac.ActionRead, rbac.ObjectName(namespace, engineName)); err != nil {
		return err
	}
	return h.next.DeleteDatabaseCluster(ctx, namespace, name, req)
}

func (h *rbacHandler) UpdateDatabaseCluster(ctx context.Context, db *everestv1alpha1.DatabaseCluster) (*everestv1alpha1.DatabaseCluster, error) {
	name := db.GetName()
	namespace := db.GetNamespace()
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusters, rbac.ActionUpdate, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	engineName := common.OperatorTypeToName[db.Spec.Engine.Type]
	if err := h.enforce(ctx, rbac.ResourceDatabaseEngines, rbac.ActionRead, rbac.ObjectName(namespace, engineName)); err != nil {
		return nil, err
	}

	oldDB, err := h.next.GetDatabaseCluster(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	oldSched := oldDB.Spec.Backup.Schedules
	updatedSched := db.Spec.Backup.Schedules

	sortFn := func(a, b everestv1alpha1.BackupSchedule) int { return strings.Compare(a.Name, b.Name) }
	slices.SortFunc(oldSched, sortFn)
	slices.SortFunc(updatedSched, sortFn)

	isSchedEqual := func() bool {
		if len(oldSched) != len(updatedSched) {
			return false
		}
		for i := range oldSched {
			if oldSched[i].Name != updatedSched[i].Name ||
				oldSched[i].Enabled != updatedSched[i].Enabled ||
				oldSched[i].BackupStorageName != updatedSched[i].BackupStorageName ||
				oldSched[i].Schedule != updatedSched[i].Schedule ||
				oldSched[i].RetentionCopies != updatedSched[i].RetentionCopies {
				return false
			}
		}
		return true
	}

	// If shedules are updated, user should have permissions to create a backup for this cluster.
	if !isSchedEqual() {
		if err := h.enforce(ctx, rbac.ResourceDatabaseClusterBackups, rbac.ActionCreate, rbac.ObjectName(oldDB.GetNamespace(), db.GetName())); err != nil {
			return nil, err
		}
	}

	// User should be able to read a backup storage to use it in a backup schedule.
	for _, sched := range updatedSched {
		if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead,
			rbac.ObjectName(oldDB.GetNamespace(), sched.BackupStorageName),
		); err != nil {
			return nil, err
		}
	}

	// Check permissions for engine features used in the database cluster.
	if err := h.enforceEngineFeaturesRead(ctx, db); err != nil {
		return nil, err
	}
	return h.next.UpdateDatabaseCluster(ctx, db)
}

func (h *rbacHandler) GetDatabaseCluster(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseCluster, error) {
	result, err := h.next.GetDatabaseCluster(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	if err := h.enforceDBClusterRead(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *rbacHandler) GetDatabaseClusterCredentials(ctx context.Context, namespace, name string) (*api.DatabaseClusterCredential, error) {
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusters, rbac.ActionRead, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusterCredentials, rbac.ActionRead, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	return h.next.GetDatabaseClusterCredentials(ctx, namespace, name)
}

func (h *rbacHandler) GetDatabaseClusterComponents(ctx context.Context, namespace, name string) ([]api.DatabaseClusterComponent, error) {
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusters, rbac.ActionRead, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	return h.next.GetDatabaseClusterComponents(ctx, namespace, name)
}

func (h *rbacHandler) GetDatabaseClusterComponentLogs(ctx context.Context, namespace, clusterName, componentName string, params api.GetDatabaseClusterComponentLogsParams, stream handlers.StreamFunc) error {
	// if users have access to the DB cluster let's give them access to read the logs
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusters, rbac.ActionRead, rbac.ObjectName(namespace, clusterName)); err != nil {
		return err
	}
	return h.next.GetDatabaseClusterComponentLogs(ctx, namespace, clusterName, componentName, params, stream)
}

func (h *rbacHandler) GetDatabaseClusterPitr(ctx context.Context, namespace, name string) (*api.DatabaseClusterPitr, error) {
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusters, rbac.ActionRead, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	return h.next.GetDatabaseClusterPitr(ctx, namespace, name)
}

func (h *rbacHandler) enforceDBClusterRead(ctx context.Context, db *everestv1alpha1.DatabaseCluster) error {
	name := db.GetName()
	namespace := db.GetNamespace()
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusters, rbac.ActionRead, rbac.ObjectName(namespace, name)); err != nil {
		return err
	}

	// Check if the user has permissions for all backup-storages in the schedule?
	for _, sched := range db.Spec.Backup.Schedules {
		bsName := sched.BackupStorageName
		if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, rbac.ObjectName(namespace, bsName)); err != nil {
			return err
		}
	}
	// Check if the user has permission for the backup-storages used by PITR (if any)?
	if bsName := pointer.Get(db.Spec.Backup.PITR.BackupStorageName); bsName != "" {
		if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, rbac.ObjectName(namespace, bsName)); err != nil {
			return err
		}
	}
	// Check if the user has permissions for MonitoringConfig?
	if mcName := pointer.Get(db.Spec.Monitoring).MonitoringConfigName; mcName != "" {
		if err := h.enforce(ctx, rbac.ResourceMonitoringInstances, rbac.ActionRead, rbac.ObjectName(namespace, mcName)); err != nil {
			return err
		}
	}

	engineName := common.OperatorTypeToName[db.Spec.Engine.Type]
	if err := h.enforce(ctx, rbac.ResourceDatabaseEngines, rbac.ActionRead, rbac.ObjectName(namespace, engineName)); err != nil {
		return err
	}

	if lbcName := db.Spec.Proxy.Expose.LoadBalancerConfigName; lbcName != "" {
		if err := h.enforce(ctx, rbac.ResourceLoadBalancerConfigs, rbac.ActionRead, lbcName); err != nil {
			return err
		}
	}

	// Check permissions for engine features used in the database cluster.
	if err := h.enforceEngineFeaturesRead(ctx, db); err != nil {
		return err
	}

	return nil
}

func (h *rbacHandler) enforceEngineFeaturesRead(ctx context.Context, db *everestv1alpha1.DatabaseCluster) error {
	namespace := db.GetNamespace()

	// PSMDB features
	if db.Spec.Engine.Type == everestv1alpha1.DatabaseEnginePSMDB {
		psmdbFeatures := pointer.Get(pointer.Get(db.Spec.EngineFeatures).PSMDB)

		// SplitHorizonDNSConfig feature.
		if psmdbFeatures.SplitHorizonDNSConfigName != "" {
			if err := h.enforce(ctx, rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs,
				rbac.ActionRead,
				rbac.ObjectName(namespace, psmdbFeatures.SplitHorizonDNSConfigName),
			); err != nil {
				return err
			}
		}
	}

	// Rest of engine features can be added here.
	return nil
}

func (h *rbacHandler) CreateDatabaseClusterSecret(ctx context.Context, namespace, dbName string, secret *corev1.Secret,
) (*corev1.Secret, error) {
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusters, rbac.ActionCreate, rbac.ObjectName(namespace, dbName)); err != nil {
		return nil, err
	}
	return h.next.CreateDatabaseClusterSecret(ctx, namespace, dbName, secret)
}
