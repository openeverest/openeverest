// everest
// Copyright (C) 2023 Percona LLC
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

// Package install ...
package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	versionpb "github.com/Percona-Lab/percona-version-service/versionpb"
	"github.com/charmbracelet/lipgloss"
	goversion "github.com/hashicorp/go-version"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/cli/helm"
	helmutils "github.com/openeverest/openeverest/v2/pkg/cli/helm/utils"
	"github.com/openeverest/openeverest/v2/pkg/cli/namespaces"
	"github.com/openeverest/openeverest/v2/pkg/cli/steps"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/output"
	. "github.com/openeverest/openeverest/v2/pkg/utils/must" //nolint:revive,stylecheck
	"github.com/openeverest/openeverest/v2/pkg/version"
	versionservice "github.com/openeverest/openeverest/v2/pkg/version_service"
)

const (
	pollInterval = 5 * time.Second
	pollTimeout  = 10 * time.Minute
)

// Installer implements the main logic for commands.
type (
	// InstallConfig holds the configuration for the `install` command.
	InstallConfig struct {
		// KubeconfigPath is the path to the kubeconfig file.
		KubeconfigPath string
		// VersionMetadataURL Version service URL to retrieve version metadata information from.
		VersionMetadataURL string
		// Version defines Everest version to be installed. If empty, the latest version is installed.
		Version string
		// DisableTelemetry disables telemetry.
		DisableTelemetry bool
		// ClusterType is the type of the Kubernetes environment.
		// If it is not set, the environment will be detected.
		ClusterType kubernetes.ClusterType
		// SkipEnvDetection skips detecting the Kubernetes environment.
		// If it is set, the environment will not be detected.
		// Set ClusterType if the environment is known and set this flag to avoid detection duplication.
		SkipEnvDetection bool
		// Pretty if set print the output in pretty mode.
		Pretty bool
		// SkipDBNamespace is set if the installation should skip provisioning database.
		SkipDBNamespace bool
		// Options related to Helm.
		HelmConfig helm.CLIOptions
		// NamespaceAddConfig is the configuration for the namespace add operation.
		NamespaceAddConfig namespaces.NamespaceAddConfig
	}

	// Installer provides the functionality to install Everest.
	Installer struct {
		l              *zap.SugaredLogger
		cfg            InstallConfig
		kubeClient     kubernetes.KubernetesConnector
		versionService versionservice.Interface
		// these are set only when Run is called.
		installVersion string
		helmInstaller  *helm.Installer
	}
)

// ------ Install Config ------

// NewInstallConfig returns a new InstallConfig.
func NewInstallConfig() InstallConfig {
	return InstallConfig{
		ClusterType:        kubernetes.ClusterTypeUnknown,
		Pretty:             true,
		NamespaceAddConfig: namespaces.NewNamespaceAddConfig(),
	}
}

