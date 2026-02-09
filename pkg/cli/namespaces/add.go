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

// Package namespaces provides the functionality to manage namespaces.
package namespaces

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/cli/values"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/cli/helm"
	helmutils "github.com/openeverest/openeverest/v2/pkg/cli/helm/utils"
	"github.com/openeverest/openeverest/v2/pkg/cli/steps"
	"github.com/openeverest/openeverest/v2/pkg/cli/tui"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/output"
	. "github.com/openeverest/openeverest/v2/pkg/utils/must" //nolint:revive,stylecheck
	"github.com/openeverest/openeverest/v2/pkg/version"
)

type (
	// OperatorConfig identifies which operators shall be installed.
	OperatorConfig struct {
		PG    bool // is set if PostgresSQL shall be installed.
		PSMDB bool // is set if MongoDB shall be installed.
		PXC   bool // is set if XtraDB Cluster shall be installed.
	}

	// NamespaceAddConfig is the configuration for adding namespaces.
	NamespaceAddConfig struct {
		// NamespaceList is a list of namespaces to be managed by Everest and install operators.
		// The property shall be set in case the namespaces are parsed and validated using ValidateNamespaces func.
		// Otherwise, use Populate function to asked user to provide the namespaces in interactive mode.
		NamespaceList []string
		// SkipWizard is set if the wizard should be skipped.
		SkipWizard bool
		// KubeconfigPath is the path to the kubeconfig file.
		KubeconfigPath string
		// DisableTelemetry is set if telemetry should be disabled.
		DisableTelemetry bool
		// TakeOwnership make an existing namespace managed by Everest.
		TakeOwnership bool
		// ClusterType is the type of the Kubernetes environment.
		// If it is not set, the environment will be detected.
		ClusterType kubernetes.ClusterType
		// SkipEnvDetection skips detecting the Kubernetes environment.
		// If it is set, the environment will not be detected.
		// Set ClusterType if the environment is known and set this flag to avoid detection duplication.
		SkipEnvDetection bool
		// AskOperators is set in case it is needed to use interactive mode and
		// ask user to provide DB operators to be installed into namespaces managed by Everest.
		// AskOperators bool
		// Operators configurations for the operators to be installed into namespaces managed by Everest.
		Operators OperatorConfig
		// Pretty if set print the output in pretty mode.
		Pretty bool
		// Update is set if the existing namespace needs to be updated.
		// This flag is set internally only, so that the add functionality may
		// be re-used for updating the namespace as well.
		Update bool
		// Helm related options
		HelmConfig helm.CLIOptions
	}

	// NamespaceAdder provides the functionality to add namespaces.
	NamespaceAdder struct {
		l          *zap.SugaredLogger
		cfg        NamespaceAddConfig
		kubeClient kubernetes.KubernetesConnector
	}
)

// --- NamespaceAddConfig functions

// NewNamespaceAddConfig returns a new NamespaceAddConfig.
func NewNamespaceAddConfig() NamespaceAddConfig {
	return NamespaceAddConfig{
		ClusterType: kubernetes.ClusterTypeUnknown,
		Pretty:      true,
	}
}

// PopulateNamespaces function to fill the configuration with the required NamespaceList.
// This function shall be called only in cases when there is no other way to obtain values for NamespaceList.
// User will be asked to provide the namespaces in interactive mode (if it is enabled).
// Provided by user namespaces will be parsed, validated and stored in the NamespaceList property.
// Note: in case NamespaceList is not empty - it will be overwritten by user's input.
func (cfg *NamespaceAddConfig) PopulateNamespaces(ctx context.Context) error {
	if cfg.SkipWizard {
		return errors.Join(fmt.Errorf("can't ask user for namespaces to install"), ErrInteractiveModeDisabled)
	}

	var err error
	var ns string
	// Ask user to provide namespaces in interactive mode.
	if ns, err = tui.NewInput(ctx,
		"Provide database namespaces to be managed by Everest",
		tui.WithInputDefaultValue(common.DefaultDBNamespaceName),
		tui.WithInputHint("Namespaces can be provided in comma-separated form: ns-1,ns-2"),
	).Run(); err != nil {
		return err
	}

	nsList := ParseNamespaceNames(ns)
	if err = cfg.ValidateNamespaces(ctx, nsList); err != nil {
		return err
	}

	cfg.NamespaceList = nsList
	return nil
}

