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

// Package backup holds commands for backup management.
package backup

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	backupcli "github.com/openeverest/openeverest/v2/pkg/cli/backup"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	listCmd = &cobra.Command{
		Use:     "list [flags]",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List backups for an instance",
		Long: `List Backup resources for an instance via the Everest API.

Displays a table with NAME, NAMESPACE, INSTANCE, STORAGE, STATE, SIZE, and
AGE. Scheduled backups (those with spec.scheduleName set) appear alongside
on-demand ones. Pass --json / --verbose to get the raw API response instead
of the formatted table.

There is currently no API to list backups across an entire namespace, so
--instance is required.`,
		Example: `  # List backups for an instance
  everestctl backup list --namespace everest --instance my-mongo

  # Using the ls alias
  everestctl backup ls --namespace everest --instance my-mongo

  # JSON output — check the latest completed backup
  everestctl backup list --namespace everest --instance my-mongo --json \
    | jq '[.[] | select(.status.state == "Succeeded")] | last'

  # Specific cluster and context
  everestctl backup list --namespace everest --instance my-mongo --cluster staging --context staging-ctx`,
		PreRun: listPreRun,
		Run:    listRun,
	}
	listCfg  = &backupcli.Config{}
	listOpts = &backupcli.ListOptions{}
)

func init() {
	listCmd.Flags().StringVarP(&listOpts.Namespace, cli.FlagBackupNamespace, "n", "", "Namespace the instance is located in")
	listCmd.Flags().StringVar(&listOpts.Instance, cli.FlagBackupInstance, "", "Instance to list backups for")
	listCmd.Flags().StringVar(&listOpts.Cluster, cli.FlagBackupCluster, "main", "Cluster name")
	listCmd.Flags().StringVar(&listOpts.Context, cli.FlagBackupContext, "", "Context to use (default: current context)")

	_ = listCmd.MarkFlagRequired(cli.FlagBackupNamespace)
	_ = listCmd.MarkFlagRequired(cli.FlagBackupInstance)
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

	runner := backupcli.NewListRunner(*listCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *listOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}
}

// GetListCmd returns the list command.
func GetListCmd() *cobra.Command {
	return listCmd
}
