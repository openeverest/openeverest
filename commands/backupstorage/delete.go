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

package backupstorage

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	backupstoragecli "github.com/openeverest/openeverest/v2/pkg/cli/backupstorage"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/cli/waitcmd"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	deleteCmd = &cobra.Command{
		Use:   "delete [flags]",
		Args:  cobra.NoArgs,
		Short: "Delete a backup storage",
		Long: `Delete a BackupStorage through the Everest API.

The name and namespace flags are required. The credentials Secret is not
deleted by this command directly. The BackupStorage controller adopts any
credentials Secret with no existing owner, whether it was CLI-generated or
supplied via --credentials-secret, and Kubernetes garbage-collects it once
the BackupStorage is actually gone. A Secret shared across multiple backup
storages via --credentials-secret is not protected from this: deleting
whichever storage adopted it first takes the Secret down too.

Interactively (a terminal, no --yes), you'll be asked to confirm with y/N.
Pass --yes/-y to skip the prompt; in a non-interactive context (no terminal,
or --json) omitting --yes fails immediately instead of hanging on a prompt
nobody can answer.

--ignore-not-found treats an already-absent backup storage as success and
skips both the confirmation prompt and --wait entirely, since there's
nothing to confirm or wait for. If --wait times out, the deletion is still
running server-side, and a stuck delete is almost always an Instance or
Backup that still references the storage, so point at 'everestctl instance
list'/'everestctl backup list' for the namespace.`,
		Example: `  # Delete a backup storage (prompts y/N)
  everestctl backup-storage delete --name my-s3 --namespace everest

  # Scripted
  everestctl backup-storage delete --name my-s3 --namespace everest -y

  # Scripted teardown, wait until fully removed
  everestctl backup-storage delete --name my-s3 --namespace everest --yes --wait --timeout 5m

  # Idempotent teardown: treat "already gone" as success
  everestctl backup-storage delete --name my-s3 --namespace everest --yes --ignore-not-found`,
		PreRun: deletePreRun,
		Run:    deleteRun,
	}
	deleteCfg  = &backupstoragecli.Config{}
	deleteOpts = &backupstoragecli.DeleteOptions{}
)

func init() {
	deleteCmd.Flags().StringVar(&deleteOpts.Name, cli.FlagBackupStorageName, "", "Backup storage name (required)")
	deleteCmd.Flags().StringVarP(&deleteOpts.Namespace, cli.FlagBackupStorageNamespace, "n", "", "Namespace the backup storage is in (required)")
	deleteCmd.Flags().StringVar(&deleteOpts.Cluster, cli.FlagBackupStorageCluster, "main", "Cluster name")
	deleteCmd.Flags().StringVar(&deleteOpts.Context, cli.FlagBackupStorageContext, "", "Context to use (default: current context)")
	deleteCmd.Flags().BoolVarP(&deleteOpts.Yes, cli.FlagYes, "y", false, "Skip the confirmation prompt")
	deleteCmd.Flags().BoolVar(&deleteOpts.IgnoreNotFound, cli.FlagBackupStorageIgnoreNotFound, false, "Treat \"backup storage already gone\" as a successful delete instead of an error")
	deleteCmd.Flags().BoolVar(&deleteOpts.Wait, cli.FlagBackupStorageWait, false, "Block until the backup storage is fully deleted (Ctrl-C cancels only the wait, not the deletion)")
	deleteCmd.Flags().DurationVar(&deleteOpts.Timeout, cli.FlagBackupStorageTimeout, 5*time.Minute, "Maximum time to wait (only valid with --wait); must be positive")

	_ = deleteCmd.MarkFlagRequired(cli.FlagBackupStorageName)
	_ = deleteCmd.MarkFlagRequired(cli.FlagBackupStorageNamespace)
}

func deletePreRun(cmd *cobra.Command, _ []string) {
	deleteCfg.Pretty = !cmd.Flag(cli.FlagVerbose).Changed && !cmd.Flag(cli.FlagJSON).Changed
	deleteOpts.JSON = cmd.Flag(cli.FlagJSON).Changed
}

func deleteRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), deleteCfg.Pretty)
		os.Exit(waitcmd.ExitFailed)
	}

	if err := waitcmd.ValidateWaitFlags(deleteOpts.Wait, cmd.Flags().Changed(cli.FlagBackupStorageTimeout), deleteOpts.Timeout); err != nil {
		output.PrintError(err, logger.GetLogger(), deleteCfg.Pretty)
		os.Exit(waitcmd.ExitFailed)
	}

	// Ctrl-C only cancels --wait; the cancellable context is set up inside
	// Deleter.waitForDeletion, not here.
	d := backupstoragecli.NewDeleter(*deleteCfg, logger.GetLogger())
	runErr := d.Run(cmd.Context(), *deleteOpts, cfgPath)
	os.Exit(waitcmd.ExitCode(runErr, logger.GetLogger(), deleteCfg.Pretty,
		fmt.Sprintf("wait cancelled; backup storage %q deletion continues in the background — check with 'everestctl backup-storage list --namespace %s'", deleteOpts.Name, deleteOpts.Namespace),
		fmt.Sprintf("timed out after %s waiting for backup storage %q to be deleted; deletion is still running server-side — a stuck delete is almost always an Instance or Backup that still references the storage, check 'everestctl instance list --namespace %s' / 'everestctl backup list --namespace %s'",
			deleteOpts.Timeout, deleteOpts.Name, deleteOpts.Namespace, deleteOpts.Namespace),
	))
}

// GetDeleteCmd returns the delete command.
func GetDeleteCmd() *cobra.Command {
	return deleteCmd
}