// PopulateOperators function to fill the configuration with the required Operators.
// This function shall be called only in cases when there is no other way to obtain values for Operators.
// User will be asked to provide the operators in interactive mode (if it is enabled).
// Provided by user operators will be stored in the Operators property.
// Note: Operators property will be overwritten by user's input.
func (cfg *NamespaceAddConfig) PopulateOperators(ctx context.Context) error {
	if cfg.SkipWizard {
		return fmt.Errorf("can't ask user for operators to install: %w", ErrInteractiveModeDisabled)
	}

	// By default, all operators are selected.
	defaultOpts := []tui.MultiSelectOption{
		{common.MySQLProductName, true},
		{common.MongoDBProductName, true},
		{common.PostgreSQLProductName, true},
	}

	var selectedOpts []tui.MultiSelectOption
	var err error
	if selectedOpts, err = tui.NewMultiSelect(
		ctx,
		"Which operators do you want to install?",
		defaultOpts,
	).Run(); err != nil {
		return err
	}

	// Copy user's choice to config.
	for _, op := range selectedOpts {
		switch op.Text {
		case common.MySQLProductName:
			cfg.Operators.PXC = op.Selected
		case common.MongoDBProductName:
			cfg.Operators.PSMDB = op.Selected
		case common.PostgreSQLProductName:
			cfg.Operators.PG = op.Selected
		}
	}

	if !(cfg.Operators.PXC || cfg.Operators.PG || cfg.Operators.PSMDB) {
		// need to select at least one operator to install
		return ErrOperatorsNotSelected
	}

	return nil
}

// ValidateNamespaces validates the provided list of namespaces.
// It validates:
// - namespace names
// - namespace ownership
func (cfg *NamespaceAddConfig) ValidateNamespaces(ctx context.Context, nsList []string) error {
	if err := validateNamespaceNames(nsList); err != nil {
		return err
	}

	k, err := cliutils.NewKubeConnector(zap.NewNop().Sugar(), cfg.KubeconfigPath)
	if err != nil {
		return err
	}

	for _, ns := range nsList {
		if err := cfg.validateNamespaceOwnership(ctx, k, ns); err != nil {
			return err
		}
	}
	return nil
}

// validateNamespaceOwnership validates the namespace existence and ownership.
func (cfg *NamespaceAddConfig) validateNamespaceOwnership(
	ctx context.Context,
	k kubernetes.KubernetesConnector,
	namespace string,
) error {
	nsExists, ownedByEverest, err := namespaceExists(ctx, k, namespace)
	if err != nil {
		return err
	}

	if cfg.Update {
		if !ownedByEverest {
			return NewErrNamespaceNotManagedByEverest(namespace)
		}

		if !nsExists {
			return NewErrNamespaceNotExist(namespace)
		}

		return nil
	}

	if nsExists && ownedByEverest {
		return NewErrNamespaceAlreadyManagedByEverest(namespace)
	}

	if nsExists && !cfg.TakeOwnership {
		return NewErrNamespaceAlreadyExists(namespace)
	}

	return nil
}

