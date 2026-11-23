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

package restore

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	restorecli "github.com/openeverest/openeverest/v2/pkg/cli/restore"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	getCmd = &cobra.Command{
		Use:     "get [flags]",
		Aliases: []string{"describe"},
		Args:    cobra.NoArgs,
		Short:   "Get a restore",
		Long: `Fetch a single Restore resource via the Everest API.

Prints the full API object as JSON (default) or YAML (--yaml). The restore is
addressed by name and namespace; --instance is not required.`,
		Example: `  # Fetch a restore
  everestctl restore get --name restore-1 --namespace everest

  # YAML output
  everestctl restore get --name restore-1 --namespace everest --yaml

  # Scripting — check state
  everestctl restore get --name restore-1 --namespace everest --json \
    | jq -r '.status.state'`,
		PreRun: getPreRun,
		Run:    getRun,
	}
	getCfg  = &restorecli.Config{}
	getOpts = &restorecli.GetOptions{}
)

func init() {
	getCmd.Flags().StringVar(&getOpts.Name, cli.FlagRestoreName, "", "Restore name (required)")
	getCmd.Flags().StringVarP(&getOpts.Namespace, cli.FlagRestoreNamespace, "n", "", "Namespace the restore is in (required)")
	getCmd.Flags().StringVar(&getOpts.Cluster, cli.FlagRestoreCluster, "main", "Cluster name")
	getCmd.Flags().StringVar(&getOpts.Context, cli.FlagRestoreContext, "", "Context to use (default: current context)")

	_ = getCmd.MarkFlagRequired(cli.FlagRestoreName)
	_ = getCmd.MarkFlagRequired(cli.FlagRestoreNamespace)
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

	runner := restorecli.NewGetRunner(*getCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *getOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), getCfg.Pretty)
		os.Exit(1)
	}
}

// GetGetCmd returns the get command.
func GetGetCmd() *cobra.Command {
	return getCmd
}
