package validation

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"golang.org/x/mod/semver"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	operatorUtils "github.com/percona/everest-operator/utils"
	"github.com/percona/everest/api"
	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/common"
)

const (
	dateFormat                      = "2006-01-02T15:04:05Z"
	minShardsNum                    = 1
	minConfigServersNumNNodeReplset = 3
	maxPXCEngineReplicas            = 5
	minPXCProxyReplicas             = 2
	minConfigServersNum1NodeReplset = 1
	pgReposLimit                    = 3
)

func (h *validateHandler) CreateDatabaseCluster(ctx context.Context, db *everestv1alpha1.DatabaseCluster) (*everestv1alpha1.DatabaseCluster, error) {
	if err := h.validateDatabaseClusterCR(ctx, db.GetNamespace(), db); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	if currentDB, err := h.kubeConnector.GetDatabaseCluster(ctx, types.NamespacedName{Namespace: db.GetNamespace(), Name: db.GetName()}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to check if DB cluster with name already exists in namespace: %w", err)
		}
	} else if currentDB.GetName() != "" {
		return nil, fmt.Errorf("db cluster with name '%s' already exists in namespace '%s'", db.GetName(), db.GetNamespace())
	}

	return h.next.CreateDatabaseCluster(ctx, db)
}

func (h *validateHandler) ListDatabaseClusters(ctx context.Context, namespace string) (*everestv1alpha1.DatabaseClusterList, error) {
	return h.next.ListDatabaseClusters(ctx, namespace)
}

func (h *validateHandler) DeleteDatabaseCluster(ctx context.Context, namespace, name string, req *api.DeleteDatabaseClusterParams) error {
	return h.next.DeleteDatabaseCluster(ctx, namespace, name, req)
}

func (h *validateHandler) UpdateDatabaseCluster(ctx context.Context, db *everestv1alpha1.DatabaseCluster) (*everestv1alpha1.DatabaseCluster, error) {
	if err := h.validateDatabaseClusterCR(ctx, db.GetNamespace(), db); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	current, err := h.kubeConnector.GetDatabaseCluster(ctx, types.NamespacedName{Namespace: db.GetNamespace(), Name: db.GetName()})
	if err != nil {
		return nil, fmt.Errorf("failed to GetDatabaseCluster: %w", err)
	}
	if err := h.validateDatabaseClusterOnUpdate(db, current); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	return h.next.UpdateDatabaseCluster(ctx, db)
}

func (h *validateHandler) GetDatabaseCluster(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseCluster, error) {
	return h.next.GetDatabaseCluster(ctx, namespace, name)
}

func (h *validateHandler) GetDatabaseClusterCredentials(ctx context.Context, namespace, name string) (*api.DatabaseClusterCredential, error) {
	return h.next.GetDatabaseClusterCredentials(ctx, namespace, name)
}

func (h *validateHandler) GetDatabaseClusterComponents(ctx context.Context, namespace, name string) ([]api.DatabaseClusterComponent, error) {
	return h.next.GetDatabaseClusterComponents(ctx, namespace, name)
}

func (h *validateHandler) GetDatabaseClusterComponentLogs(ctx context.Context, namespace, clusterName, componentName string, params api.GetDatabaseClusterComponentLogsParams, stream handlers.StreamFunc) error {
	// in openapi 3.1 can be handled by the schema itself, but currently we use 3.0.2
	if params.SinceSeconds != nil && params.SinceTime != nil {
		return errors.Join(ErrInvalidRequest, errors.New("at most one of sinceSeconds or sinceTime may be specified"))
	}
	return h.next.GetDatabaseClusterComponentLogs(ctx, namespace, clusterName, componentName, params, stream)
}

func (h *validateHandler) GetDatabaseClusterPitr(ctx context.Context, namespace, name string) (*api.DatabaseClusterPitr, error) {
	return h.next.GetDatabaseClusterPitr(ctx, namespace, name)
}