// detectKubernetesEnv detects the Kubernetes environment where Everest is installed.
func (cfg *NamespaceAddConfig) detectKubernetesEnv(ctx context.Context, l *zap.SugaredLogger) error {
	if cfg.SkipEnvDetection {
		return nil
	}

	client, err := cliutils.NewKubeConnector(l, cfg.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	t, err := client.GetClusterType(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect cluster type: %w", err)
	}

	cfg.ClusterType = t
	// Skip detecting Kubernetes environment in the future.
	cfg.SkipEnvDetection = true
	l.Infof("Detected Kubernetes environment: %s", t)
	return nil
}

// --- NewNamespaceAdd functions

// NewNamespaceAdd returns a new CLI operation to add namespaces.
func NewNamespaceAdd(c NamespaceAddConfig, l *zap.SugaredLogger) (*NamespaceAdder, error) {
	{
		// validate the provided configuration
		if len(c.NamespaceList) == 0 {
			// need to provide at least one namespace to install
			return nil, ErrNamespaceListEmpty
		}

		if !(c.Operators.PXC || c.Operators.PG || c.Operators.PSMDB) {
			// need to select at least one operator to install
			return nil, ErrOperatorsNotSelected
		}
	}

	n := &NamespaceAdder{
		cfg: c,
		l:   l.With("component", "namespace-adder"),
	}
	if c.Pretty {
		n.l = zap.NewNop().Sugar()
	}

	k, err := cliutils.NewKubeConnector(n.l, c.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	n.kubeClient = k
	return n, nil
}

// Run namespace add operation.
func (n *NamespaceAdder) Run(ctx context.Context) error {
	// This command expects a Helm based installation (>= 1.4.0)
	dbNSChartVersion, err := cliutils.CheckHelmInstallation(ctx, n.kubeClient)
	if err != nil {
		return err
	}

	if version.IsDev(dbNSChartVersion) && n.cfg.HelmConfig.ChartDir == "" {
		// Note: new value will be set to n.cfg.ChartDir inside SetupEverestDevChart
		cleanup, err := helmutils.SetupEverestDevChart(n.l, &n.cfg.HelmConfig.ChartDir)
		if err != nil {
			return err
		}
		defer cleanup()
	}

	if err := n.cfg.detectKubernetesEnv(ctx, n.l); err != nil {
		return fmt.Errorf("failed to detect Kubernetes environment: %w", err)
	}

	installSteps, err := n.GetNamespaceInstallSteps(ctx, dbNSChartVersion)
	if err != nil {
		return err
	}

	if err := steps.RunStepsWithSpinner(ctx, n.l, installSteps, n.cfg.Pretty); err != nil {
		return err
	}
	return nil
}

// GetNamespaceInstallSteps returns the steps to install namespaces.
func (n *NamespaceAdder) GetNamespaceInstallSteps(ctx context.Context, dbNSChartVersion string) ([]steps.Step, error) {
	if n.cfg.Update {
		// validate operators updated list for each namespace.
		for _, namespace := range n.cfg.NamespaceList {
			err := n.validateNamespaceUpdate(ctx, namespace)
			if errors.Is(err, ErrCannotRemoveOperators) {
				msg := "Removal of an installed operator is not supported. Proceeding without removal."
				_, _ = fmt.Fprint(os.Stdout, output.Warn("%s", msg))
				n.l.Warn(msg)
				break
			} else if err != nil {
				return nil, fmt.Errorf("namespace validation error: %w", err)
			}
		}
	}

	var installSteps []steps.Step
	for _, namespace := range n.cfg.NamespaceList {
		installSteps = append(installSteps,
			n.newStepInstallNamespace(dbNSChartVersion, namespace),
		)
	}

	return installSteps, nil
}

func (n *NamespaceAdder) getValues() values.Options {
	var v []string
	v = append(v, "cleanupOnUninstall=false") // uninstall command will do the clean-up on its own.
	v = append(v, fmt.Sprintf("pxc=%t", n.cfg.Operators.PXC))
	v = append(v, fmt.Sprintf("postgresql=%t", n.cfg.Operators.PG))
	v = append(v, fmt.Sprintf("psmdb=%t", n.cfg.Operators.PSMDB))
	v = append(v, fmt.Sprintf("telemetry=%t", !n.cfg.DisableTelemetry))

	if n.cfg.ClusterType == kubernetes.ClusterTypeOpenShift {
		v = append(v, "compatibility.openshift=true")
	}
	return values.Options{Values: v}
}

func (n *NamespaceAdder) newStepInstallNamespace(version, namespace string) steps.Step {
	action := "Provisioning"
	if n.cfg.Update {
		action = "Updating"
	}
	return steps.Step{
		Desc: fmt.Sprintf("%s database namespace '%s'", action, namespace),
		F: func(ctx context.Context) error {
			return n.provisionDBNamespace(ctx, version, namespace)
		},
	}
}

func (n *NamespaceAdder) provisionDBNamespace(
	ctx context.Context,
	version string,
	namespace string,
) error {
	nsExists, _, err := namespaceExists(ctx, n.kubeClient, namespace)
	if err != nil {
		return err
	}
	values := Must(helmutils.MergeVals(n.getValues(), nil))
	installer := helm.Installer{
		ReleaseName:            namespace,
		ReleaseNamespace:       namespace,
		Values:                 values,
		CreateReleaseNamespace: !nsExists,
	}
	if err := installer.Init(n.cfg.KubeconfigPath, helm.ChartOptions{
		Directory: cliutils.DBNamespaceSubChartPath(n.cfg.HelmConfig.ChartDir),
		URL:       n.cfg.HelmConfig.RepoURL,
		Name:      helm.EverestDBNamespaceChartName,
		Version:   version,
	}); err != nil {
		return fmt.Errorf("could not initialize Helm installer: %w", err)
	}
	n.l.Info("Installing DB namespace Helm chart in namespace ", namespace)
	return installer.Install(ctx)
}

func (n *NamespaceAdder) validateNamespaceUpdate(ctx context.Context, namespace string) error {
	subscriptions, err := n.kubeClient.ListSubscriptions(ctx, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("cannot list subscriptions: %w", err)
	}
	if !ensureNoOperatorsRemoved(subscriptions.Items,
		n.cfg.Operators.PG, n.cfg.Operators.PXC, n.cfg.Operators.PSMDB,
	) {
		return ErrCannotRemoveOperators
	}
	return nil
}
