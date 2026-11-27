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
	getCmd = &cobra.Command{
		Use:     "get [flags]",
		Aliases: []string{"describe"},
		Args:    cobra.NoArgs,
		Short:   "Get an instance preset",
		Long: `Fetch a single InstancePreset resource via the Everest API.

Prints the full API object as JSON (default) or YAML (-o yaml). Use -o yaml
to export a template for create/update or to version in git.`,
		Example: `  # Fetch an instance preset
  everestctl preset get --name mongodb-production --namespace everest

  # YAML export for git
  everestctl preset get --name mongodb-production -n everest -o yaml > mongodb-production.yaml`,
		PreRun: getPreRun,
		Run:    getRun,
	}
	getCfg  = &presetcli.Config{}
	getOpts = &presetcli.GetOptions{}
)

func init() {
	getCmd.Flags().StringVar(&getOpts.Name, cli.FlagPresetName, "", "Instance preset name (required)")
	getCmd.Flags().StringVarP(&getOpts.Namespace, cli.FlagPresetNamespace, "n", "", "Namespace the instance preset is in (required)")
	getCmd.Flags().StringVar(&getOpts.Cluster, cli.FlagPresetCluster, "main", "Cluster name")
	getCmd.Flags().StringVar(&getOpts.Context, cli.FlagPresetContext, "", "Context to use (default: current context)")

	_ = getCmd.MarkFlagRequired(cli.FlagPresetName)
	_ = getCmd.MarkFlagRequired(cli.FlagPresetNamespace)
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

	runner := presetcli.NewGetRunner(*getCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *getOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), getCfg.Pretty)
		os.Exit(1)
	}
}

// GetGetCmd returns the get command.
func GetGetCmd() *cobra.Command {
	return getCmd
}
