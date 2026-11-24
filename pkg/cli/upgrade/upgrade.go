// everest
// Copyright (C) 2023 Percona LLC
// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package upgrade implements upgrade logic for the CLI.
package upgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	version "github.com/Percona-Lab/percona-version-service/versionpb"
	goversion "github.com/hashicorp/go-version"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/pkg/cli/helm"
	helmutils "github.com/openeverest/openeverest/v2/pkg/cli/helm/utils"
	"github.com/openeverest/openeverest/v2/pkg/cli/steps"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/output"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
	. "github.com/openeverest/openeverest/v2/pkg/utils/must" //nolint:revive,staticcheck
	cliVersion "github.com/openeverest/openeverest/v2/pkg/version"
	versionservice "github.com/openeverest/openeverest/v2/pkg/version_service"
)

const (
	pollInterval = 5 * time.Second
	pollTimeout  = 10 * time.Minute

	legacyMonitoringComponent     = "monitoring"
	legacyMonitoringComponentType = "pmm"
	legacyMonitoringRBACResource  = "monitoring-configs"

	// coreVersionCRD is the core CRD whose openeverest.io/version label records the installed release.
	coreVersionCRD = "providers.core.openeverest.io"
)

// inClusterConfig is read from environment variables when the upgrade
// runs as a Helm pre-upgrade hook Job. The Job needs to know which
// namespace OpenEverest is installed in to build a Kubernetes client.
// The Helm chart injects NAMESPACE via the Kubernetes downward API.
type inClusterConfig struct {
	Namespace string `envconfig:"NAMESPACE" required:"true"`
}

type (
	// Config defines configuration required for upgrade command.
	Config struct {
		helm.CLIOptions

		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string
		// InCluster is set if the upgrade process should use in-cluster configuration.
		InCluster bool
		// VersionMetadataURL stores hostname to retrieve version metadata information from.
		VersionMetadataURL string
		// DryRun is set if the upgrade process should only perform pre-upgrade checks and not perform the actual upgrade.
		DryRun bool
		// If set, we will print the pretty output.
		Pretty bool
		// SkipEnvDetection skips detecting the Kubernetes environment.
		SkipEnvDetection bool
		// DisableTelemetry disables telemetry.
		DisableTelemetry bool

		// VersionToUpgrade specifies the version to upgrade to.
		// This version may be ahead by at most one minor version from the current version.
		VersionToUpgrade string
	}

	// Upgrade struct implements upgrade command.
	Upgrade struct {
		l *zap.SugaredLogger

		config         *Config
		kubeConnector  kubernetes.KubernetesConnector
		versionService versionservice.Interface

		// these are set on calling Run
		clusterType       kubernetes.ClusterType
		helmReleaseExists bool
		upgradeToVersion  string
		helmInstaller     *helm.Installer
	}

	// upgradeChart is what the in-place upgrade check reads from Helm: the chart of the target
	// release and the chart version of the deployed one.
	upgradeChart interface {
		ChartMetadata() *chart.Metadata
		DeployedChartVersion() (string, error)
	}
)

// ErrNoUpdateAvailable is returned when no update is available.
var (
	ErrNoUpdateAvailable                      = errors.New("no update available")
	ErrDowngradeNotAllowed                    = errors.New("downgrade not allowed")
	ErrCannotUpgradeByMoreThanOneMinorVersion = errors.New("cannot upgrade by more than one minor version")
	ErrLegacyMonitoringInUse                  = errors.New("legacy monitoring is still in use")
	ErrFreshInstallRequired                   = errors.New("requires a fresh install")
)

// NewUpgrade returns a new Upgrade struct.
func NewUpgrade(cfg *Config, l *zap.SugaredLogger) (*Upgrade, error) {
	cli := &Upgrade{
		config: cfg,
		l:      l.With("component", "upgrade"),
	}
	if cfg.Pretty {
		cli.l = zap.NewNop().Sugar()
	}

	var kubeClient kubernetes.KubernetesConnector
	if cfg.InCluster {
		var ic inClusterConfig
		if err := envconfig.Process("", &ic); err != nil {
			return nil, fmt.Errorf("could not read in-cluster config: %w", err)
		}
		k, err := kubernetes.NewInCluster(context.Background(), cli.l, nil, ic.Namespace)
		if err != nil {
			return nil, fmt.Errorf("could not create in-cluster kubernetes client: %w", err)
		}
		kubeClient = k
	}
	if cfg.KubeconfigPath != "" {
		k, err := cliutils.NewKubeConnector(cli.l, cfg.KubeconfigPath)
		if err != nil {
			return nil, err
		}
		kubeClient = k
	}
	if kubeClient == nil {
		return nil, errors.New("must provide kubeconfig path or run in-cluster")
	}

	cli.kubeConnector = kubeClient
	cli.versionService = versionservice.New(cfg.VersionMetadataURL)
	return cli, nil
}

