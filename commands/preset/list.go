// everest
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

// Package preset holds commands for InstancePreset management.
package preset

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	presetcli "github.com/openeverest/openeverest/v2/pkg/cli/preset"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	listCmd = &cobra.Command{
		Use:     "list [flags]",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List instance presets",
		Long: `List InstancePreset resources via the Everest API.

Displays a table with NAME, NAMESPACE, PROVIDER, VERSION and AGE. Full
objects remain available via -o json / -o yaml / --verbose.`,
		Example: `  # List instance presets in a namespace
  everestctl preset list --namespace everest

  # List across all namespaces
  everestctl preset ls --all-namespaces

  # Only presets for a provider
  everestctl preset list -n everest --provider percona-server-mongodb

  # Export for git
  everestctl preset list -n everest -o yaml`,
		PreRun: listPreRun,
		Run:    listRun,
	}
	listCfg  = &presetcli.Config{}
	listOpts = &presetcli.ListOptions{}
)

func init() {
	listCmd.Flags().StringVarP(&listOpts.Namespace, cli.FlagPresetNamespace, "n", "", "Namespace to list instance presets from")
	listCmd.Flags().BoolVarP(&listOpts.AllNamespaces, cli.FlagPresetAllNamespaces, "A", false, "List instance presets across all namespaces")
	listCmd.Flags().StringVar(&listOpts.Provider, cli.FlagPresetProvider, "", "Only list presets for this provider")
	listCmd.Flags().StringVar(&listOpts.Cluster, cli.FlagPresetCluster, "main", "Cluster name")
	listCmd.Flags().StringVar(&listOpts.Context, cli.FlagPresetContext, "", "Context to use (default: current context)")

	listCmd.MarkFlagsOneRequired(cli.FlagPresetNamespace, cli.FlagPresetAllNamespaces)
	listCmd.MarkFlagsMutuallyExclusive(cli.FlagPresetNamespace, cli.FlagPresetAllNamespaces)
}

func listPreRun(cmd *cobra.Command, _ []string) {
	listCfg.Pretty = cli.IsPretty(cmd)
	listCfg.YAML = cli.IsYAML(cmd)
}

func listRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}

	runner := presetcli.NewListRunner(*listCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *listOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}
}

// GetListCmd returns the list command.
func GetListCmd() *cobra.Command {
	return listCmd
}
