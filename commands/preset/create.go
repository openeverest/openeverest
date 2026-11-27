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

//nolint:dupl // create and update share the same -f identity flag surface
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
	createCmd = &cobra.Command{
		Use:   "create [flags]",
		Args:  cobra.NoArgs,
		Short: "Create an instance preset",
		Long: `Create an InstancePreset from a YAML/JSON manifest via the Everest API.

-f is required and must contain an InstancePreset object (apiVersion/kind
optional). metadata.name and metadata.namespace may live in the file or be
set with --name / --namespace (flags win). Server-managed fields from a
prior 'preset get -o yaml' (status, resourceVersion, uid, …) are stripped
before create.`,
		Example: `  # Create from a manifest
  everestctl preset create -f mongodb-production.yaml

  # Promote into another namespace under a new name
  everestctl preset create -f mongodb-production.yaml -n staging --name mongodb-staging`,
		PreRun: createPreRun,
		Run:    createRun,
	}
	createCfg  = &presetcli.Config{}
	createOpts = &presetcli.CreateOptions{}
)

func init() {
	createCmd.Flags().StringVarP(&createOpts.File, cli.FlagPresetFile, "f", "", "Path to an InstancePreset YAML/JSON manifest (required)")
	createCmd.Flags().StringVar(&createOpts.Name, cli.FlagPresetName, "", "Override metadata.name from the file")
	createCmd.Flags().StringVarP(&createOpts.Namespace, cli.FlagPresetNamespace, "n", "", "Override metadata.namespace from the file")
	createCmd.Flags().StringVar(&createOpts.Cluster, cli.FlagPresetCluster, "main", "Cluster name")
	createCmd.Flags().StringVar(&createOpts.Context, cli.FlagPresetContext, "", "Context to use (default: current context)")

	_ = createCmd.MarkFlagRequired(cli.FlagPresetFile)
}

func createPreRun(cmd *cobra.Command, _ []string) {
	createCfg.Pretty = cli.IsPretty(cmd)
	createCfg.YAML = cli.IsYAML(cmd)
}

func createRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(1)
	}

	runner := presetcli.NewCreateRunner(*createCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *createOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(1)
	}
}

// GetCreateCmd returns the create command.
func GetCreateCmd() *cobra.Command {
	return createCmd
}
