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
	deleteCmd = &cobra.Command{
		Use:   "delete [flags]",
		Args:  cobra.NoArgs,
		Short: "Delete an instance preset",
		Long: `Delete an InstancePreset through the Everest API.

The name and namespace flags are required. Interactively (a terminal, no
--yes), you'll be asked to confirm with y/N. Pass --yes/-y to skip the
prompt; in a non-interactive context (no terminal, or -o json|yaml)
omitting --yes fails immediately. --ignore-not-found treats an already-
absent preset as success.`,
		Example: `  # Delete an instance preset (prompts y/N)
  everestctl preset delete --name mongodb-production --namespace everest

  # Scripted
  everestctl preset delete --name mongodb-production -n everest -y

  # Idempotent teardown
  everestctl preset delete --name mongodb-production -n everest --yes --ignore-not-found`,
		PreRun: deletePreRun,
		Run:    deleteRun,
	}
	deleteCfg  = &presetcli.Config{}
	deleteOpts = &presetcli.DeleteOptions{}
)

func init() {
	deleteCmd.Flags().StringVar(&deleteOpts.Name, cli.FlagPresetName, "", "Instance preset name (required)")
	deleteCmd.Flags().StringVarP(&deleteOpts.Namespace, cli.FlagPresetNamespace, "n", "", "Namespace the instance preset is in (required)")
	deleteCmd.Flags().StringVar(&deleteOpts.Cluster, cli.FlagPresetCluster, "main", "Cluster name")
	deleteCmd.Flags().StringVar(&deleteOpts.Context, cli.FlagPresetContext, "", "Context to use (default: current context)")
	deleteCmd.Flags().BoolVarP(&deleteOpts.Yes, cli.FlagYes, "y", false, "Skip the confirmation prompt")
	deleteCmd.Flags().BoolVar(&deleteOpts.IgnoreNotFound, cli.FlagPresetIgnoreNotFound, false, "Treat \"instance preset already gone\" as a successful delete instead of an error")

	_ = deleteCmd.MarkFlagRequired(cli.FlagPresetName)
	_ = deleteCmd.MarkFlagRequired(cli.FlagPresetNamespace)
}

func deletePreRun(cmd *cobra.Command, _ []string) {
	deleteCfg.Pretty = cli.IsPretty(cmd)
	deleteCfg.YAML = cli.IsYAML(cmd)
	deleteOpts.MachineReadable = cli.IsMachineReadable(cmd)
}

func deleteRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), deleteCfg.Pretty)
		os.Exit(1)
	}

	d := presetcli.NewDeleter(*deleteCfg, logger.GetLogger())
	if err := d.Run(cmd.Context(), *deleteOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), deleteCfg.Pretty)
		os.Exit(1)
	}
}

// GetDeleteCmd returns the delete command.
func GetDeleteCmd() *cobra.Command {
	return deleteCmd
}