// detectKubernetesEnv detects the Kubernetes environment where Everest is installed.
func (cfg *InstallConfig) detectKubernetesEnv(ctx context.Context, l *zap.SugaredLogger) error {
	if cfg.SkipEnvDetection {
		return nil
	}

	kubeClient, err := cliutils.NewKubeConnector(l, cfg.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	t, err := kubeClient.GetClusterType(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect cluster type: %w", err)
	}
	cfg.ClusterType = t
	cfg.NamespaceAddConfig.ClusterType = t

	// Skip detecting Kubernetes environment in the future.
	cfg.SkipEnvDetection = true
	cfg.NamespaceAddConfig.SkipEnvDetection = true
	l.Infof("Detected Kubernetes environment: %s", t)
	return nil
}

// ------ Installer ------

// NewInstall returns a new Installer struct.
func NewInstall(c InstallConfig, l *zap.SugaredLogger) (*Installer, error) {
	cli := &Installer{
		l: l.With("component", "install"),
	}
	if c.Pretty {
		cli.l = zap.NewNop().Sugar()
	}

	c.NamespaceAddConfig.Pretty = c.Pretty
	c.NamespaceAddConfig.HelmConfig = c.HelmConfig
	c.NamespaceAddConfig.KubeconfigPath = c.KubeconfigPath
	c.NamespaceAddConfig.DisableTelemetry = c.DisableTelemetry
	c.NamespaceAddConfig.SkipEnvDetection = c.SkipEnvDetection
	cli.cfg = c

	var err error
	cli.kubeClient, err = cliutils.NewKubeConnector(cli.l, c.KubeconfigPath)
	if err != nil {
		return nil, err
	}

	cli.versionService = versionservice.New(c.VersionMetadataURL)
	return cli, nil
}

// Run the Everest installation process.
func (o *Installer) Run(ctx context.Context) error {
	if err := o.cfg.detectKubernetesEnv(ctx, o.l); err != nil {
		return fmt.Errorf("failed to detect Kubernetes environment: %w", err)
	}

	if err := o.setVersionInfo(ctx); err != nil {
		return fmt.Errorf("failed to get Everest version info: %w", err)
	}

	if version.IsDev(o.installVersion) && o.cfg.HelmConfig.ChartDir == "" {
		// Note: n.cfg.HelmConfig.ChartDir will be rewritten inside SetupEverestDevChart
		cleanup, err := helmutils.SetupEverestDevChart(o.l, &o.cfg.HelmConfig.ChartDir)
		if err != nil {
			return err
		}
		o.cfg.NamespaceAddConfig.HelmConfig = o.cfg.HelmConfig
		defer cleanup()
	}

	if err := o.setupHelmInstaller(ctx); err != nil {
		return err
	}

	installSteps := o.newInstallSteps()
	if !o.cfg.SkipDBNamespace {
		// DB namespaces creation is required.
		if dbInstallSteps, err := o.getDBNamespacesInstallSteps(ctx); err != nil {
			return fmt.Errorf("could not create db install step: %w", err)
		} else if dbInstallSteps != nil {
			installSteps = append(installSteps, dbInstallSteps...)
		}
	}

	var out io.Writer = os.Stdout
	if !o.cfg.Pretty {
		out = io.Discard
	}

	// Run steps.
	_, _ = fmt.Fprintln(out, output.Info("Installing Everest version %s", o.installVersion))
	if err := steps.RunStepsWithSpinner(ctx, o.l, installSteps, o.cfg.Pretty); err != nil {
		return err
	}
	o.l.Infof("Everest '%s' has been successfully installed", o.installVersion)
	o.printPostInstallMessage(out)
	return nil
}

// getDBNamespacesInstallSteps returns the steps to install the database namespaces.
// It returns nil if the namespaces are already installed.
// Note: o.cfg.NamespaceAddConfig.NamespaceList and o.cfg.NamespaceAddConfig.Operators
// must be set before calling this function.
func (o *Installer) getDBNamespacesInstallSteps(ctx context.Context) ([]steps.Step, error) {
	i, err := namespaces.NewNamespaceAdd(o.cfg.NamespaceAddConfig, o.l)
	if err != nil {
		return nil, err
	}

	return i.GetNamespaceInstallSteps(ctx, o.installVersion)
}

//nolint:gochecknoglobals
var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	commandStyle = lipgloss.NewStyle().Italic(true)
)

func (o *Installer) printPostInstallMessage(out io.Writer) {
	message := "\n" + output.Rocket("Thank you for installing Everest (v%s)!\n", o.installVersion)
	message += "Follow the steps below to get started:"

	count := 1
	if len(o.cfg.NamespaceAddConfig.NamespaceList) == 0 {
		message += fmt.Sprintf("\n\n%s", output.Numeric(count, "%s", titleStyle.Render("PROVISION A NAMESPACE FOR YOUR DATABASE:")))
		count++
		message += "Install a namespace for your databases using the following command:\n\n"
		message += fmt.Sprintf("\t%s", commandStyle.Render("everestctl namespaces add [NAMESPACE]"))
	}

	message += fmt.Sprintf("\n\n%s", output.Numeric(count, "%s", titleStyle.Render("RETRIEVE THE INITIAL ADMIN PASSWORD:")))
	count++
	message += "Run the following command to get the initial admin password:\n\n"
	message += fmt.Sprintf("\t%s\n\n", commandStyle.Render("everestctl accounts initial-admin-password"))
	message += output.Warn("NOTE: The initial password is stored in plain text. For security, change it immediately using the following command:\n")
	message += fmt.Sprintf("\t%s", commandStyle.Render("everestctl accounts set-password --username admin"))

	values := o.helmInstaller.GetParsedValues()

	urlScheme := "http"
	targetPort := values.Server.Service.Port
	pfSrcPort := 8080
	svcName := values.Server.Service.Name
	if values.Server.TLS.Enabled {
		urlScheme = "https"
		pfSrcPort = 8443
		targetPort = 443
	}

	webURL := fmt.Sprintf("%s://localhost:%d", urlScheme, pfSrcPort)
	portFwdCmd := fmt.Sprintf("kubectl port-forward -n everest-system svc/%s %d:%d", svcName, pfSrcPort, targetPort)
	message += fmt.Sprintf("\n\n%s", output.Numeric(count, "%s", titleStyle.Render("ACCESS THE EVEREST UI:")))
	count++
	message += fmt.Sprintf("To access the web UI, set up port-forwarding and visit %s in your browser:\n\n", webURL)
	message += fmt.Sprintf("\t%s\n", commandStyle.Render(portFwdCmd))

	_, _ = fmt.Fprint(out, message)
}