// Run runs the operators installation process.
func (u *Upgrade) Run(ctx context.Context) error {
	everestVersion, err := cliVersion.EverestVersionFromDeployment(ctx, u.kubeConnector, u.kubeConnector.Namespace())
	if err != nil {
		return errors.Join(err, errors.New("could not retrieve Everest version"))
	}

	var out io.Writer = os.Stdout
	if !u.config.Pretty {
		out = io.Discard
	}

	meta, err := u.setVersionInfo(ctx, everestVersion)
	if err != nil {
		if errors.Is(err, ErrNoUpdateAvailable) {
			u.l.Info("You're running the latest version of Everest")
			_, _ = fmt.Fprintln(out, "\n", output.Rocket("You're running the latest version of Everest"))
			return nil
		}
		return err
	}

	if err := u.setKubernetesEnv(ctx); err != nil {
		return fmt.Errorf("could not detect Kubernetes environment: %w", err)
	}

	// The chart of the target release declares which releases it upgrades from in place,
	// so it is resolved before the checks run, dry run included.
	if err := u.setupHelmInstaller(ctx); err != nil {
		return fmt.Errorf("could not initialize Helm installer: %w", err)
	}

	if err := u.verifyRequirements(ctx, meta); err != nil {
		return err
	}

	if u.config.DryRun {
		return nil
	}

	u.l.Infof("Upgrading Everest to %s in namespace %s", u.upgradeToVersion, u.kubeConnector.Namespace())

	// Helm based installation was added to the CLI in 1.4.0. Versions below that are not managed by helm.
	// We use this flag to trigger an adoption of the existing installation to Helm chart.
	u.helmReleaseExists = common.CheckConstraint(everestVersion, ">= 1.4.0")

	upgradeSteps := u.newUpgradeSteps()

	// Run steps.
	_, _ = fmt.Fprintln(out, output.Info("Upgrading Everest to version %s", u.upgradeToVersion))
	if err := steps.RunStepsWithSpinner(ctx, u.l, upgradeSteps, u.config.Pretty); err != nil {
		return err
	}

	u.l.Infof("Everest has been upgraded to version %s", u.upgradeToVersion)
	return u.printPostUpgradeMessage(ctx, out)
}

// setVersionInfo picks the version to upgrade to and returns its version service metadata.
func (u *Upgrade) setVersionInfo(ctx context.Context, everestVersion *goversion.Version) (*version.MetadataVersion, error) {
	u.l.Infof("Current Everest version is %s", everestVersion)

	upgradeEverestTo, meta, err := u.versionToUpgradeTo(ctx, everestVersion)
	if err != nil {
		return nil, err
	}

	u.l.Infof("Found available upgrade to Everest version %s", upgradeEverestTo)
	u.upgradeToVersion = upgradeEverestTo.String()
	return meta, nil
}

func (u *Upgrade) setKubernetesEnv(ctx context.Context) error {
	if u.config.SkipEnvDetection {
		return nil
	}
	t, err := u.kubeConnector.GetClusterType(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect cluster type: %w", err)
	}
	u.clusterType = t
	return nil
}

func (u *Upgrade) setupHelmInstaller(_ context.Context) error {
	overrides := helm.NewValues(helm.Values{
		ClusterType:        u.clusterType,
		VersionMetadataURL: u.config.VersionMetadataURL,
		DisableTelemetry:   u.config.DisableTelemetry,
	})

	values := Must(helmutils.MergeVals(u.config.Values, overrides))
	installer := &helm.Installer{
		ReleaseName:      u.kubeConnector.Namespace(),
		ReleaseNamespace: u.kubeConnector.Namespace(),
		Values:           values,
	}
	if err := installer.Init(u.config.KubeconfigPath, helm.ChartOptions{
		URL:       u.config.RepoURL,
		Directory: u.config.ChartDir,
		Name:      helm.EverestChartName,
		Version:   u.upgradeToVersion,
	}); err != nil {
		return fmt.Errorf("could not initialize Helm installer: %w", err)
	}
	u.helmInstaller = installer
	return nil
}

