package k8s

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/cenkalti/backoff"
	goversion "github.com/hashicorp/go-version"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	versionservice "github.com/percona/everest/pkg/version_service"
)

var (
	errDBEngineUpgradeUnavailable   = errors.New("provided target version is not available for upgrade")
	errDBEngineInvalidTargetVersion = errors.New("invalid target version provided for upgrade")
)

func (h *k8sHandler) ListDatabaseEngines(ctx context.Context, namespace string) (*everestv1alpha1.DatabaseEngineList, error) {
	return h.kubeConnector.ListDatabaseEngines(ctx, ctrlclient.InNamespace(namespace))
}

func (h *k8sHandler) GetDatabaseEngine(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseEngine, error) {
	return h.kubeConnector.GetDatabaseEngine(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

func (h *k8sHandler) UpdateDatabaseEngine(ctx context.Context, req *everestv1alpha1.DatabaseEngine) (*everestv1alpha1.DatabaseEngine, error) {
	return h.kubeConnector.UpdateDatabaseEngine(ctx, req)
}

func (h *k8sHandler) GetUpgradePlan(ctx context.Context, namespace string) (*api.UpgradePlan, error) {
	result, err := h.getUpgradePlan(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to getUpgradePlan: %w", err)
	}
	// No upgrades available, so we will check if our clusters are ready for current version.
	if len(pointer.Get(result.Upgrades)) == 0 {
		result.PendingActions = pointer.To([]api.UpgradeTask{})
		engines, err := h.kubeConnector.ListDatabaseEngines(ctx, ctrlclient.InNamespace(namespace))
		if err != nil {
			return nil, err
		}
		for _, engine := range engines.Items {
			tasks, err := h.getDBPostUpgradeTasks(ctx, &engine)
			if err != nil {
				return nil, err
			}
			for _, t := range tasks {
				*result.PendingActions = append(*result.PendingActions, api.UpgradeTask{
					Name:        t.Name,
					PendingTask: t.PendingTask,
					Message:     t.Message,
				})
			}
		}
	}
	return result, nil
}

func (h *k8sHandler) ApproveUpgradePlan(ctx context.Context, namespace string) error {
	up, err := h.getUpgradePlan(ctx, namespace)
	if err != nil {
		return err
	}
	// lock all engines that will be upgraded.
	if err := h.setLockDBEnginesForUpgrade(ctx, namespace, up, true); err != nil {
		return errors.Join(err, errors.New("failed to lock engines"))
	}
	// Check if we're ready to upgrade?
	if slices.ContainsFunc(pointer.Get(up.PendingActions), func(task api.UpgradeTask) bool {
		return pointer.Get(task.PendingTask) != api.Ready
	}) {
		// Not ready for upgrade, release the lock and return a failured message.
		if err := h.setLockDBEnginesForUpgrade(ctx, namespace, up, false); err != nil {
			return errors.Join(err, errors.New("failed to release lock"))
		}
		return errors.New("one or more database clusters are not ready for upgrade")
	}
	// start upgrade process.
	if err := h.startOperatorUpgradeWithRetry(ctx, namespace); err != nil {
		// Upgrade has failed, so we release the lock.
		if err := h.setLockDBEnginesForUpgrade(ctx, namespace, up, false); err != nil {
			return errors.Join(err, errors.New("failed to release lock"))
		}
		return err
	}
	return nil
}

func (h *k8sHandler) setLockDBEnginesForUpgrade(
	ctx context.Context,
	namespace string,
	up *api.UpgradePlan,
	lock bool,
) error {
	return backoff.Retry(func() error {
		for _, upgrade := range pointer.Get(up.Upgrades) {
			if err := h.kubeConnector.SetDatabaseEngineLock(ctx, types.NamespacedName{Namespace: namespace, Name: pointer.Get(upgrade.Name)}, lock); err != nil {
				return err
			}
		}
		return nil
	}, backoff.WithContext(everestAPIConstantBackoff, ctx),
	)
}

func (h *k8sHandler) getUpgradePlan(
	ctx context.Context,
	namespace string,
) (*api.UpgradePlan, error) {
	engines, err := h.kubeConnector.ListDatabaseEngines(ctx, ctrlclient.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	result := &api.UpgradePlan{
		Upgrades:       pointer.To([]api.Upgrade{}),
		PendingActions: pointer.To([]api.UpgradeTask{}),
	}

	for _, engine := range engines.Items {
		nextVersion := engine.Status.GetNextUpgradeVersion()
		if nextVersion == "" {
			continue
		}

		upgrade := &api.Upgrade{
			CurrentVersion: pointer.To(engine.Status.OperatorVersion),
			Name:           pointer.To(engine.GetName()),
			TargetVersion:  pointer.To(nextVersion),
		}
		*result.Upgrades = append(*result.Upgrades, *upgrade)
		pf, err := h.getOperatorUpgradePreflight(ctx, nextVersion, &engine)
		if err != nil {
			return nil, err
		}
		*result.PendingActions = append(*result.PendingActions, pf.databases...)
	}
	return result, nil
}

type operatorUpgradePreflight struct {
	currentVersion string
	databases      []api.UpgradeTask
}

type upgradePreflightCheckArgs struct {
	targetVersion  string
	engine         *everestv1alpha1.DatabaseEngine
	versionService versionservice.Interface
}

func (h *k8sHandler) getOperatorUpgradePreflight(
	ctx context.Context,
	targetVersion string,
	engine *everestv1alpha1.DatabaseEngine,
) (*operatorUpgradePreflight, error) {
	namespace := engine.GetNamespace()
	// Get all database clusters in the namespace.
	databases, err := h.kubeConnector.ListDatabaseClusters(ctx, ctrlclient.InNamespace(namespace))
	if err != nil {
		return nil, err
	}
	// Filter out databases not using this engine type.
	databases.Items = slices.DeleteFunc(databases.Items, func(db everestv1alpha1.DatabaseCluster) bool {
		return db.Spec.Engine.Type != engine.Spec.Type
	})

	if err := validateOperatorUpgradeVersion(engine.Status.OperatorVersion, targetVersion); err != nil {
		return nil, err
	}

	args := upgradePreflightCheckArgs{
		targetVersion:  targetVersion,
		engine:         engine,
		versionService: versionservice.New(h.versionServiceURL),
	}
	result, err := getUpgradePreflightChecksResult(ctx, databases.Items, args)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to run preflight checks"))
	}
	return result, nil
}

func getUpgradePreflightChecksResult(
	ctx context.Context,
	dbs []everestv1alpha1.DatabaseCluster,
	args upgradePreflightCheckArgs,
) (*operatorUpgradePreflight, error) {
	// Check that this version is available for upgrade.
	if u := args.engine.Status.GetPendingUpgrade(args.targetVersion); u == nil {
		return nil, errDBEngineUpgradeUnavailable
	}

	// Perform checks for each given DB.
	dbResults := make([]api.UpgradeTask, 0, len(dbs))
	for _, db := range dbs {
		result, err := getUpgradePreflightCheckResultForDatabase(ctx, db, args)
		if err != nil {
			return nil, err
		}
		dbResults = append(dbResults, result)
	}

	// Sort by name.
	sort.Slice(dbResults, func(i, j int) bool {
		return strings.Compare(
			pointer.Get(dbResults[i].Name),
			pointer.Get(dbResults[j].Name),
		) < 0
	})

	return &operatorUpgradePreflight{
		databases:      dbResults,
		currentVersion: args.engine.Status.OperatorVersion,
	}, nil
}

func validateOperatorUpgradeVersion(currentVersion, targetVersion string) error {
	targetsv, err := goversion.NewSemver(targetVersion)
	if err != nil {
		return err
	}
	currentsv, err := goversion.NewSemver(currentVersion)
	if err != nil {
		return err
	}
	if targetsv.LessThanOrEqual(currentsv) {
		return errors.Join(errDBEngineInvalidTargetVersion, errors.New("target version must be greater than the current version"))
	}
	return nil
}

func getUpgradePreflightCheckResultForDatabase(
	ctx context.Context,
	database everestv1alpha1.DatabaseCluster,
	args upgradePreflightCheckArgs,
) (api.UpgradeTask, error) {
	// Check that the database engine is at the desired version.
	if valid, minReqVer, err := preflightCheckDBEngineVersion(ctx, database, args); err != nil {
		return api.UpgradeTask{},
			errors.Join(err, errors.New("failed to validate database engine version for operator upgrade"))
	} else if !valid {
		return api.UpgradeTask{
			Name:        pointer.To(database.GetName()),
			PendingTask: pointer.To(api.UpgradeEngine),
			Message: pointer.ToString(
				fmt.Sprintf("Upgrade DB version to %s or higher", minReqVer)),
		}, nil
	}

	// Check that DB is at recommended CRVersion.
	if recCRVersion := database.Status.RecommendedCRVersion; recCRVersion != nil {
		return api.UpgradeTask{
			Name:        pointer.To(database.GetName()),
			PendingTask: pointer.To(api.Restart),
			Message: pointer.ToString(
				fmt.Sprintf("Update CRVersion to %s", *recCRVersion)),
		}, nil
	}

	// Check that DB is running.
	if database.Status.Status != everestv1alpha1.AppStateReady {
		return api.UpgradeTask{
			Name:        pointer.To(database.GetName()),
			PendingTask: pointer.To(api.NotReady),
			Message:     pointer.ToString("Database is not ready"),
		}, nil
	}

	// Database is in desired state for performing operator upgrade.
	return api.UpgradeTask{
		Name:        pointer.To(database.GetName()),
		PendingTask: pointer.To(api.Ready),
	}, nil
}

// preflightCheckDBEngineVersion checks that the current database engine version is
// greater than or equal to the minimum supported version for the target operator version.
func preflightCheckDBEngineVersion(
	ctx context.Context,
	database everestv1alpha1.DatabaseCluster,
	args upgradePreflightCheckArgs,
) (bool, string, error) {
	engineType := args.engine.Spec.Type
	operator, found := versionservice.EngineTypeToOperatorName[engineType]
	if !found {
		return false, "", fmt.Errorf("unsupported engine type %s", engineType)
	}

	allSupportedVersions, err := args.versionService.GetSupportedEngineVersions(ctx, operator, args.targetVersion)
	if err != nil {
		return false, "", errors.Join(err, errors.New("failed to get supported engine versions"))
	}

	currentVersion, err := goversion.NewVersion(database.Spec.Engine.Version)
	if err != nil {
		return false, "", err
	}

	// We search for the smallest available version greater than or equal to the current major version.
	var minSupportedMajVersion *goversion.Version
	for _, supportedVersion := range allSupportedVersions {
		ver, err := goversion.NewVersion(supportedVersion)
		if err != nil {
			return false, "", err
		}
		if currentVersion.Segments()[0] > ver.Segments()[0] {
			continue // ignore if major version is less than the current major version.
		}
		if minSupportedMajVersion == nil || ver.LessThan(minSupportedMajVersion) {
			minSupportedMajVersion = ver
		}
	}

	if minSupportedMajVersion == nil {
		return false, "", fmt.Errorf("no minimum supported versions found for %s", operator)
	}

	return currentVersion.GreaterThanOrEqual(minSupportedMajVersion), minSupportedMajVersion.Original(), nil
}

func (h *k8sHandler) getDBPostUpgradeTasks(
	ctx context.Context,
	engine *everestv1alpha1.DatabaseEngine,
) ([]api.UpgradeTask, error) {
	namespace := engine.GetNamespace()
	// List all clusters in this namespace.
	clusters, err := h.kubeConnector.ListDatabaseClusters(ctx, ctrlclient.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	// Check that every cluster is using the recommended CRVersion.
	checks := []api.UpgradeTask{}
	for _, cluster := range clusters.Items {
		if cluster.Spec.Engine.Type != engine.Spec.Type {
			continue
		}
		check := api.UpgradeTask{
			Name: pointer.To(cluster.Name),
		}
		check.PendingTask = pointer.To(api.Ready)
		if recVer := cluster.Status.RecommendedCRVersion; recVer != nil {
			check.PendingTask = pointer.To(api.Restart)
			check.Message = pointer.To(fmt.Sprintf("Database needs restart to use CRVersion '%s'", *recVer))
		}
		checks = append(checks, check)
	}
	return checks, nil
}

// startOperatorUpgradeWithRetry wraps the startOperatorUpgrade function with a retry mechanism.
// This is done to reduce the chances of failures due to resource conflicts.
func (h *k8sHandler) startOperatorUpgradeWithRetry(ctx context.Context, namespace string) error {
	return backoff.Retry(func() error {
		return h.startOperatorUpgrade(ctx, namespace)
	},
		backoff.WithContext(everestAPIConstantBackoff, ctx),
	)
}

func (h *k8sHandler) startOperatorUpgrade(ctx context.Context, namespace string) error {
	engines, err := h.kubeConnector.ListDatabaseEngines(ctx, ctrlclient.InNamespace(namespace))
	if err != nil {
		return err
	}

	// gather install plans to approve.
	installPlans := []string{}
	for _, engine := range engines.Items {
		nextVer := engine.Status.GetNextUpgradeVersion()
		if nextVer == "" {
			continue
		}
		for _, pending := range engine.Status.PendingOperatorUpgrades {
			if pending.TargetVersion == nextVer {
				installPlans = append(installPlans, pending.InstallPlanRef.Name)
			}
		}
	}

	// de-duplicate the list.
	slices.Sort(installPlans)
	installPlans = slices.Compact(installPlans)

	// approve install plans.
	for _, plan := range installPlans {
		if err := backoff.Retry(func() error {
			_, err := h.kubeConnector.ApproveInstallPlan(ctx, types.NamespacedName{Namespace: namespace, Name: plan})
			return err
		}, backoff.WithContext(everestAPIConstantBackoff, ctx),
		); err != nil {
			return err
		}
	}
	return nil
}
