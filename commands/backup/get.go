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
	getCmd = &cobra.Command{
		Use:     "get [flags]",
		Aliases: []string{"describe"},
		Args:    cobra.NoArgs,
		Short:   "Get a backup",
		Long: `Fetch a single Backup resource via the Everest API.

Prints the full API object as JSON (default) or YAML (-o yaml). The backup is
addressed by name and namespace; --instance is not required.`,
		Example: `  # Fetch a backup
  everestctl backup get --name pre-upgrade --namespace everest

  # YAML output
  everestctl backup get --name pre-upgrade --namespace everest -o yaml

  # Scripting — check state
  everestctl backup get --name pre-upgrade --namespace everest -o json \
    | jq -r '.status.state'`,
		PreRun: getPreRun,
		Run:    getRun,
	}
	getCfg  = &backupcli.Config{}
	getOpts = &backupcli.GetOptions{}
)

func init() {
	getCmd.Flags().StringVar(&getOpts.Name, cli.FlagBackupName, "", "Backup name (required)")
	getCmd.Flags().StringVarP(&getOpts.Namespace, cli.FlagBackupNamespace, "n", "", "Namespace the backup is in (required)")
	getCmd.Flags().StringVar(&getOpts.Cluster, cli.FlagBackupCluster, "main", "Cluster name")
	getCmd.Flags().StringVar(&getOpts.Context, cli.FlagBackupContext, "", "Context to use (default: current context)")

	_ = getCmd.MarkFlagRequired(cli.FlagBackupName)
	_ = getCmd.MarkFlagRequired(cli.FlagBackupNamespace)
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

	runner := backupcli.NewGetRunner(*getCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *getOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), getCfg.Pretty)
		os.Exit(1)
	}
}

// GetGetCmd returns the get command.
func GetGetCmd() *cobra.Command {
	return getCmd
}