// setVersionInfo fetches the latest Everest version information from Version service.
func (o *Installer) setVersionInfo(ctx context.Context) error {
	meta, err := o.versionService.GetEverestMetadata(ctx)
	if err != nil {
		return errors.Join(err, errors.New("could not fetch version metadata"))
	}
	latest, latestMeta, err := o.latestVersion(meta)
	if err != nil {
		return err
	}

	supVer, err := common.NewSupportedVersion(latestMeta)
	if err != nil {
		return err
	}
	if err := o.checkRequirements(supVer); err != nil {
		return err
	}

	o.l.Debugf("Everest latest version available: %s", latest)
	o.l.Debugf("Everest version information %#v", latestMeta)
	o.installVersion = latest.String()
	return nil
}

func (o *Installer) checkRequirements(supVer *common.SupportedVersion) error {
	if err := cliutils.VerifyCLIVersion(supVer); err != nil {
		return err
	}
	return nil
}

// setupHelmInstaller initializes the Helm installer.
func (o *Installer) setupHelmInstaller(ctx context.Context) error {
	nsExists, err := o.namespaceExists(ctx, common.SystemNamespace)
	if err != nil {
		return err
	}
	overrides := helm.NewValues(helm.Values{
		ClusterType:        o.cfg.ClusterType,
		VersionMetadataURL: o.cfg.VersionMetadataURL,
	})
	values := Must(helmutils.MergeVals(o.cfg.HelmConfig.Values, overrides))
	installer := &helm.Installer{
		ReleaseName:            common.SystemNamespace,
		ReleaseNamespace:       common.SystemNamespace,
		Values:                 values,
		CreateReleaseNamespace: !nsExists,
	}
	if err := installer.Init(o.cfg.KubeconfigPath, helm.ChartOptions{
		Directory: o.cfg.HelmConfig.ChartDir,
		URL:       o.cfg.HelmConfig.RepoURL,
		Name:      helm.EverestChartName,
		Version:   o.installVersion,
	}); err != nil {
		return fmt.Errorf("could not initialize Helm installer: %w", err)
	}
	o.helmInstaller = installer
	return nil
}

func (o *Installer) newInstallSteps() []steps.Step {
	return []steps.Step{
		o.newStepInstallEverestHelmChart(),
		o.newStepEnsureEverestAPI(),
		o.newStepEnsureEverestOperator(),
		o.newStepEnsureEverestOLM(),
		o.newStepEnsureCatalogSource(),
		o.newStepEnsureEverestMonitoring(),
	}
}

func (o *Installer) latestVersion(meta *versionpb.MetadataResponse) (*goversion.Version, *versionpb.MetadataVersion, error) {
	var (
		latest     *goversion.Version
		latestMeta *versionpb.MetadataVersion

		targetVersion *goversion.Version
		err           error
	)

	if o.cfg.Version != "" {
		targetVersion, err = goversion.NewSemver(o.cfg.Version)
		if err != nil {
			return nil, nil, errors.Join(err, fmt.Errorf("could not parse target version %q", o.cfg.Version))
		}
	}

	for _, v := range meta.GetVersions() {
		ver, err := goversion.NewSemver(v.GetVersion())
		if err != nil {
			o.l.Debugf("Could not parse version %s. Error: %s", v.GetVersion(), err)
			continue
		}

		if targetVersion != nil {
			if ver.Equal(targetVersion) {
				return ver, v, nil
			}
		} else {
			if latest == nil || ver.GreaterThan(latest) {
				latest = ver
				latestMeta = v
				continue
			}
		}
	}

	if latest == nil {
		return nil, nil, errors.New("could not determine the latest Everest version")
	}

	return latest, latestMeta, nil
}

func (o *Installer) namespaceExists(ctx context.Context, namespace string) (bool, error) {
	_, err := o.kubeClient.GetNamespace(ctx, types.NamespacedName{Name: namespace})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("cannot check if namesapce exists: %w", err)
	}
	return true, nil
}

// CheckEverestAlreadyinstalled checks if Everest is already installed.
func CheckEverestAlreadyinstalled(ctx context.Context, l *zap.SugaredLogger, kubeConfig string) error {
	kubeClient, err := cliutils.NewKubeConnector(l, kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	installedVersion, err := version.EverestVersionFromDeployment(ctx, kubeClient)
	if client.IgnoreNotFound(err) != nil {
		return errors.Join(err, errors.New("cannot check if Everest is already installed"))
	} else if err == nil {
		return fmt.Errorf("everest is already installed. Version: %s", installedVersion)
	}

	return nil
}