func (u *Upgrade) printPostUpgradeMessage(ctx context.Context, out io.Writer) error {
	_, _ = fmt.Fprintln(out, "\n", output.Rocket("Everest has been upgraded to version %s", u.upgradeToVersion))
	if isSecure, err := u.kubeConnector.Accounts().IsSecure(ctx, common.EverestAdminUser); err != nil {
		return fmt.Errorf("could not check if the admin password is secure: %w", err)
	} else if !isSecure {
		_, _ = fmt.Fprint(os.Stdout, "\n", common.InitialPasswordWarningMessage, "\n")
	}
	return nil
}

func (u *Upgrade) newUpgradeSteps() []steps.Step {
	return []steps.Step{
		u.newStepUpgradeCRDs(),
		u.newStepUpgradeHelmChart(),
		u.newStepEnsureEverestAPI(),
		u.newStepEnsureEverestOperator(),
		u.newStepEnsureCatalogSource(),
	}
}

func validateVersionToUpgrade(
	currentEverestVersion, targetEverestVersion *goversion.Version,
) error {
	// Downgrade is not allowed.
	if targetEverestVersion.LessThan(currentEverestVersion) {
		return ErrDowngradeNotAllowed
	}
	// No upgrade is needed.
	if targetEverestVersion.Equal(currentEverestVersion) {
		return ErrNoUpdateAvailable
	}
	// Cannot upgrade by more than one minor version.
	currentMinor := currentEverestVersion.Segments()[1]
	targetMinor := targetEverestVersion.Segments()[1]
	if targetMinor-currentMinor > 1 {
		return ErrCannotUpgradeByMoreThanOneMinorVersion
	}
	return nil
}

func (u *Upgrade) handleSpecifiedVersion(
	currentEverestVersion *goversion.Version,
	req *version.MetadataResponse,
) (*goversion.Version, *version.MetadataVersion, error) {
	// Parse the specified version.
	upgradeTo, err := goversion.NewVersion(u.config.VersionToUpgrade)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse version %s: %w", u.config.VersionToUpgrade, err)
	}

	// Validate the specified version.
	if err := validateVersionToUpgrade(currentEverestVersion, upgradeTo); err != nil {
		return nil, nil, err
	}

	// Find the version metadata.
	idx := slices.IndexFunc(req.GetVersions(), func(v *version.MetadataVersion) bool {
		ver, err := goversion.NewVersion(v.GetVersion())
		if err != nil {
			u.l.Debugf("Could not parse version %s. Error: %s", v.GetVersion(), err)
			return false
		}
		return ver.Equal(upgradeTo)
	})
	if idx == -1 {
		return nil, nil, fmt.Errorf("version '%s' not found in metadata", upgradeTo)
	}
	return upgradeTo, req.GetVersions()[idx], nil
}

// versionToUpgradeTo returns version to which the current Everest version can be upgraded to.
func (u *Upgrade) versionToUpgradeTo(
	ctx context.Context, currentEverestVersion *goversion.Version,
) (*goversion.Version, *version.MetadataVersion, error) {
	req, err := u.versionService.GetEverestMetadata(ctx)
	if err != nil {
		return nil, nil, err
	}

	// User specified version, handle it and return early.
	if u.config.VersionToUpgrade != "" {
		return u.handleSpecifiedVersion(currentEverestVersion, req)
	}

	upgradeTo, meta := u.findNextMinorVersion(req, currentEverestVersion)
	if upgradeTo == nil {
		upgradeTo = currentEverestVersion
	}

	// Find the latest patch version for the given minor version.
	for _, v := range req.GetVersions() {
		ver, err := goversion.NewVersion(v.GetVersion())
		if err != nil {
			u.l.Debugf("Could not parse version %s. Error: %s", v.GetVersion(), err)
			continue
		}

		if currentEverestVersion.GreaterThanOrEqual(ver) {
			continue
		}

		// Select the latest patch version for the same major and minor version.
		verSeg := ver.Segments()
		uSeg := upgradeTo.Segments()
		if len(verSeg) >= 3 && len(uSeg) >= 3 && verSeg[0] == uSeg[0] && verSeg[1] == uSeg[1] {
			if verSeg[2] <= uSeg[2] {
				continue
			}
			upgradeTo = ver
			meta = v
			continue
		}
	}

	if upgradeTo == nil || meta == nil {
		return nil, nil, ErrNoUpdateAvailable
	}

	return upgradeTo, meta, nil
}

