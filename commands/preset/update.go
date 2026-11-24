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
	updateCmd = &cobra.Command{
		Use:   "update [flags]",
		Args:  cobra.NoArgs,
		Short: "Update an instance preset",
		Long: `Replace an InstancePreset with a YAML/JSON manifest via the Everest API.

The API replaces the entire resource (PUT). -f is required. metadata.name
and metadata.namespace may live in the file or be set with --name /
--namespace (flags win). The live resourceVersion is applied automatically
so manifests checked into git without one still update.`,
		Example: `  # Replace from a versioned manifest
  everestctl preset update -f mongodb-production.yaml

  # Name/namespace from flags when the file omits them
  everestctl preset update --name mongodb-production -n everest -f mongodb-production.yaml`,
		PreRun: updatePreRun,
		Run:    updateRun,
	}
	updateCfg  = &presetcli.Config{}
	updateOpts = &presetcli.UpdateOptions{}
)

func init() {
	updateCmd.Flags().StringVarP(&updateOpts.File, cli.FlagPresetFile, "f", "", "Path to an InstancePreset YAML/JSON manifest (required)")
	updateCmd.Flags().StringVar(&updateOpts.Name, cli.FlagPresetName, "", "Override metadata.name from the file")
	updateCmd.Flags().StringVarP(&updateOpts.Namespace, cli.FlagPresetNamespace, "n", "", "Override metadata.namespace from the file")
	updateCmd.Flags().StringVar(&updateOpts.Cluster, cli.FlagPresetCluster, "main", "Cluster name")
	updateCmd.Flags().StringVar(&updateOpts.Context, cli.FlagPresetContext, "", "Context to use (default: current context)")

	_ = updateCmd.MarkFlagRequired(cli.FlagPresetFile)
}

func updatePreRun(cmd *cobra.Command, _ []string) {
	updateCfg.Pretty = cli.IsPretty(cmd)
	updateCfg.YAML = cli.IsYAML(cmd)
}

func updateRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), updateCfg.Pretty)
		os.Exit(1)
	}

	runner := presetcli.NewUpdater(*updateCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *updateOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), updateCfg.Pretty)
		os.Exit(1)
	}
}

// GetUpdateCmd returns the update command.
func GetUpdateCmd() *cobra.Command {
	return updateCmd
}
