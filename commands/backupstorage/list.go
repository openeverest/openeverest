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

// Package backupstorage holds commands for backup storage management.
package backupstorage

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	backupstoragecli "github.com/openeverest/openeverest/v2/pkg/cli/backupstorage"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	listCmd = &cobra.Command{
		Use:     "list [flags]",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List backup storages",
		Long: `List BackupStorage resources via the Everest API.

Displays a table with NAME, NAMESPACE, TYPE, and AGE. Type-specific details
(e.g. S3 bucket, region, endpoint) remain available via --json / --verbose.
Credentials are never included in the output.`,
		Example: `  # List backup storages in a namespace
  everestctl backup-storage list --namespace everest

  # List across all namespaces
  everestctl bs ls --all-namespaces

  # JSON output — inspect S3 details
  everestctl backup-storage list --namespace everest --json | jq '.[].spec.s3'

  # Specific cluster and context
  everestctl backup-storage list --namespace everest --cluster staging --context staging-ctx`,
		PreRun: listPreRun,
		Run:    listRun,
	}
	listCfg  = &backupstoragecli.Config{}
	listOpts = &backupstoragecli.ListOptions{}
)

func init() {
	listCmd.Flags().StringVarP(&listOpts.Namespace, cli.FlagBackupStorageNamespace, "n", "", "Namespace to list backup storages from")
	listCmd.Flags().BoolVarP(&listOpts.AllNamespaces, cli.FlagBackupStorageAllNamespaces, "A", false, "List backup storages across all namespaces")
	listCmd.Flags().StringVar(&listOpts.Cluster, cli.FlagBackupStorageCluster, "main", "Cluster name")
	listCmd.Flags().StringVar(&listOpts.Context, cli.FlagBackupStorageContext, "", "Context to use (default: current context)")

	listCmd.MarkFlagsOneRequired(cli.FlagBackupStorageNamespace, cli.FlagBackupStorageAllNamespaces)
	listCmd.MarkFlagsMutuallyExclusive(cli.FlagBackupStorageNamespace, cli.FlagBackupStorageAllNamespaces)
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

	runner := backupstoragecli.NewListRunner(*listCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *listOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}
}

// GetListCmd returns the list command.
func GetListCmd() *cobra.Command {
	return listCmd
}
