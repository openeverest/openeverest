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

// Package restore holds commands for restore management.
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
	listCmd = &cobra.Command{
		Use:     "list [flags]",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List restores for an instance",
		Long: `List Restore resources for an instance via the Everest API.

Displays a table with NAME, NAMESPACE, INSTANCE, BACKUP, STATE, and AGE. Pass
--json / --verbose to get the raw API response instead of the formatted
table.

There is currently no API to list restores across an entire namespace, so
--instance is required.`,
		Example: `  # List restores for an instance
  everestctl restore list --namespace everest --instance my-mongo

  # Using the ls alias
  everestctl restore ls --namespace everest --instance my-mongo

  # JSON output — check the state of the most recent restore
  everestctl restore list --namespace everest --instance my-mongo --json \
    | jq 'last | .status.state'

  # Specific cluster and context
  everestctl restore list --namespace everest --instance my-mongo --cluster staging --context staging-ctx`,
		PreRun: listPreRun,
		Run:    listRun,
	}
	listCfg  = &restorecli.Config{}
	listOpts = &restorecli.ListOptions{}
)

func init() {
	listCmd.Flags().StringVarP(&listOpts.Namespace, cli.FlagRestoreNamespace, "n", "", "Namespace the instance is located in")
	listCmd.Flags().StringVar(&listOpts.Instance, cli.FlagRestoreInstance, "", "Instance to list restores for")
	listCmd.Flags().StringVar(&listOpts.Cluster, cli.FlagRestoreCluster, "main", "Cluster name")
	listCmd.Flags().StringVar(&listOpts.Context, cli.FlagRestoreContext, "", "Context to use (default: current context)")

	_ = listCmd.MarkFlagRequired(cli.FlagRestoreNamespace)
	_ = listCmd.MarkFlagRequired(cli.FlagRestoreInstance)
}

func listPreRun(cmd *cobra.Command, _ []string) {
	listCfg.Pretty = !cmd.Flag(cli.FlagVerbose).Changed && !cmd.Flag(cli.FlagJSON).Changed
}

func listRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}

	runner := restorecli.NewListRunner(*listCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *listOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}
}

// GetListCmd returns the list command.
func GetListCmd() *cobra.Command {
	return listCmd
}