//nolint:cyclop
func (h *validateHandler) validateDatabaseClusterCR(
	ctx context.Context,
	namespace string,
	databaseCluster *everestv1alpha1.DatabaseCluster,
) error {
	if err := validateMetadata(databaseCluster); err != nil {
		return err
	}
	if err := validateCreateDatabaseClusterRequest(databaseCluster); err != nil {
		return err
	}

	engineName, ok := common.OperatorTypeToName[databaseCluster.Spec.Engine.Type]
	if !ok {
		return errors.New("unsupported database engine")
	}
	engine, err := h.kubeConnector.GetDatabaseEngine(ctx, types.NamespacedName{Namespace: namespace, Name: engineName})
	if err != nil {
		return err
	}
	if err := validateEngine(databaseCluster, engine); err != nil {
		return err
	}
	if databaseCluster.Spec.Proxy.Type != "" {
		if err := validateProxy(databaseCluster); err != nil {
			return err
		}
	}
	if err := validateBackupSpec(databaseCluster); err != nil {
		return err
	}

	if err = h.validateBackupStoragesFor(ctx, namespace, databaseCluster); err != nil {
		return err
	}

	if databaseCluster.Spec.DataSource != nil {
		if err := validateDataSource(databaseCluster.Spec.DataSource); err != nil {
			return err
		}
	}

	if databaseCluster.Spec.Engine.Type == everestv1alpha1.DatabaseEnginePostgresql {
		if err = h.validatePGSchedulesRestrictions(ctx, databaseCluster); err != nil {
			return err
		}
		if err = validatePGReposForAPIDB(ctx, databaseCluster, h.kubeConnector.ListDatabaseClusterBackups); err != nil {
			return err
		}
	}
	if err := validateSharding(databaseCluster); err != nil {
		return err
	}

	if err = h.validatePodSchedulingPolicy(ctx, databaseCluster); err != nil {
		h.log.Errorf("failed to validate .spec.podSchedulingPolicyName='%s': %v", databaseCluster.Spec.PodSchedulingPolicyName, err)
		return err
	}
	return validateResourceLimits(databaseCluster)
}

func validateSharding(dbc *everestv1alpha1.DatabaseCluster) error {
	if dbc.Spec.Sharding == nil || !dbc.Spec.Sharding.Enabled {
		return nil
	}
	if dbc.Spec.Engine.Type != everestv1alpha1.DatabaseEnginePSMDB {
		return errShardingIsNotSupported
	}
	if dbc.Spec.Engine.Version == "" {
		return errShardingVersion
	}
	version, err := goversion.NewVersion(dbc.Spec.Engine.Version)
	if err != nil {
		return errShardingVersion
	}
	if !common.CheckConstraint(version, ">=1.17.0") {
		return errShardingVersion
	}
	if dbc.Spec.Sharding.Shards < minShardsNum {
		return errInsufficientShardsNumber
	}
	if dbc.Spec.Engine.Replicas != 0 &&
		dbc.Spec.Engine.Replicas == 1 &&
		dbc.Spec.Sharding.ConfigServer.Replicas < minConfigServersNum1NodeReplset {
		return errInsufficientCfgSrvNumber1Node
	}
	if dbc.Spec.Engine.Replicas != 0 &&
		dbc.Spec.Engine.Replicas > 1 &&
		dbc.Spec.Sharding.ConfigServer.Replicas < minConfigServersNumNNodeReplset {
		return errInsufficientCfgSrvNumber
	}
	if dbc.Spec.Sharding.ConfigServer.Replicas%2 == 0 {
		return errEvenServersNumber
	}
	return nil
}

func validateCreateDatabaseClusterRequest(dbc *everestv1alpha1.DatabaseCluster) error {
	return operatorUtils.ValidateEverestResourceName(dbc.GetName(), "metadata.name")
}

func validateEngine(databaseCluster *everestv1alpha1.DatabaseCluster, engine *everestv1alpha1.DatabaseEngine) error {
	if err := validateVersion(databaseCluster.Spec.Engine.Version, engine); err != nil {
		return err
	}

	switch databaseCluster.Spec.Engine.Type {
	case everestv1alpha1.DatabaseEnginePXC:
		if databaseCluster.Spec.Engine.Replicas > 0 && databaseCluster.Spec.Engine.Replicas%2 == 0 {
			return errEvenEngineReplicas
		}
		if databaseCluster.Spec.Engine.Replicas > 0 && databaseCluster.Spec.Engine.Replicas > maxPXCEngineReplicas {
			return errMaxPXCEngineReplicas
		}
	case everestv1alpha1.DatabaseEnginePSMDB:
		if databaseCluster.Spec.Engine.Replicas > 0 && databaseCluster.Spec.Engine.Replicas%2 == 0 {
			return errEvenEngineReplicas
		}
	case everestv1alpha1.DatabaseEnginePostgresql:
		// no restrictions for now
	}
	return nil
}

