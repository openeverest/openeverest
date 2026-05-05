package upgrade

import (
	"context"
	"fmt"

	"github.com/AlekSi/pointer"
	goversion "github.com/hashicorp/go-version"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"helm.sh/helm/v3/pkg/cli/values"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/cli/helm"
	helmutils "github.com/percona/everest/pkg/cli/helm/utils"
	"github.com/percona/everest/pkg/cli/steps"
	"github.com/percona/everest/pkg/cli/utils"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
	. "github.com/percona/everest/pkg/utils/must" //nolint:revive,stylecheck
	"github.com/percona/everest/pkg/version"
)

func (u *Upgrade) newStepUpgradeCRDs() steps.Step {
	return steps.Step{
		Desc: "Upgrading Custom Resource Definitions",
		F: func(ctx context.Context) error {
			return u.upgradeCustomResourceDefinitions(ctx)
		},
	}
}

func (u *Upgrade) newStepUpgradeHelmChart() steps.Step {
	return steps.Step{
		Desc: "Upgrading Helm chart",
		F: func(ctx context.Context) error {
			return u.upgradeHelmChart(ctx)
		},
	}
}

func (u *Upgrade) newStepEnsureEverestOperator() steps.Step {
	return steps.Step{
		Desc: "Ensuring Everest operator deployment is ready",
		F: func(ctx context.Context) error {
			return u.waitForDeployment(ctx, common.PerconaEverestOperatorDeploymentName, common.SystemNamespace)
		},
	}
}

func (u *Upgrade) newStepEnsureEverestAPI() steps.Step {
	return steps.Step{
		Desc: "Ensuring Everest API deployment is ready",
		F: func(ctx context.Context) error {
			return u.waitForDeployment(ctx, common.PerconaEverestDeploymentName, common.SystemNamespace)
		},
	}
}

func (u *Upgrade) newStepEnsureCatalogSource() steps.Step {
	return steps.Step{
		Desc: "Ensuring Everest CatalogSource is ready",
		F: func(ctx context.Context) error {
			manifest, err := u.helmInstaller.RenderTemplates(ctx)
			if err != nil {
				return fmt.Errorf("could not render Helm templates: %w", err)
			}
			catalogNs, err := manifest.GetEverestCatalogNamespace()
			if err != nil {
				return fmt.Errorf("could not get Everest CatalogSource namespace: %w", err)
			}
			return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, false, func(ctx context.Context) (bool, error) {
				cs, err := u.kubeConnector.GetCatalogSource(ctx, types.NamespacedName{Namespace: catalogNs, Name: common.PerconaEverestCatalogName})
				if err != nil {
					return false, fmt.Errorf("cannot get CatalogSource: %w", err)
				}
				return pointer.Get(cs.Status.GRPCConnectionState).LastObservedState == "READY", nil
			})
		},
	}
}

func (u *Upgrade) waitForDeployment(ctx context.Context, name, namespace string) error {
	u.l.Infof("Waiting for Deployment '%s' in namespace '%s'", name, namespace)
	if err := u.kubeConnector.WaitForRollout(ctx, types.NamespacedName{Namespace: namespace, Name: name}); err != nil {
		return err
	}
	u.l.Infof("Deployment '%s' in namespace '%s' is ready", name, namespace)
	return nil
}

func (u *Upgrade) upgradeCustomResourceDefinitions(ctx context.Context) error {
	// Use legacy method for versions below 1.9.0.
	if goversion.Must(goversion.NewVersion(u.upgradeToVersion)).LessThan(goversion.Must(goversion.NewVersion("1.9.0"))) &&
		!version.IsDev(u.upgradeToVersion) {
		return u.legacyUpgradeCRDs(ctx)
	}
	installer := helm.Installer{
		ReleaseName:      helm.EverestCRDChartName,
		ReleaseNamespace: common.SystemNamespace,
	}
	if err := installer.Init(u.config.KubeconfigPath, helm.ChartOptions{
		URL:       u.config.RepoURL,
		Directory: utils.CRDSubChartPath(u.config.ChartDir),
		Name:      helm.EverestCRDChartName,
		Version:   u.upgradeToVersion,
	}); err != nil {
		return fmt.Errorf("could not initialize Helm installer: %w", err)
	}
	return installer.Install(ctx)
}

