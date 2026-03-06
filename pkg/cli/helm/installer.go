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

// Package helm has the logic to install and uninstall Helm charts.
package helm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	helmcli "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/percona/everest/pkg/cli/helm/utils"
)

// Everest Helm chart names.
const (
	// DefaultHelmRepoURL is the default Helm repository URL to download the Everest charts.
	DefaultHelmRepoURL = "https://openeverest.github.io/helm-charts/"
	// EverestChartName is the name of the Everest Helm chart that installs the Everest operator.
	EverestChartName = "openeverest"
	// EverestDBNamespaceChartName is the name of the Everest Helm chart that is installed
	// into DB namespaces managed by Everest.
	EverestDBNamespaceChartName = "everest-db-namespace"
	// EverestCRDChartName is the name of the Everest Helm chart that installs the Everest CRDs.
	EverestCRDChartName = "everest-crds"
)

var settings = helmcli.New() //nolint:gochecknoglobals

// CLIOptions contains common options for the CLI.
type (
	CLIOptions struct {
		// ChartDir path to the local directory with the Helm chart to be installed.
		ChartDir string
		// RepoURL URL of the Helm repository to download the chart from.
		RepoURL string
		// Values Helm values to be used during installation.
		Values values.Options
		// Devel indicates whether to use development versions of Helm charts, if available.
		Devel bool
		// ReuseValues indicates whether to reuse the last release's values during release upgrade.
		ReuseValues bool
		// ResetValues indicates whether to reset the last release's values during release upgrade.
		ResetValues bool
		// ResetThenReuseValues indicates whether to reset the last release's values then reuse them during release upgrade.
		ResetThenReuseValues bool
	}

	// Installer installs a Helm chart.
	Installer struct {
		// ReleaseName is the name of the Helm release.
		ReleaseName string
		// ReleaseNamespace is the namespace where the Helm release will be installed.
		ReleaseNamespace string
		// Values are the Helm values to be used during installation.
		Values map[string]interface{}
		// CreateReleaseNamespace indicates whether to create the release namespace.
		CreateReleaseNamespace bool
		// internal fields, set only after Init() is called.
		chart *chart.Chart
		cfg   *action.Configuration
		// This is set only after Install/Upgrade is called.
		release *release.Release

		// contains the complete set of values used to render the chart.
		// This is set only after Init is called.
		allParsedValues *utils.Values
	}

	// ChartOptions provide the options for loading a Helm chart.
	ChartOptions struct {
		// Directory to load the Helm chart from.
		// If set, ignores URL.
		Directory string
		// URL of the repository to pull the chart from.
		URL string
		// Version of the helm chart to install.
		// If loading from a directory, needs to match the chart version.
		Version string
		// Name of the Helm chart to install.
		// Required only if pulling from the specified URL.
		Name string
	}
)

// Init initializes the Installer with the specified options.
func (i *Installer) Init(kubeconfigPath string, o ChartOptions) error {
	if o.Directory == "" && o.URL == "" {
		return errors.New("either chart directory or URL must be set")
	}

	if o.Version == "" {
		return errors.New("chart version must be set")
	}

	chart, err := resolveHelmChart(o.Version, o.Name, o.URL, o.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve Helm chart: %w", err)
	}
	i.chart = chart

	defaultVals := chart.Values
	userVals := i.Values
	// copy userVals into a copy of defaultVals
	merged := utils.MergeMaps(defaultVals, userVals)

	parsedVals, err := utils.ParseValues(merged)
	if err != nil {
		return fmt.Errorf("failed to parse Helm chart values: %w", err)
	}
	i.allParsedValues = parsedVals

	cfg, err := newActionsCfg(i.ReleaseNamespace, kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create Helm action configuration: %w", err)
	}
	i.cfg = cfg
	return nil
}

// GetParsedValues returns the values parsed into a struct with the known values.
func (i *Installer) GetParsedValues() *utils.Values {
	return i.allParsedValues
}