func validateVersion(version string, engine *everestv1alpha1.DatabaseEngine) error {
	if version != "" {
		if len(engine.Spec.AllowedVersions) > 0 {
			if !containsVersion(version, engine.Spec.AllowedVersions) {
				return fmt.Errorf("using %s version for %s is not allowed", version, engine.Spec.Type)
			}
			return nil
		}
		if _, ok := engine.Status.AvailableVersions.Engine[version]; !ok {
			return fmt.Errorf("%s is not in available versions list", version)
		}
	}
	return nil
}

func containsVersion(version string, versions []string) bool {
	if version == "" {
		return true
	}
	for _, allowedVersion := range versions {
		if version == allowedVersion {
			return true
		}
	}
	return false
}

func validateProxy(databaseCluster *everestv1alpha1.DatabaseCluster) error {
	if err := validateProxyType(databaseCluster.Spec.Engine.Type, databaseCluster.Spec.Proxy.Type); err != nil {
		return err
	}

	if databaseCluster.Spec.Engine.Type == everestv1alpha1.DatabaseEnginePXC {
		if databaseCluster.Spec.Engine.Replicas > 1 &&
			databaseCluster.Spec.Proxy.Replicas != nil && *databaseCluster.Spec.Proxy.Replicas < minPXCProxyReplicas {
			return errMinPXCProxyReplicas
		}
	}

	rangesMap := make(map[everestv1alpha1.IPSourceRange]struct{})
	for _, sourceRange := range databaseCluster.Spec.Proxy.Expose.IPSourceRanges {
		if _, ok := rangesMap[sourceRange]; ok {
			return ErrDuplicateSourceRange(sourceRange)
		}
		rangesMap[sourceRange] = struct{}{}
	}

	return nil
}

func validateProxyType(engineType everestv1alpha1.EngineType, proxyType everestv1alpha1.ProxyType) error {
	if engineType == everestv1alpha1.DatabaseEnginePXC {
		if proxyType != everestv1alpha1.ProxyTypeProxySQL && proxyType != everestv1alpha1.ProxyTypeHAProxy {
			return errUnsupportedPXCProxy
		}
	}

	if engineType == everestv1alpha1.DatabaseEnginePostgresql && proxyType != everestv1alpha1.ProxyTypePGBouncer {
		return errUnsupportedPGProxy
	}
	if engineType == everestv1alpha1.DatabaseEnginePSMDB && proxyType != everestv1alpha1.ProxyTypeMongos {
		return errUnsupportedPSMDBProxy
	}
	return nil
}

func validateBackupSpec(cluster *everestv1alpha1.DatabaseCluster) error {
	if err := validatePitrSpec(cluster); err != nil {
		return err
	}

	for _, schedule := range cluster.Spec.Backup.Schedules {
		if schedule.Name == "" {
			return errNoNameInSchedule
		}
		if schedule.Enabled && schedule.BackupStorageName == "" {
			return errScheduleNoBackupStorageName
		}
	}
	return checkDuplicateSchedules(cluster.Spec.Backup.Schedules)
}

func validatePitrSpec(cluster *everestv1alpha1.DatabaseCluster) error {
	if !cluster.Spec.Backup.PITR.Enabled {
		return nil
	}

	if cluster.Spec.Engine.Type == everestv1alpha1.DatabaseEnginePXC &&
		(cluster.Spec.Backup.PITR.BackupStorageName == nil || *cluster.Spec.Backup.PITR.BackupStorageName == "") {
		return errPitrNoBackupStorageName
	}

	if cluster.Spec.Backup.PITR.UploadIntervalSec != nil && *cluster.Spec.Backup.PITR.UploadIntervalSec <= 0 {
		return errPitrUploadInterval
	}

	return nil
}

func checkDuplicateSchedules(schedules []everestv1alpha1.BackupSchedule) error {
	unique := make(map[string]struct{})
	for _, schedule := range schedules {
		key := schedule.Schedule
		if _, ok := unique[key]; ok {
			return errDuplicatedSchedules
		}
		unique[key] = struct{}{}
	}
	return nil
}