func (u *Upgrade) findNextMinorVersion(
	req *version.MetadataResponse, currentEverestVersion *goversion.Version,
) (*goversion.Version, *version.MetadataVersion) {
	var (
		upgradeTo *goversion.Version
		meta      *version.MetadataVersion
	)

	for _, v := range req.GetVersions() {
		ver, err := goversion.NewVersion(v.GetVersion())
		if err != nil {
			u.l.Debugf("Could not parse version %s. Error: %s", v.GetVersion(), err)
			continue
		}

		if currentEverestVersion.GreaterThanOrEqual(ver) {
			continue
		}

		verSeg := ver.Segments()
		evSeg := currentEverestVersion.Segments()
		if len(verSeg) >= 3 && len(evSeg) >= 3 && verSeg[0] == evSeg[0] && verSeg[1] == evSeg[1] {
			continue
		}

		if upgradeTo == nil {
			upgradeTo = ver
			meta = v
			continue
		}

		if upgradeTo.GreaterThan(ver) {
			upgradeTo = ver
			meta = v
			continue
		}
	}

	return upgradeTo, meta
}

// verifyRequirements refuses the upgrade to u.upgradeToVersion when the cluster or this CLI does not meet
// its requirements. It needs the chart of the target release, resolved by setupHelmInstaller.
func (u *Upgrade) verifyRequirements(ctx context.Context, meta *version.MetadataVersion) error {
	u.l.Infof("Checking requirements for upgrade to Everest %s", u.upgradeToVersion)

	// First: when the target release can only be installed fresh, no other requirement matters.
	if err := u.checkInPlaceUpgrade(ctx, u.helmInstaller); err != nil {
		return err
	}

	supVer, err := common.NewSupportedVersion(meta)
	if err != nil {
		return err
	}
	if err := cliutils.VerifyCLIVersion(supVer); err != nil {
		return err
	}

	return u.checkLegacyMonitoring(ctx)
}

// checkInPlaceUpgrade refuses the upgrade unless the chart of the target release lists the installed release
// as one it upgrades from in place.
func (u *Upgrade) checkInPlaceUpgrade(ctx context.Context, c upgradeChart) error {
	installed, err := u.installedCoreVersion(ctx, c)
	if err != nil {
		return err
	}
	return checkUpgradableFrom(u.upgradeToVersion, installed, c.ChartMetadata().Annotations)
}

// installedCoreVersion returns the installed release: the openeverest.io/version label of the core CRDs, which
// is there whatever installed them (everestctl, Helm under any release name, a GitOps tool that only renders
// the chart). The chart version of the deployed Helm release stands in for CRDs that predate the label.
func (u *Upgrade) installedCoreVersion(ctx context.Context, c upgradeChart) (string, error) {
	crd, err := u.kubeConnector.GetCRD(ctx, coreVersionCRD)
	switch {
	case err == nil:
		if v := crd.GetLabels()[contract.CoreVersionKey]; v != "" {
			return v, nil
		}
	case !k8serrors.IsNotFound(err):
		return "", fmt.Errorf("could not get CRD %s: %w", coreVersionCRD, err)
	}

	v, err := c.DeployedChartVersion()
	if err != nil {
		return "", fmt.Errorf(
			"could not tell which OpenEverest release is installed: the core CRDs carry no %s label and the deployed Helm release cannot be read: %w",
			contract.CoreVersionKey, err,
		)
	}
	return v, nil
}

// checkUpgradableFrom refuses an in-place upgrade from the installed release to target unless the annotations
// of the target chart list the installed release in openeverest.io/upgradable-from. An empty list means target
// can only be installed fresh; a chart without the annotation predates in-place upgrades, and every release
// before them required a fresh install.
func checkUpgradableFrom(target, installed string, chartAnnotations map[string]string) error {
	hint := fmt.Sprintf(
		"To move to %s, uninstall OpenEverest with \"everestctl uninstall\", which deletes every instance "+
			"and its data, then install %s",
		target, target,
	)

	upgradableFrom, declared := chartAnnotations[contract.UpgradableFromAnnotation]
	if !declared {
		return fmt.Errorf(
			"OpenEverest %s %w; upgrading from %s is not supported: its chart predates in-place upgrades "+
				"(it has no %s annotation), and every release before them required a fresh install.\n%s",
			target, ErrFreshInstallRequired, installed, contract.UpgradableFromAnnotation, hint,
		)
	}

	var releases []string
	for r := range strings.SplitSeq(upgradableFrom, ",") {
		if r = strings.TrimSpace(r); r != "" {
			releases = append(releases, r)
		}
	}
	if slices.Contains(releases, installed) {
		return nil
	}

	var only string
	if len(releases) > 0 {
		only = fmt.Sprintf(" (it upgrades in place from %s only)", strings.Join(releases, ", "))
	}
	return fmt.Errorf(
		"OpenEverest %s %w; upgrading from %s is not supported%s.\n%s",
		target, ErrFreshInstallRequired, installed, only, hint,
	)
}