// legacyUpgradeCRDs upgrades the CRDs for any version below 1.9.0.
// This is kept for backward compatibility with versions below 1.9.0.
func (u *Upgrade) legacyUpgradeCRDs(ctx context.Context) error {
	manifests, err := u.helmInstaller.RenderTemplates(ctx)
	if err != nil {
		return fmt.Errorf("could not render Helm templates: %w", err)
	}
	crds, err := manifests.GetCRDs()
	if err != nil {
		return fmt.Errorf("could not get CRDs: %w", err)
	}
	return u.kubeConnector.ApplyManifestFile(ctx, helmutils.YAMLStringsToBytes(crds), common.SystemNamespace)
}

func (u *Upgrade) upgradeHelmChart(ctx context.Context) error {
	if !u.helmReleaseExists {
		// We're on the legacy installation.
		// Perform migration of the existing resources to Helm.
		return u.migrateLegacyInstallationToHelm(ctx)
	}

	// First upgrade DB namespaces.
	// This upgrade is no-op, it just updates the version metadata in helm.
	if err := u.upgradeEverestDBNamespaceHelmCharts(ctx); err != nil {
		return fmt.Errorf("could not upgrade DB namespaces Helm charts: %w", err)
	}
	// Upgrade the main chart.
	return u.helmInstaller.Upgrade(ctx, helm.UpgradeOptions{
		ReuseValues:          u.config.ReuseValues,
		ResetValues:          u.config.ResetValues,
		ResetThenReuseValues: u.config.ResetThenReuseValues,
	})
}

func (u *Upgrade) upgradeEverestDBNamespaceHelmCharts(ctx context.Context) error {
	dbNamespaces, err := u.kubeConnector.GetDBNamespaces(ctx)
	if err != nil {
		return fmt.Errorf("could not get database namespaces: %w", err)
	}
	for _, ns := range dbNamespaces.Items {
		if err := u.upgradeEverestDBNamespaceHelmChart(ctx, ns.GetName()); err != nil {
			return fmt.Errorf("could not upgrade DB namespace '%s' Helm chart: %w", ns.GetName(), err)
		}
	}
	return nil
}

func (u *Upgrade) upgradeEverestDBNamespaceHelmChart(ctx context.Context, namespace string) error {
	installer := helm.Installer{
		ReleaseName:      namespace,
		ReleaseNamespace: namespace,
	}
	if err := installer.Init(u.config.KubeconfigPath, helm.ChartOptions{
		URL:       u.config.RepoURL,
		Directory: utils.DBNamespaceSubChartPath(u.config.ChartDir),
		Name:      helm.EverestDBNamespaceChartName,
		Version:   u.upgradeToVersion,
	}); err != nil {
		return fmt.Errorf("could not initialize Helm installer: %w", err)
	}

	return installer.Upgrade(ctx, helm.UpgradeOptions{
		DisableHooks: true,
		// This will preserve old values and use any new values from the chart.
		ResetThenReuseValues: true,
	})
}

func (u *Upgrade) migrateLegacyInstallationToHelm(ctx context.Context) error {
	u.l.Info("Migrating existing installation to Helm")
	if err := u.cleanupLegacyResources(ctx); err != nil {
		return fmt.Errorf("failed to cleanupLegacyResources: %w", err)
	}
	if err := u.helmInstaller.Install(ctx); err != nil {
		return fmt.Errorf("failed to install Helm chart: %w", err)
	}
	dbNamespaces, err := u.kubeConnector.GetDBNamespaces(ctx)
	if err != nil {
		return fmt.Errorf("could not get database namespaces: %w", err)
	}
	for _, ns := range dbNamespaces.Items {
		if err := u.helmAdoptDBNamespaces(ctx, ns.GetName(), u.upgradeToVersion); err != nil {
			return fmt.Errorf("could not migrate DB namespace '%s' installation to Helm: %w", ns.GetName(), err)
		}
	}
	return nil
}