func (h *validateHandler) validateBackupStoragesFor(
	ctx context.Context,
	namespace string,
	databaseCluster *everestv1alpha1.DatabaseCluster,
) error {
	storages := make(map[string]bool)
	for _, schedule := range databaseCluster.Spec.Backup.Schedules {
		storages[schedule.BackupStorageName] = true
	}

	if databaseCluster.Spec.Engine.Type == everestv1alpha1.DatabaseEnginePSMDB {
		// attempt to configure more than one storage for psmdb
		if len(storages) > 1 {
			return errPSMDBMultipleStorages
		}
		// attempt to use a storage other than the active one
		activeStorage := databaseCluster.Status.ActiveStorage
		for name := range storages {
			if activeStorage != "" && name != activeStorage {
				return errPSMDBViolateActiveStorage
			}
		}
	}

	if !databaseCluster.Spec.Backup.PITR.Enabled {
		return nil
	}

	if databaseCluster.Spec.Engine.Type == everestv1alpha1.DatabaseEnginePXC {
		if databaseCluster.Spec.Backup.PITR.BackupStorageName == nil || *databaseCluster.Spec.Backup.PITR.BackupStorageName == "" {
			return errPitrNoBackupStorageName
		}
		storage, err := h.kubeConnector.GetBackupStorage(ctx,
			types.NamespacedName{
				Namespace: namespace,
				Name:      *databaseCluster.Spec.Backup.PITR.BackupStorageName,
			},
		)
		if err != nil {
			return err
		}
		// pxc only supports s3 for pitr
		if storage.Spec.Type != everestv1alpha1.BackupStorageTypeS3 {
			return errPXCPitrS3Only
		}
	}

	return nil
}

func validateDataSource(dataSource *everestv1alpha1.DataSource) error {
	if dataSource == nil {
		return nil
	}

	ensureOneDataSourceSpecified := func() bool {
		sources := 0
		if dataSource.DBClusterBackupName != "" {
			sources++
		}
		if dataSource.BackupSource != nil {
			sources++
		}
		if dataSource.DataImport != nil {
			sources++
		}

		return sources == 1
	}
	if !ensureOneDataSourceSpecified() {
		return errDataSourceConfig
	}

	if dataSource.BackupSource != nil {
		if dataSource.BackupSource.BackupStorageName == "" {
			return errDataSourceNoBackupStorageName
		}

		if dataSource.BackupSource.Path == "" {
			return errDataSourceNoPath
		}
	}

	if dataSource.PITR != nil { //nolint:nestif
		if dataSource.PITR.Type == "" || dataSource.PITR.Type == everestv1alpha1.PITRTypeDate {
			if dataSource.PITR.Date == nil {
				return errDataSourceNoPitrDateSpecified
			}

			if dataSource.PITR.Date.IsZero() {
				return errDataSourceWrongDateFormat
			}
		} else {
			return errUnsupportedPitrType
		}
	}
	return nil
}

func (h *validateHandler) validatePGSchedulesRestrictions(ctx context.Context, newDbc *everestv1alpha1.DatabaseCluster) error {
	dbcName := newDbc.GetName()
	dbcNamespace := newDbc.GetNamespace()
	existingDbc, err := h.kubeConnector.GetDatabaseCluster(ctx, types.NamespacedName{Namespace: dbcNamespace, Name: dbcName})
	if err != nil {
		// if there was no such cluster before (creating cluster) - check only the duplicates for storages
		if k8serrors.IsNotFound(err) {
			return checkStorageDuplicates(newDbc)
		}
		return err
	}
	// if there is an old cluster - compare old and new schedules
	return checkSchedulesChanges(existingDbc, newDbc)
}

func checkStorageDuplicates(dbc *everestv1alpha1.DatabaseCluster) error {
	if len(dbc.Spec.Backup.Schedules) == 0 {
		return nil
	}
	schedules := dbc.Spec.Backup.Schedules
	storagesMap := make(map[string]bool)
	for _, schedule := range schedules {
		if _, inUse := storagesMap[schedule.BackupStorageName]; inUse {
			return errDuplicatedStoragePG
		}
		storagesMap[schedule.BackupStorageName] = true
	}
	return nil
}

