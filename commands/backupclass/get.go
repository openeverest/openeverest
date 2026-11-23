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
	getCmd = &cobra.Command{
		Use:     "get [flags]",
		Aliases: []string{"describe"},
		Args:    cobra.NoArgs,
		Short:   "Get a backup class",
		Long: `Fetch a single BackupClass resource via the Everest API.

Prints the full API object as JSON (default) or YAML (--yaml).`,
		Example: `  # Fetch a backup class
  everestctl backup-class get --name pg-backup

  # YAML output
  everestctl backup-class get --name pg-backup --yaml

  # Scripting — inspect execution mode
  everestctl backup-class get --name pg-backup --json \
    | jq -r '.spec.executionMode'`,
		PreRun: getPreRun,
		Run:    getRun,
	}
	getCfg  = &backupclasscli.Config{}
	getOpts = &backupclasscli.GetOptions{}
)

func init() {
	getCmd.Flags().StringVar(&getOpts.Name, cli.FlagBackupClassName, "", "Backup class name (required)")
	getCmd.Flags().StringVar(&getOpts.Cluster, cli.FlagBackupClassCluster, "main", "Cluster name")
	getCmd.Flags().StringVar(&getOpts.Context, cli.FlagBackupClassContext, "", "Context to use (default: current context)")

	_ = getCmd.MarkFlagRequired(cli.FlagBackupClassName)
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

	runner := backupclasscli.NewGetRunner(*getCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *getOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), getCfg.Pretty)
		os.Exit(1)
	}
}

// GetGetCmd returns the get command.
func GetGetCmd() *cobra.Command {
	return getCmd
}
