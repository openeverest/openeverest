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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
deleted by this command directly — it's owned by the BackupStorage and is
garbage-collected by Kubernetes once the BackupStorage is actually gone,
unless it was externally referenced before being adopted.

Interactively (a terminal, no --yes), you'll be asked to confirm with y/N.
Pass --yes/-y to skip the prompt; in a non-interactive context (no terminal,
or --json) omitting --yes fails immediately instead of hanging on a prompt
nobody can answer.`,
		Example: `  # Delete a backup storage (prompts y/N)
  everestctl backup-storage delete --name my-s3 --namespace everest

  # Scripted
  everestctl backup-storage delete --name my-s3 --namespace everest -y

  # Scripted teardown, wait until fully removed
  everestctl backup-storage delete --name my-s3 --namespace everest --yes --wait --timeout 5m`,
		PreRun: deletePreRun,
		Run:    deleteRun,
	}
	deleteCfg  = &backupstoragecli.Config{}
	deleteOpts = &backupstoragecli.DeleteOptions{}
)

func init() {
	deleteCmd.Flags().StringVar(&deleteOpts.Name, cli.FlagBackupStorageName, "", "Backup storage name (required)")
	deleteCmd.Flags().StringVar(&deleteOpts.Namespace, cli.FlagBackupStorageNamespace, "", "Namespace the backup storage is in (required)")
	deleteCmd.Flags().StringVar(&deleteOpts.Cluster, cli.FlagBackupStorageCluster, "main", "Cluster name")
	deleteCmd.Flags().StringVar(&deleteOpts.Context, cli.FlagBackupStorageContext, "", "Context to use (default: current context)")
	deleteCmd.Flags().BoolVarP(&deleteOpts.Yes, cli.FlagYes, "y", false, "Skip the confirmation prompt")
	deleteCmd.Flags().BoolVar(&deleteOpts.Wait, cli.FlagBackupStorageWait, false, "Block until the backup storage is fully deleted (Ctrl-C cancels only the wait, not the deletion)")
	deleteCmd.Flags().DurationVar(&deleteOpts.Timeout, cli.FlagBackupStorageTimeout, 5*time.Minute, "Maximum time to wait (only valid with --wait); must be positive")

	_ = deleteCmd.MarkFlagRequired(cli.FlagBackupStorageName)
	_ = deleteCmd.MarkFlagRequired(cli.FlagBackupStorageNamespace)
}

func deletePreRun(cmd *cobra.Command, _ []string) {
	deleteCfg.Pretty = !cmd.Flag(cli.FlagVerbose).Changed && !cmd.Flag(cli.FlagJSON).Changed
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

	ctx := cmd.Context()
	stop := func() {}
	if deleteOpts.Wait {
		// Ctrl-C cancels only the wait; deletion continues server-side.
		var cancel context.CancelFunc
		ctx, cancel = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		stop = cancel
	}

	d := backupstoragecli.NewDeleter(*deleteCfg, logger.GetLogger())
	runErr := d.Run(ctx, *deleteOpts, cfgPath)
	// Release the signal handler before os.Exit (which skips defers).
	stop()
	os.Exit(waitcmd.ExitCode(runErr, logger.GetLogger(), deleteCfg.Pretty,
		fmt.Sprintf("wait cancelled; backup storage %q deletion continues in the background", deleteOpts.Name),
		fmt.Sprintf("timed out after %s waiting for backup storage %q to be deleted — it may still be referenced by an Instance or Backup", deleteOpts.Timeout, deleteOpts.Name),
	))
}

// GetDeleteCmd returns the delete command.
func GetDeleteCmd() *cobra.Command {
	return deleteCmd
}