func checkSchedulesChanges(oldDbc, newDbc *everestv1alpha1.DatabaseCluster) error {
	if len(newDbc.Spec.Backup.Schedules) == 0 {
		return nil
	}
	newSchedules := newDbc.Spec.Backup.Schedules
	for _, oldSched := range oldDbc.Spec.Backup.Schedules {
		for _, newShed := range newSchedules {
			// check the existing schedule storage wasn't changed
			if oldSched.Name == newShed.Name {
				if oldSched.BackupStorageName != newShed.BackupStorageName {
					return errStorageChangePG
				}
			}
		}
	}
	// check there is no duplicated storages
	return checkStorageDuplicates(newDbc)
}

func validatePGReposForAPIDB(
	ctx context.Context,
	dbc *everestv1alpha1.DatabaseCluster,
	getBackupsFunc func(ctx context.Context, opts ...ctrlclient.ListOption) (*everestv1alpha1.DatabaseClusterBackupList, error),
) error {
	bs := make(map[string]bool)
	for _, shed := range dbc.Spec.Backup.Schedules {
		bs[shed.BackupStorageName] = true
	}

	dbcName := dbc.GetName()
	dbcNamespace := dbc.GetNamespace()

	backups, err := getBackupsFunc(ctx, ctrlclient.InNamespace(dbcNamespace), ctrlclient.MatchingLabels{common.DatabaseClusterNameLabel: dbcName})
	if err != nil {
		return err
	}

	for _, backup := range backups.Items {
		// repos count is increased only if there wasn't such a BS used
		if _, ok := bs[backup.Spec.BackupStorageName]; !ok {
			bs[backup.Spec.BackupStorageName] = true
		}
	}

	// second check if there are too many repos used.
	if len(bs) > pgReposLimit {
		return errTooManyPGStorages
	}

	return nil
}

func validateResourceLimits(cluster *everestv1alpha1.DatabaseCluster) error {
	if err := ensureNonEmptyResources(cluster); err != nil {
		return err
	}
	if err := validateCPU(cluster); err != nil {
		return err
	}
	if err := validateMemory(cluster); err != nil {
		return err
	}
	return validateStorageSize(cluster)
}

func ensureNonEmptyResources(cluster *everestv1alpha1.DatabaseCluster) error {
	if cluster.Spec.Engine.Resources.CPU.IsZero() && cluster.Spec.Engine.Resources.Memory.IsZero() {
		return errNoResourceDefined
	}
	if cluster.Spec.Engine.Resources.CPU.IsZero() {
		return errNotEnoughCPU
	}
	if cluster.Spec.Engine.Resources.Memory.IsZero() {
		return errNotEnoughMemory
	}
	return nil
}

func validateCPU(cluster *everestv1alpha1.DatabaseCluster) error {
	cpu := cluster.Spec.Engine.Resources.CPU
	if cpu.Cmp(minCPUQuantity) == -1 {
		return errNotEnoughCPU
	}
	return nil
}

func validateMemory(cluster *everestv1alpha1.DatabaseCluster) error {
	mem := cluster.Spec.Engine.Resources.Memory
	if mem.Cmp(minMemQuantity) == -1 {
		return errNotEnoughMemory
	}
	return nil
}

func validateStorageSize(cluster *everestv1alpha1.DatabaseCluster) error {
	size := cluster.Spec.Engine.Storage.Size
	if size.Cmp(minStorageQuantity) == -1 {
		return errNotEnoughDiskSize
	}
	return nil
}

func validateMetadata(obj metav1.Object) error {
	if obj.GetNamespace() == "" {
		return errEmptyNamespace
	}
	if obj.GetName() == "" {
		return errEmptyName
	}
	return nil
}

func (h *validateHandler) validateDatabaseClusterOnUpdate(
	dbc, oldDB *everestv1alpha1.DatabaseCluster,
) error {
	if !isDatabaseClusterUpdateAllowed(oldDB) {
		return fmt.Errorf("db operations are not allowed in current db state: %s", oldDB.Status.Status)
	}

	newVersion := dbc.Spec.Engine.Version
	oldVersion := oldDB.Spec.Engine.Version
	if newVersion != "" && newVersion != oldVersion {
		if err := validateDBEngineVersionUpgrade(oldDB.Spec.Engine.Type, newVersion, oldVersion); err != nil {
			return err
		}
	}
	if dbc.Spec.Engine.Replicas < oldDB.Spec.Engine.Replicas && dbc.Spec.Engine.Replicas == 1 {
		// XXX: We can scale down multiple node clusters to a single node but we need to set
		// `allowUnsafeConfigurations` to `true`. Having this configuration is not recommended
		// and makes a database cluster unsafe. Once allowUnsafeConfigurations set to true you
		// can't set it to false for all operators and psmdb operator does not support it.
		//
		// Once it is supported by all operators we can revert this.
		return fmt.Errorf("cannot scale down %d node cluster to 1. The operation is not supported", oldDB.Spec.Engine.Replicas)
	}

	// Do not allow shrinking storage size.
	if dbc.Spec.Engine.Storage.Size.Cmp(oldDB.Spec.Engine.Storage.Size) < 0 {
		return errCannotShrinkStorageSize
	}

	if err := validateShardingOnUpdate(dbc, oldDB); err != nil {
		return err
	}
	return nil
}