// Creates an installation of the `everest-db-namespace` Helm chart for the given DB namesapce
// and adopts its resources.
func (u *Upgrade) helmAdoptDBNamespaces(ctx context.Context, namespace, version string) error {
	dbEngines, err := u.kubeConnector.ListDatabaseEngines(ctx, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("cannot list database engines in namespace %s: %w", namespace, err)
	}
	overrides := helm.NewValues(helm.Values{
		ClusterType:        u.clusterType,
		VersionMetadataURL: u.config.VersionMetadataURL,
	})
	values := Must(helmutils.MergeVals(helmValuesForDBEngines(dbEngines), overrides))
	installer := helm.Installer{
		ReleaseName:      namespace,
		ReleaseNamespace: namespace,
		Values:           values,
	}

	if err := installer.Init(u.config.KubeconfigPath, helm.ChartOptions{
		URL:       u.config.RepoURL,
		Directory: utils.DBNamespaceSubChartPath(u.config.ChartDir),
		Name:      helm.EverestDBNamespaceChartName,
		Version:   version,
	}); err != nil {
		return fmt.Errorf("could not initialize Helm installer: %w", err)
	}
	if err := installer.Install(ctx); err != nil {
		return fmt.Errorf("could not install Helm chart: %w", err)
	}
	// This Upgrade is a no-op, howeverer, it is needed so that the existing Subscriptions and OperatorGroup in
	// this namespace are correctly adopted by the Helm release.
	// Running Install() first is needed to ensure that the release is created, but it does not adopt resources.
	return installer.Upgrade(ctx, helm.UpgradeOptions{
		DisableHooks: true,
		ReuseValues:  true,
		Force:        true, // since the version is not changing, we want to ensure that new manifests are still applied.
	})
}

func helmValuesForDBEngines(list *everestv1alpha1.DatabaseEngineList) values.Options {
	var vals []string
	for _, dbEngine := range list.Items {
		t := dbEngine.Spec.Type
		vals = append(vals, fmt.Sprintf("%s=%t", t, dbEngine.Status.State == everestv1alpha1.DBEngineStateInstalled))
	}
	vals = append(vals, "cleanupOnUninstall=false") // uninstall command will do the clean-up on its own.
	// TODO: figure out how to set telemetry.
	return values.Options{Values: vals}
}

// cleanupLegacyResources removes resources that were created as a part of the legacy installation method
// and no longer exist as a part of the Helm chart.
func (u *Upgrade) cleanupLegacyResources(ctx context.Context) error {
	// Delete OLM PackageServer CSV.
	if err := wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		delObj := &olmv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: kubernetes.OLMNamespace,
				Name:      "packageserver",
			},
		}
		if err := u.kubeConnector.DeleteClusterServiceVersion(ctx, delObj); err != nil {
			if k8serrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}); err != nil {
		return err
	}

	// Delete resources related to Everest Operator Subscription.
	if err := deleteOLMOperator(ctx, u.kubeConnector, common.EverestOperatorName, common.SystemNamespace); err != nil {
		return fmt.Errorf("could not delete operator='%s' in namespace='%s': %w",
			common.EverestOperatorName,
			common.SystemNamespace,
			err,
		)
	}
	// Delete resources related to victoria metrics operator Subscription.
	if err := deleteOLMOperator(ctx, u.kubeConnector, common.VictoriaMetricsOperatorName, common.MonitoringNamespace); err != nil {
		return fmt.Errorf("could not delete operator='%s' in namespace='%s': %w",
			common.VictoriaMetricsOperatorName,
			common.MonitoringNamespace,
			err,
		)
	}
	delDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.SystemNamespace,
			Name:      common.PerconaEverestDeploymentName,
		},
	}
	if err := u.kubeConnector.DeleteDeployment(ctx, delDep); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("could not delete deployment='%s' in namespace='%s': %w",
			common.PerconaEverestDeploymentName,
			common.SystemNamespace,
			err,
		)
	}
	// Delete Everest Catalog.
	// This is not a legacy resource, but we need to delete it so that Helm creates a new one that is owned by the release.
	delObj := &olmv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: kubernetes.OLMNamespace,
			Name:      common.PerconaEverestCatalogName,
		},
	}
	if err := u.kubeConnector.DeleteCatalogSource(ctx, delObj); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("could not delete CatalogSource='%s' in namespace='%s': %w",
			common.PerconaEverestCatalogName,
			kubernetes.OLMNamespace,
			err,
		)
	}
	return nil
}

// deleteOLMOperator deletes the subscription and current installed CSV.
func deleteOLMOperator(ctx context.Context, k kubernetes.KubernetesConnector, subscription, namespace string) error {
	currentCSV := ""
	if err := wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		sub, err := k.GetSubscription(ctx, types.NamespacedName{Name: subscription, Namespace: namespace})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		currentCSV = sub.Status.InstalledCSV
		delObj := &olmv1alpha1.Subscription{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      subscription,
			},
		}
		return false, k.DeleteSubscription(ctx, delObj)
	}); err != nil {
		return err
	}
	if currentCSV != "" {
		if err := wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
			delObj := &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      currentCSV,
				},
			}
			if err := k.DeleteClusterServiceVersion(ctx, delObj); err != nil {
				if k8serrors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			return false, nil
		}); err != nil {
			return err
		}
	}
	return nil
}
