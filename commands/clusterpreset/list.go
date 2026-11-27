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

// Package clusterpreset holds commands for ClusterInstancePreset management.
package clusterpreset

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	clusterpresetcli "github.com/openeverest/openeverest/v2/pkg/cli/clusterpreset"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	listCmd = &cobra.Command{
		Use:     "list [flags]",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List cluster instance presets",
		Long: `List ClusterInstancePreset resources via the Everest API.

Displays a table with NAME, PROVIDER, VERSION and AGE. Full objects remain
available via -o json / -o yaml / --verbose.`,
		Example: `  # List cluster instance presets
  everestctl cluster-preset list

  # Only presets for a provider
  everestctl cluster-preset list --provider percona-server-mongodb

  # Export for git
  everestctl cluster-preset list -o yaml`,
		PreRun: listPreRun,
		Run:    listRun,
	}
	listCfg  = &clusterpresetcli.Config{}
	listOpts = &clusterpresetcli.ListOptions{}
)

func init() {
	listCmd.Flags().StringVar(&listOpts.Provider, cli.FlagClusterPresetProvider, "", "Only list presets for this provider")
	listCmd.Flags().StringVar(&listOpts.Cluster, cli.FlagClusterPresetCluster, "main", "Cluster name")
	listCmd.Flags().StringVar(&listOpts.Context, cli.FlagClusterPresetContext, "", "Context to use (default: current context)")
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

	runner := clusterpresetcli.NewListRunner(*listCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *listOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}
}

// GetListCmd returns the list command.
func GetListCmd() *cobra.Command {
	return listCmd
}