// validateDBEngineVersionUpgrade validates if upgrade of DBEngine from `oldVersion` to `newVersion` is allowed.
func validateDBEngineVersionUpgrade(engineType everestv1alpha1.EngineType, newVersion, oldVersion string) error {
	// Ensure a "v" prefix so that it is a valid semver.
	if !strings.HasPrefix(newVersion, "v") {
		newVersion = "v" + newVersion
	}
	if !strings.HasPrefix(oldVersion, "v") {
		oldVersion = "v" + oldVersion
	}

	// Check semver validity.
	if !semver.IsValid(newVersion) {
		return errInvalidVersion
	}

	// We will not allow downgrades.
	if semver.Compare(newVersion, oldVersion) < 0 {
		return errDBEngineDowngrade
	}
	// We will not allow major upgrades for PXC and PG.
	// - PXC: Major upgrades are not supported.
	// - PG: Major upgrades are in technical preview. https://docs.percona.com/percona-operator-for-postgresql/2.0/update.html#major-version-upgrade
	if engineType != everestv1alpha1.DatabaseEnginePSMDB && semver.Major(oldVersion) != semver.Major(newVersion) {
		return errDBEngineMajorVersionUpgrade
	}

	// It's fine to ignore the errors here because we have already validated the version.
	newMajorInt, _ := strconv.Atoi(semver.Major(newVersion)[1:])
	oldMajorInt, _ := strconv.Atoi(semver.Major(oldVersion)[1:])
	// We will not allow major upgrades if the versions are not sequential.
	if newMajorInt-oldMajorInt > 1 {
		fmt.Println("errDBEngineMajorUpgradeNotSeq")
		return errDBEngineMajorUpgradeNotSeq
	}
	return nil
}

func validateShardingOnUpdate(dbc, oldDB *everestv1alpha1.DatabaseCluster) error {
	if oldDB.Spec.Sharding == nil || !oldDB.Spec.Sharding.Enabled {
		if dbc.Spec.Sharding != nil && dbc.Spec.Sharding.Enabled {
			return errShardingEnablingNotSupported
		}
		return nil
	}
	if dbc.Spec.Sharding == nil || !dbc.Spec.Sharding.Enabled {
		return errDisableShardingNotSupported
	}
	return validateSharding(dbc)
}

// isDatabaseClusterUpdateAllowed checks if the requested change is allowed for the database cluster.
// The returns false in case DB cluster is in one of the following states:
// - restoring
// - deleting
// - upgrading
// - resizingVolumes
func isDatabaseClusterUpdateAllowed(currentDB *everestv1alpha1.DatabaseCluster) bool {
	if currentDB == nil {
		return false
	}

	switch currentDB.Status.Status {
	case everestv1alpha1.AppStateRestoring,
		everestv1alpha1.AppStateDeleting,
		everestv1alpha1.AppStateUpgrading,
		everestv1alpha1.AppStateResizingVolumes:
		return false
	}

	return true
}