// RenderTemplates renders the Helm templates from the provided chart.
// If the chart has been installed, it returns the rendered templates from the installed release.
// If the chart has not been installed, it returns the rendered templates from a dry-run install.
// In the latter case, the installation step does not talk to the kube-apiserver. So Helm functions like `lookup` will not work.
func (i *Installer) RenderTemplates(ctx context.Context) (RenderedTemplate, error) {
	if i.release != nil {
		return newRenderedTemplate(i.release.Manifest), nil
	}

	// create a new actions configuration so that it does not accidentally interfere with the actual installation.
	cfg, err := newActionsCfg(i.ReleaseNamespace, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create Helm action configuration: %w", err)
	}

	rel, err := installDryRun(ctx, cfg, i.chart, i.ReleaseName, i.ReleaseNamespace, i.Values)
	if err != nil {
		return nil, err
	}
	return newRenderedTemplate(rel.Manifest), nil
}

func installDryRun(
	ctx context.Context,
	cfg *action.Configuration,
	chart *chart.Chart,
	releaseName, releaseNamespace string,
	values map[string]interface{},
) (*release.Release, error) {
	install := action.NewInstall(cfg)
	install.ReleaseName = releaseName
	install.Namespace = releaseNamespace
	install.CreateNamespace = true
	install.Wait = false
	install.DisableHooks = true
	install.DryRun = true
	install.ClientOnly = true
	install.Replace = true
	install.IncludeCRDs = true

	parsedKubeVersion, err := chartutil.ParseKubeVersion("1.30.0")
	if err != nil {
		return nil, fmt.Errorf("failed to parse kube version: %w", err)
	}
	install.KubeVersion = parsedKubeVersion
	return install.RunWithContext(ctx, chart, values)
}

// Install the Helm chart.
// Calling Install multiple times is idempotent; it will re-apply the manifests using upgrade.
func (i *Installer) Install(ctx context.Context) error {
	rel, err := action.NewGet(i.cfg).Run(i.ReleaseName)
	if err != nil {
		// Release does not exist, install it.
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return i.install(ctx)
		}
		return err
	}
	// If the release already exists, we will re-apply the manifests using upgrade.
	// We're not actually upgrading to a new version, but using upgrade to re-apply manifests.
	// This is how Helm expects us to re-apply manifests.
	// To prevent accidental version upgrades, we will explicitly check that the resolved chart version matches the installed chart version.
	if i.chart.Metadata.Version != rel.Chart.Metadata.Version {
		return fmt.Errorf("cannot overwrite existing release with a different chart version. Expected %s, got %s",
			rel.Chart.Metadata.Version, i.chart.Metadata.Version,
		)
	}
	return i.Upgrade(ctx, UpgradeOptions{})
}

// GetRelease gets the installed Helm release.
func (i *Installer) GetRelease() (*release.Release, error) {
	if i.release == nil {
		return nil, errors.New("chart not installed")
	}
	return i.release, nil
}

func (i *Installer) install(ctx context.Context) error {
	install := action.NewInstall(i.cfg)
	install.ReleaseName = i.ReleaseName
	install.Namespace = i.ReleaseNamespace
	install.CreateNamespace = i.CreateReleaseNamespace
	install.TakeOwnership = true

	rel, err := install.RunWithContext(ctx, i.chart, i.Values)
	if err != nil {
		return err
	}
	i.release = rel
	return nil
}

// UpgradeOptions provide options for upgrading a Helm chart.
type UpgradeOptions struct {
	DisableHooks         bool
	ReuseValues          bool
	ResetValues          bool
	ResetThenReuseValues bool
	Force                bool
}

// Upgrade the Helm chart.
func (i *Installer) Upgrade(ctx context.Context, opts UpgradeOptions) error {
	upgrade := action.NewUpgrade(i.cfg)
	upgrade.Namespace = i.ReleaseNamespace
	upgrade.TakeOwnership = true
	upgrade.ReuseValues = opts.ReuseValues
	upgrade.ResetValues = opts.ResetValues
	upgrade.ResetThenReuseValues = opts.ResetThenReuseValues
	upgrade.DisableHooks = opts.DisableHooks
	upgrade.Force = opts.Force

	rel, err := upgrade.RunWithContext(ctx, i.ReleaseName, i.chart, i.Values)
	if err != nil {
		return err
	}
	i.release = rel
	return nil
}

