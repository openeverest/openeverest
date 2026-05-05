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
	"os"

	"github.com/spf13/cobra"

	"github.com/percona/everest/pkg/cli"
	"github.com/percona/everest/pkg/cli/helm"
	"github.com/percona/everest/pkg/cli/upgrade"
	"github.com/percona/everest/pkg/logger"
	"github.com/percona/everest/pkg/output"
)

var (
	upgradeCmd = &cobra.Command{
		Use:    "upgrade [flags]",
		Args:   cobra.NoArgs,
		Long:   "Upgrade Percona Everest using Helm",
		Short:  "Upgrade Percona Everest using Helm",
		PreRun: upgradePreRun,
		Run:    upgradeRun,
	}

	upgradeCfg = &upgrade.Config{}
)

func init() {
	rootCmd.AddCommand(upgradeCmd)

	// local command flags
	upgradeCmd.Flags().StringVar(&upgradeCfg.VersionMetadataURL, cli.FlagVersionMetadataURL, "https://check.percona.com", "URL to retrieve version metadata information from")
	upgradeCmd.Flags().BoolVar(&upgradeCfg.SkipEnvDetection, cli.FlagSkipEnvDetection, false, "Skip detecting Kubernetes environment where Everest is installed")
	upgradeCmd.Flags().BoolVar(&upgradeCfg.DryRun, cli.FlagUpgradeDryRun, false, "If set, only executes the pre-upgrade checks")
	upgradeCmd.Flags().BoolVar(&upgradeCfg.InCluster, cli.FlagUpgradeInCluster, false, "If set, uses the in-cluster Kubernetes client configuration")
	upgradeCmd.Flags().StringVar(&upgradeCfg.VersionToUpgrade, cli.FlagUpgradeVersionToUpgrade, "", "(Optional) Version to upgrade to. This version may be ahead by at most one minor version from the current version")
	_ = upgradeCmd.Flags().MarkHidden(cli.FlagUpgradeInCluster)

	// --helm.* flags
	upgradeCmd.Flags().StringVar(&upgradeCfg.ChartDir, helm.FlagChartDir, "", "Path to the chart directory. If not set, the chart will be downloaded from the repository")
	_ = upgradeCmd.Flags().MarkHidden(helm.FlagChartDir)
	upgradeCmd.Flags().StringVar(&upgradeCfg.RepoURL, helm.FlagRepository, helm.DefaultHelmRepoURL, "Helm chart repository to download the Everest charts from")
	upgradeCmd.Flags().StringSliceVar(&upgradeCfg.Values.Values, helm.FlagHelmSet, []string{}, "Set helm values on the command line (can specify multiple values with commas: key1=val1,key2=val2)")
	upgradeCmd.Flags().StringSliceVarP(&upgradeCfg.Values.ValueFiles, helm.FlagHelmValues, "f", []string{}, "Specify values in a YAML file or a URL (can specify multiple)")
	upgradeCmd.Flags().BoolVar(&upgradeCfg.ResetThenReuseValues, helm.FlagHelmReuseValues, false, "Reuse the last release's values and merge in any overrides from the command line via --helm.set and -f")
	upgradeCmd.Flags().BoolVar(&upgradeCfg.ResetValues, helm.FlagHelmResetValues, false, "Reset the values to the ones built into the chart")
	upgradeCmd.Flags().BoolVar(&upgradeCfg.ResetThenReuseValues, helm.FlagHelmResetThenReuseValues, false, "Reset the values to the ones built into the chart, apply the last release's values and merge in any overrides from the command line via --set and -f.")
}

func upgradePreRun(_ *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	upgradeCfg.Pretty = rootCmdFlags.Pretty
	upgradeCfg.KubeconfigPath = rootCmdFlags.KubeconfigPath
}

func upgradeRun(cmd *cobra.Command, _ []string) { //nolint:revive
	op, err := upgrade.NewUpgrade(upgradeCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), upgradeCfg.Pretty)
		os.Exit(1)
	}

	if err := op.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), upgradeCfg.Pretty)
		os.Exit(1)
	}
}