var (
	// Engine type mismatch error
	errDBClusterPSPEngineTypeMismatch = func(pspName string, engineType everestv1alpha1.EngineType) error {
		return fmt.Errorf("requested pod scheduling policy='%s' is not applicable with engineType='%s'", pspName, engineType)
	}
	// Affinity config errors
	errDBClusterInvalidPSPAffinityConfig = func(pspName string) error {
		return fmt.Errorf("pod scheduling policy='%s' is not applicable: affinityConfig is absent or empty", pspName)
	}
	// PXC affinity config errors
	errDBClusterInvalidPSPAffinityPXCEmpty = func(pspName string) error {
		return fmt.Errorf("pod scheduling policy='%s' is not applicable: .spec.affinityConfig.pxc is required", pspName)
	}
	errDBClusterInvalidPSPAffinityPXCComponentsEmpty = func(pspName string) error {
		return fmt.Errorf("pod scheduling policy='%s' is not applicable: .spec.affinityConfig.pxc.engine or .spec.affinityConfig.pxc.proxy is required", pspName)
	}
	// PSMDB affinity config errors
	errDBClusterInvalidPSPAffinityPSMDBEmpty = func(pspName string) error {
		return fmt.Errorf("pod scheduling policy='%s' is not applicable: .spec.affinityConfig.psmdb is required", pspName)
	}
	errDBClusterInvalidPSPAffinityPSMDBComponentsEmpty = func(pspName string) error {
		return fmt.Errorf("pod scheduling policy='%s' is not applicable: .spec.affinityConfig.psmdb.engine or .spec.affinityConfig.psmdb.proxy or .spec.affinityConfig.psmdb.configServer is required", pspName)
	}
	// Postgresql affinity config errors
	errDBClusterInvalidPSPAffinityPostgresqlEmpty = func(pspName string) error {
		return fmt.Errorf("pod scheduling policy='%s' is not applicable: .spec.affinityConfig.postgresql is required", pspName)
	}
	errDBClusterInvalidPSPAffinityPostgresqlComponentsEmpty = func(pspName string) error {
		return fmt.Errorf("pod scheduling policy='%s' is not applicable: .spec.affinityConfig.postgresql.engine or .spec.affinityConfig.postgresql.proxy is required", pspName)
	}
)

func (h *validateHandler) validatePodSchedulingPolicy(ctx context.Context, db *everestv1alpha1.DatabaseCluster) error {
	var psp *everestv1alpha1.PodSchedulingPolicy
	var err error
	pspName := db.Spec.PodSchedulingPolicyName

	if pspName == "" {
		return nil
	}

	if psp, err = h.kubeConnector.GetPodSchedulingPolicy(ctx, types.NamespacedName{Name: pspName}); err != nil {
		if k8serrors.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("failed to check if requested pod scheduling policy with name='%s' exists: %v", pspName, err)
	}

	if psp.Spec.EngineType != db.Spec.Engine.Type {
		return errDBClusterPSPEngineTypeMismatch(pspName, db.Spec.Engine.Type)
	}

	affinityConfig := psp.Spec.AffinityConfig
	if affinityConfig == nil {
		return errDBClusterInvalidPSPAffinityConfig(pspName)
	}

	// Policy has affinityConfig - need to validate it.
	switch psp.Spec.EngineType {
	case everestv1alpha1.DatabaseEnginePXC:
		if affinityConfig.PXC == nil {
			return errDBClusterInvalidPSPAffinityPXCEmpty(pspName)
		}
		if affinityConfig.PXC.Engine == nil && affinityConfig.PXC.Proxy == nil {
			return errDBClusterInvalidPSPAffinityPXCComponentsEmpty(pspName)
		}
	case everestv1alpha1.DatabaseEnginePSMDB:
		if affinityConfig.PSMDB == nil {
			return errDBClusterInvalidPSPAffinityPSMDBEmpty(pspName)
		}
		if affinityConfig.PSMDB.Engine == nil && affinityConfig.PSMDB.Proxy == nil && affinityConfig.PSMDB.ConfigServer == nil {
			return errDBClusterInvalidPSPAffinityPSMDBComponentsEmpty(pspName)
		}
	case everestv1alpha1.DatabaseEnginePostgresql:
		if affinityConfig.PostgreSQL == nil {
			return errDBClusterInvalidPSPAffinityPostgresqlEmpty(pspName)
		}
		if affinityConfig.PostgreSQL.Engine == nil && affinityConfig.PostgreSQL.Proxy == nil {
			return errDBClusterInvalidPSPAffinityPostgresqlComponentsEmpty(pspName)
		}
	}
	return nil
}

func (h *validateHandler) CreateDatabaseClusterSecret(ctx context.Context, namespace, dbName string, secret *corev1.Secret,
) (*corev1.Secret, error) {
	return h.next.CreateDatabaseClusterSecret(ctx, namespace, dbName, secret)
}
