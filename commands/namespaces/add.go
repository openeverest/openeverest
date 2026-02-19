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

// Package namespaces provides the namespaces CLI command.
package namespaces

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/helm"
	"github.com/openeverest/openeverest/v2/pkg/cli/namespaces"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

//nolint:gochecknoglobals
var (
	takeOwnershipHintMessage = fmt.Sprintf("HINT: set '--%s' flag to use existing namespaces", cli.FlagTakeNamespaceOwnership)
	updateHintMessage        = "HINT: use 'everestctl namespaces update' to update the namespace"
	namespacesAddCmd         = &cobra.Command{
		Use:   "add <namespaces> [flags]",
		Args:  cobra.ExactArgs(1),
		Long:  "Add a new namespace and make managed by Everest",
		Short: "Add a new namespace and make managed by Everest",
		Example: fmt.Sprintf("everestctl namespaces add ns-1,ns-2 --%s --%s=true --%s=false --%s=false",
			cli.FlagSkipWizard, cli.FlagOperatorMySQL, cli.FlagOperatorPostgresql, cli.FlagOperatorMongoDB,
		),
		PreRun: namespacesAddPreRun,
		Run:    namespacesAddRun,
	}
	namespacesAddCfg = namespaces.NewNamespaceAddConfig()
)

func init() {
	// local command flags
	namespacesAddCmd.Flags().BoolVar(&namespacesAddCfg.DisableTelemetry, cli.FlagDisableTelemetry, false, "Disable telemetry")
	_ = namespacesAddCmd.Flags().MarkHidden(cli.FlagDisableTelemetry) //nolint:errcheck,gosec
	namespacesAddCmd.Flags().BoolVar(&namespacesAddCfg.SkipWizard, cli.FlagSkipWizard, false, "Skip installation wizard")
	namespacesAddCmd.Flags().BoolVar(&namespacesAddCfg.TakeOwnership, cli.FlagTakeNamespaceOwnership, false, "If the specified namespace already exists, take ownership of it")
	namespacesAddCmd.Flags().BoolVar(&namespacesAddCfg.SkipEnvDetection, cli.FlagSkipEnvDetection, false, "Skip detecting Kubernetes environment where Everest is installed")

	// --helm.* flags
	namespacesAddCmd.Flags().StringVar(&namespacesAddCfg.HelmConfig.ChartDir, helm.FlagChartDir, "", "Path to the chart directory. If not set, the chart will be downloaded from the repository")
	_ = namespacesAddCmd.Flags().MarkHidden(helm.FlagChartDir) //nolint:errcheck,gosec
	namespacesAddCmd.Flags().StringVar(&namespacesAddCfg.HelmConfig.RepoURL, helm.FlagRepository, helm.DefaultHelmRepoURL, "Helm chart repository to download the Everest charts from")
	namespacesAddCmd.Flags().StringSliceVar(&namespacesAddCfg.HelmConfig.Values.Values, helm.FlagHelmSet, []string{}, "Set helm values on the command line (can specify multiple values with commas: key1=val1,key2=val2)")
	namespacesAddCmd.Flags().StringSliceVarP(&namespacesAddCfg.HelmConfig.Values.ValueFiles, helm.FlagHelmValues, "f", []string{}, "Specify values in a YAML file or a URL (can specify multiple)")

	// --operator.* flags
	namespacesAddCmd.Flags().BoolVar(&namespacesAddCfg.Operators.PSMDB, cli.FlagOperatorMongoDB, true, "Install MongoDB operator")
	namespacesAddCmd.Flags().BoolVar(&namespacesAddCfg.Operators.PG, cli.FlagOperatorPostgresql, true, "Install PostgreSQL operator")
	namespacesAddCmd.Flags().BoolVar(&namespacesAddCfg.Operators.PXC, cli.FlagOperatorXtraDBCluster, true, "Install XtraDB Cluster operator")
	_ = namespacesAddCmd.Flags().MarkDeprecated(cli.FlagOperatorXtraDBCluster, fmt.Sprintf("please use --%s instead", cli.FlagOperatorMySQL))
	namespacesAddCmd.Flags().BoolVar(&namespacesAddCfg.Operators.PXC, cli.FlagOperatorMySQL, true, "Install MySQL operator")
}

func namespacesAddPreRun(cmd *cobra.Command, args []string) { //nolint:revive
	// Copy global flags to config
	namespacesAddCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	namespacesAddCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()

	{
		// Parse and validate provided namespaces
		nsList := namespaces.ParseNamespaceNames(args[0])
		if err := namespacesAddCfg.ValidateNamespaces(cmd.Context(), nsList); err != nil {
			if errors.Is(err, namespaces.ErrNamespaceAlreadyExists) {
				err = fmt.Errorf("%w. %s", err, takeOwnershipHintMessage)
			}
			if errors.Is(err, namespaces.ErrNamespaceAlreadyManagedByEverest) {
				err = fmt.Errorf("%w. %s", err, updateHintMessage)
			}
			output.PrintError(err, logger.GetLogger(), namespacesAddCfg.Pretty)
			os.Exit(1)
		}

		namespacesAddCfg.NamespaceList = nsList
	}

	// If user doesn't pass any --operator.* flags - need to ask explicitly.
	askOperators := !(cmd.Flags().Lookup(cli.FlagOperatorMongoDB).Changed ||
		cmd.Flags().Lookup(cli.FlagOperatorPostgresql).Changed ||
		cmd.Flags().Lookup(cli.FlagOperatorXtraDBCluster).Changed ||
		cmd.Flags().Lookup(cli.FlagOperatorMySQL).Changed ||
		namespacesAddCfg.SkipWizard)

	if askOperators {
		// need to ask user to provide operators to be installed in interactive mode.
		if err := namespacesAddCfg.PopulateOperators(cmd.Context()); err != nil {
			output.PrintError(err, logger.GetLogger(), namespacesAddCfg.Pretty)
			os.Exit(1)
		}
	}
}

func namespacesAddRun(cmd *cobra.Command, _ []string) {
	op, err := namespaces.NewNamespaceAdd(namespacesAddCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), namespacesAddCfg.Pretty)
		os.Exit(1)
	}

	if err := op.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), namespacesAddCfg.Pretty)
		os.Exit(1)
	}
}

// GetNamespacesAddCmd returns the command to add namespaces.
func GetNamespacesAddCmd() *cobra.Command {
	return namespacesAddCmd
}