func resolveHelmChart(version, chartName, repoURL, dir string) (*chart.Chart, error) {
	if dir != "" {
		return resolveDir(version, dir)
	}
	return resolveRepo(version, chartName, repoURL)
}

func resolveDir(version, dir string) (*chart.Chart, error) {
	if err := buildChartDeps(dir); err != nil {
		return nil, err
	}
	chart, err := loader.LoadDir(dir)
	if err != nil {
		return nil, err
	}
	// When loading from a directory, ensure that the loaded chart version
	// matches the specified version.
	if chart.Metadata.Version != version {
		return nil, fmt.Errorf("chart version does not match specified version."+
			"Expected chart version %s, got %s", version, chart.Metadata.Version,
		)
	}
	return chart, nil
}

func resolveRepo(version, chartName, repoURL string) (*chart.Chart, error) {
	// Download Helm chart from repo and cache it for later use.
	chart, err := newChartFromRemoteWithCache(version, chartName, repoURL)
	if err != nil {
		return nil, err
	}
	return chart, nil
}

// newChartFromRemoteWithCache downloads the chart from the remote repository and caches it.
func newChartFromRemoteWithCache(version, name string, repository string) (*chart.Chart, error) {
	cacheDir, err := everestctlCacheDir()
	if err != nil {
		return nil, err
	}

	file := path.Join(cacheDir, fmt.Sprintf("%s-%s.tgz", name, version))
	if _, err = os.Stat(file); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}

		// Download the chart from remote repository
		actionConfig := &action.Configuration{}
		pull := action.NewPullWithOpts(action.WithConfig(actionConfig))
		pull.Settings = settings
		pull.Version = version
		pull.DestDir = cacheDir
		pull.RepoURL = repository
		if _, err = pull.Run(name); err != nil {
			return nil, err
		}
	}

	f, err := os.Open(file) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck
	return loader.LoadArchive(f)
}

func everestctlCacheDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	res := path.Join(cacheDir, "everestctl")
	err = os.MkdirAll(res, 0o755) //nolint:gosec,mnd
	if err != nil && !os.IsExist(err) {
		return "", err
	}
	return res, nil
}

// Runs `helm dependency update` in the chart directory.
func buildChartDeps(chartDir string) error {
	man := &downloader.Manager{
		Out:              io.Discard,
		ChartPath:        chartDir,
		Getters:          getter.All(settings),
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
	}
	return man.Update()
}

func newActionsCfg(namespace, kubeconfig string) (*action.Configuration, error) {
	logger := func(_ string, _ ...interface{}) {}
	cfg := action.Configuration{}
	restClientGetter := genericclioptions.ConfigFlags{
		Namespace: &namespace,
	}
	if kubeconfig != "" {
		home := os.Getenv("HOME")
		kubeconfig = strings.ReplaceAll(kubeconfig, "~", home)
		restClientGetter.KubeConfig = &kubeconfig
	}
	if err := cfg.Init(&restClientGetter, namespace, "", logger); err != nil {
		return nil, err
	}
	cfg.Releases.MaxHistory = 3
	return &cfg, nil
}

// Uninstaller uninstalls a Helm release.
type Uninstaller struct {
	ReleaseName      string
	ReleaseNamespace string
	actionCfg        *action.Configuration
}

// NewUninstaller creates a new Uninstaller.
func NewUninstaller(relName, relNamespace, kubeconfigPath string) (*Uninstaller, error) {
	cfg, err := newActionsCfg(relNamespace, kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return &Uninstaller{
		ReleaseName:      relName,
		ReleaseNamespace: relNamespace,
		actionCfg:        cfg,
	}, nil
}

// Uninstall a Helm release.
// Returns true if a release was found.
// If dryRun is set, returns true if a release exists, but doesn't actually uninstall it.
func (u *Uninstaller) Uninstall(dryRun bool) (bool, error) {
	uninstall := action.NewUninstall(u.actionCfg)
	uninstall.Wait = false
	uninstall.IgnoreNotFound = true
	uninstall.DryRun = dryRun

	resp, err := uninstall.Run(u.ReleaseName)
	if err != nil {
		if dryRun && errors.Is(err, driver.ErrReleaseNotFound) {
			return false, nil
		}
		return false, err
	}
	return resp != nil && resp.Release != nil, nil
}
