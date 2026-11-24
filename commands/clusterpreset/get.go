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
	getCmd = &cobra.Command{
		Use:     "get [flags]",
		Aliases: []string{"describe"},
		Args:    cobra.NoArgs,
		Short:   "Get a cluster instance preset",
		Long: `Fetch a single ClusterInstancePreset resource via the Everest API.

Prints the full API object as JSON (default) or YAML (-o yaml).`,
		Example: `  # Fetch a cluster instance preset
  everestctl cluster-preset get --name mongodb-default

  # YAML export
  everestctl cluster-preset get --name mongodb-default -o yaml`,
		PreRun: getPreRun,
		Run:    getRun,
	}
	getCfg  = &clusterpresetcli.Config{}
	getOpts = &clusterpresetcli.GetOptions{}
)

func init() {
	getCmd.Flags().StringVar(&getOpts.Name, cli.FlagClusterPresetName, "", "Cluster instance preset name (required)")
	getCmd.Flags().StringVar(&getOpts.Cluster, cli.FlagClusterPresetCluster, "main", "Cluster name")
	getCmd.Flags().StringVar(&getOpts.Context, cli.FlagClusterPresetContext, "", "Context to use (default: current context)")

	_ = getCmd.MarkFlagRequired(cli.FlagClusterPresetName)
}

func getPreRun(cmd *cobra.Command, _ []string) {
	getCfg.Pretty = cli.IsPretty(cmd)
	getCfg.YAML = cli.IsYAML(cmd)
}

func getRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), getCfg.Pretty)
		os.Exit(1)
	}

	runner := clusterpresetcli.NewGetRunner(*getCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *getOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), getCfg.Pretty)
		os.Exit(1)
	}
}

// GetGetCmd returns the get command.
func GetGetCmd() *cobra.Command {
	return getCmd
}
