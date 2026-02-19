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

// Package commands ...
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/helm"
	"github.com/openeverest/openeverest/v2/pkg/cli/install"
	"github.com/openeverest/openeverest/v2/pkg/cli/namespaces"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	installCmd = &cobra.Command{
		Use: "install [flags]",
		// The command expects no arguments. So to prevent users from misspelling and confusion
		// in cases with unexpected spaces like
		//       ./everestctl install --namespaces=aaa, a
		// it will return
		//        Error: unknown command "a" for "everestctl install"
		Args:  cobra.NoArgs,
		Short: "Install Percona Everest using Helm",
		Long:  "Install Percona Everest using Helm",
		Example: fmt.Sprintf("everestctl install --%s dev,staging,prod --%s=true --%s=false --%s=false --%s",
			cli.FlagNamespaces, cli.FlagOperatorMongoDB, cli.FlagOperatorPostgresql, cli.FlagOperatorMySQL, cli.FlagSkipWizard,
		),
		PreRun: installPreRun,
		Run:    installRun,
	}
	installCfg      = install.NewInstallConfig()
	namespacesToAdd string
)

func init() {
	rootCmd.AddCommand(installCmd)

	// local command flags
	installCmd.Flags().StringVar(&namespacesToAdd, cli.FlagNamespaces, common.DefaultDBNamespaceName, "Comma-separated namespaces list Percona Everest can manage")
	installCmd.Flags().BoolVar(&installCfg.NamespaceAddConfig.SkipWizard, cli.FlagSkipWizard, false, "Skip installation wizard")
	installCmd.Flags().StringVar(&installCfg.VersionMetadataURL, cli.FlagVersionMetadataURL, "https://check.percona.com", "URL to retrieve version metadata information from")
	installCmd.Flags().StringVar(&installCfg.Version, cli.FlagVersion, "", "Everest version to install. By default the latest version is installed")
	installCmd.Flags().BoolVar(&installCfg.DisableTelemetry, cli.FlagDisableTelemetry, false, "Disable telemetry")
	_ = installCmd.Flags().MarkHidden(cli.FlagDisableTelemetry)
	installCmd.Flags().BoolVar(&installCfg.SkipEnvDetection, cli.FlagSkipEnvDetection, false, "Skip detecting Kubernetes environment where Everest is installed")
	installCmd.Flags().BoolVar(&installCfg.SkipDBNamespace, cli.FlagInstallSkipDBNamespace, false, "Skip creating a database namespace with install")

	// --namespaces and --skip-db-namespace flags are mutually exclusive
	installCmd.MarkFlagsMutuallyExclusive(cli.FlagNamespaces, cli.FlagInstallSkipDBNamespace)

	// --helm.* flags
	installCmd.Flags().StringVar(&installCfg.HelmConfig.ChartDir, helm.FlagChartDir, "", "Path to the chart directory. If not set, the chart will be downloaded from the repository")
	_ = installCmd.Flags().MarkHidden(helm.FlagChartDir)
	installCmd.Flags().StringVar(&installCfg.HelmConfig.RepoURL, helm.FlagRepository, helm.DefaultHelmRepoURL, "Helm chart repository to download the Everest charts from")
	installCmd.Flags().StringSliceVar(&installCfg.HelmConfig.Values.Values, helm.FlagHelmSet, []string{}, "Set helm values on the command line (can specify multiple values with commas: key1=val1,key2=val2)")
	installCmd.Flags().StringSliceVarP(&installCfg.HelmConfig.Values.ValueFiles, helm.FlagHelmValues, "f", []string{}, "Specify values in a YAML file or a URL (can specify multiple)")

	// --operator.* flags
	installCmd.Flags().BoolVar(&installCfg.NamespaceAddConfig.Operators.PSMDB, cli.FlagOperatorMongoDB, true, "Install MongoDB operator")
	installCmd.Flags().BoolVar(&installCfg.NamespaceAddConfig.Operators.PG, cli.FlagOperatorPostgresql, true, "Install PostgreSQL operator")
	installCmd.Flags().BoolVar(&installCfg.NamespaceAddConfig.Operators.PXC, cli.FlagOperatorXtraDBCluster, true, "Install XtraDB Cluster operator")
	_ = installCmd.Flags().MarkDeprecated(cli.FlagOperatorXtraDBCluster, fmt.Sprintf("please use --%s instead", cli.FlagOperatorMySQL))
	installCmd.Flags().BoolVar(&installCfg.NamespaceAddConfig.Operators.PXC, cli.FlagOperatorMySQL, true, "Install MySQL operator")
}

func installPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	installCfg.Pretty = rootCmdFlags.Pretty
	installCfg.KubeconfigPath = rootCmdFlags.KubeconfigPath
	installCfg.NamespaceAddConfig.KubeconfigPath = rootCmdFlags.KubeconfigPath

	// Check if Everest is already installed.
	if err := install.CheckEverestAlreadyinstalled(cmd.Context(), logger.GetLogger(), installCfg.KubeconfigPath); err != nil {
		output.PrintError(err, logger.GetLogger(), installCfg.Pretty)
		os.Exit(1)
	}

	if !installCfg.SkipDBNamespace {
		if err := checkDBNamespaceParameters(cmd); err != nil {
			output.PrintError(err, logger.GetLogger(), installCfg.Pretty)
			os.Exit(1)
		}
	}
}

// checkDBNamespaceParameters checks, validates and sets the database namespace parameters into installCfg.
// If the user doesn't pass '--namespaces' or '--operators.*' flags,
// it will ask the user to provide them in interactive mode (if it is enabled).
func checkDBNamespaceParameters(cmd *cobra.Command) error {
	// Check DB namespaces parameters
	// If user doesn't pass --namespaces flag - need to ask explicitly.
	askNamespaces := !(cmd.Flags().Lookup(cli.FlagNamespaces).Changed ||
		installCfg.NamespaceAddConfig.SkipWizard)

	// Note: there are the following cases possible:
	// - user doesn't provide '--namespaces' flag -> namespacesToAdd="everest" (default).
	// - user provides '--namespaces' flag -> namespacesToAdd contains the user provided value.
	if askNamespaces {
		// need to ask user in interactive mode to provide database namespaces to be created.
		if err := installCfg.NamespaceAddConfig.PopulateNamespaces(cmd.Context()); err != nil {
			return err
		}
	} else {
		// Parse and validate user provided namespaces.
		nsList := namespaces.ParseNamespaceNames(namespacesToAdd)
		if err := installCfg.NamespaceAddConfig.ValidateNamespaces(cmd.Context(), nsList); err != nil {
			return err
		}

		installCfg.NamespaceAddConfig.NamespaceList = nsList
	}

	// If user doesn't pass any --operator.* flags - need to ask explicitly.
	askOperators := !(cmd.Flags().Lookup(cli.FlagOperatorMongoDB).Changed ||
		cmd.Flags().Lookup(cli.FlagOperatorPostgresql).Changed ||
		cmd.Flags().Lookup(cli.FlagOperatorXtraDBCluster).Changed ||
		cmd.Flags().Lookup(cli.FlagOperatorMySQL).Changed ||
		installCfg.NamespaceAddConfig.SkipWizard)

	if askOperators {
		// need to ask user to provide operators to be installed in interactive mode.
		if err := installCfg.NamespaceAddConfig.PopulateOperators(cmd.Context()); err != nil {
			return err
		}
	}

	return nil
}

func installRun(cmd *cobra.Command, _ []string) { //nolint:revive
	op, err := install.NewInstall(installCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), installCfg.Pretty)
		os.Exit(1)
	}

	if err := op.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), installCfg.Pretty)
		os.Exit(1)
	}
}