// checkLegacyMonitoring refuses the upgrade while MonitoringConfigs, Instances with a "pmm"
// monitoring component or RBAC policy lines granting "monitoring-configs" exist, because the
// new version serves none of them.
func (u *Upgrade) checkLegacyMonitoring(ctx context.Context) error {
	configs, err := u.kubeConnector.ListLegacyMonitoringConfigs(ctx)
	if err != nil {
		return fmt.Errorf("could not list MonitoringConfigs: %w", err)
	}
	instances, err := u.kubeConnector.ListInstances(ctx)
	if err != nil {
		return fmt.Errorf("could not list instances: %w", err)
	}
	rbacKey := types.NamespacedName{Namespace: u.kubeConnector.Namespace(), Name: common.EverestRBACConfigMapName}
	policyLines, err := u.legacyMonitoringPolicyLines(ctx, rbacKey)
	if err != nil {
		return err
	}

	var inUse, commands []string
	for _, in := range instances.Items {
		if in.Spec.Components[legacyMonitoringComponent].Type != legacyMonitoringComponentType {
			continue
		}
		inUse = append(inUse, fmt.Sprintf(
			"Instance %s/%s (component %q of type %q)",
			in.GetNamespace(), in.GetName(), legacyMonitoringComponent, legacyMonitoringComponentType,
		))
		commands = append(commands, fmt.Sprintf(
			`kubectl patch instance -n %s %s --type=json -p '[{"op":"remove","path":"/spec/components/%s"}]'`,
			in.GetNamespace(), in.GetName(), legacyMonitoringComponent,
		))
	}
	for _, c := range configs.Items {
		inUse = append(inUse, fmt.Sprintf("MonitoringConfig %s/%s", c.GetNamespace(), c.GetName()))
		commands = append(commands, fmt.Sprintf("kubectl delete monitoringconfig -n %s %s", c.GetNamespace(), c.GetName()))
	}
	for _, line := range policyLines {
		inUse = append(inUse, fmt.Sprintf("ConfigMap %s policy line %q (use %q instead)", rbacKey, line, rbac.ResourceMonitoringDestinations))
	}
	if len(policyLines) > 0 {
		commands = append(commands, fmt.Sprintf(
			"kubectl edit configmap -n %s %s  # in policy.csv, replace %q with %q",
			rbacKey.Namespace, rbacKey.Name, legacyMonitoringRBACResource, rbac.ResourceMonitoringDestinations,
		))
	}
	if len(inUse) == 0 {
		return nil
	}
	return fmt.Errorf("%w: the new version removes MonitoringConfig, the %q Instance component of type %q "+
		"and the %q RBAC resource, and the cluster still has:\n"+
		"  - %s\n"+
		"To upgrade:\n"+
		"  1. Before the upgrade, run:\n"+
		"       %s\n"+
		"  2. After the upgrade, install a monitoring extension (monitoring-pmm, monitoring-prometheus or "+
		"monitoring-victoriametrics), create a MonitoringDestination of its MonitoringClass and reference it "+
		"from spec.monitoring.destinations[] of each Instance to monitor",
		ErrLegacyMonitoringInUse, legacyMonitoringComponent, legacyMonitoringComponentType, legacyMonitoringRBACResource,
		strings.Join(inUse, "\n  - "), strings.Join(commands, "\n       "),
	)
}

// legacyMonitoringPolicyLines returns the lines of the RBAC policy that grant "monitoring-configs".
func (u *Upgrade) legacyMonitoringPolicyLines(ctx context.Context, key types.NamespacedName) ([]string, error) {
	cm, err := u.kubeConnector.GetConfigMap(ctx, key)
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not get ConfigMap %s: %w", key, err)
	}
	var lines []string
	for line := range strings.Lines(cm.Data["policy.csv"]) {
		line = strings.TrimSpace(line)
		fields := strings.Split(line, ",")
		if len(fields) >= 3 && strings.TrimSpace(fields[0]) == "p" && strings.TrimSpace(fields[2]) == legacyMonitoringRBACResource {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
