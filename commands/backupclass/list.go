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

// Package backupclass holds commands for backup class management.
package backupclass

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	backupclasscli "github.com/openeverest/openeverest/v2/pkg/cli/backupclass"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	listCmd = &cobra.Command{
		Use:     "list [flags]",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List backup classes",
		Long: `List BackupClass resources via the Everest API.

Displays a table with NAME, PROVIDER, EXECUTION MODE, and AGE. Details such
as job specs, config schemas, and UI hints remain available via --json / --verbose.`,
		Example: `  # List all backup classes
  everestctl backup-class list

  # Using aliases
  everestctl bc ls

  # JSON output for scripting
  everestctl backup-class list --json | jq '.[].metadata.name'

  # Specific cluster and context
  everestctl backup-class list --cluster staging --context staging-ctx`,
		PreRun: listPreRun,
		Run:    listRun,
	}
	listCfg  = &backupclasscli.Config{}
	listOpts = &backupclasscli.ListOptions{}
)

func init() {
	listCmd.Flags().StringVar(&listOpts.Cluster, cli.FlagBackupClassCluster, "main", "Cluster name")
	listCmd.Flags().StringVar(&listOpts.Context, cli.FlagBackupClassContext, "", "Context to use (default: current context)")
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

	runner := backupclasscli.NewListRunner(*listCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *listOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}
}

// GetListCmd returns the list command.
func GetListCmd() *cobra.Command {
	return listCmd
}
