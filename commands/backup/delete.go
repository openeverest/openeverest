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
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	backupcli "github.com/openeverest/openeverest/v2/pkg/cli/backup"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/cli/waitcmd"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	deleteCmd = &cobra.Command{
		Use:   "delete [flags]",
		Args:  cobra.NoArgs,
		Short: "Delete a backup",
		Long: `Delete a Backup through the Everest API.

The name and namespace flags are required. What deletion means is governed by
the backup's own spec.deletionPolicy: Delete (the default) removes both the
engine-native backup resource and its data in the BackupStorage; Retain
removes only the resource and leaves the data recoverable out-of-band. There
is no policy-override flag at delete time — change spec.deletionPolicy on the
Backup first if you need a different outcome for this delete.

The confirmation tier follows the effective policy: Delete asks you to type
the backup name back (irreversible data loss); Retain asks y/N (the data
survives). Pass --yes/-y to skip the prompt; in a non-interactive context (no
terminal, or --json) omitting --yes fails immediately instead of hanging on a
prompt nobody can answer.

If the backup is still Pending or Running, has no status yet, or is in the
Error state (the controller may still be retrying it), the command refuses
by default, since deleting an in-flight backup interrupts it mid-operation.
Pass --force to override the guard; forcing still shows the confirmation
prompt, it does not skip it, --force is orthogonal to --yes, scripted
force-deletes need both.

--ignore-not-found treats an already-absent backup as success and skips both
the confirmation prompt and --wait entirely, since there's nothing to confirm
or wait for. If --wait times out, cleanup (engine resource and, for policy
Delete, the storage artifact) is still running server-side behind a
finalizer.`,
		Example: `  # Delete a backup (policy Delete: type-the-name; policy Retain: y/N)
  everestctl backup delete --name pre-upgrade --namespace everest

  # Scripted retention pruning
  everestctl backup delete --name nightly-20260701 --namespace everest -y --ignore-not-found

  # Abort and delete a backup that is still running
  everestctl backup delete --name hung-backup --namespace everest -y --force

  # Wait until the artifact cleanup finished
  everestctl backup delete --name pre-upgrade --namespace everest -y --wait --timeout 10m`,
		PreRun: deletePreRun,
		Run:    deleteRun,
	}
	deleteCfg  = &backupcli.Config{}
	deleteOpts = &backupcli.DeleteOptions{}
)

func init() {
	deleteCmd.Flags().StringVar(&deleteOpts.Name, cli.FlagBackupName, "", "Backup name (required)")
	deleteCmd.Flags().StringVarP(&deleteOpts.Namespace, cli.FlagBackupNamespace, "n", "", "Namespace the backup is in (required)")
	deleteCmd.Flags().StringVar(&deleteOpts.Cluster, cli.FlagBackupCluster, "main", "Cluster name")
	deleteCmd.Flags().StringVar(&deleteOpts.Context, cli.FlagBackupContext, "", "Context to use (default: current context)")
	deleteCmd.Flags().BoolVarP(&deleteOpts.Yes, cli.FlagYes, "y", false, "Skip the confirmation prompt")
	deleteCmd.Flags().BoolVar(&deleteOpts.Force, cli.FlagBackupForce, false, "Override the in-flight (Pending/Running/Error/no status yet) delete guard; does not skip the confirmation prompt")
	deleteCmd.Flags().BoolVar(&deleteOpts.IgnoreNotFound, cli.FlagBackupIgnoreNotFound, false, "Treat \"backup already gone\" as a successful delete instead of an error")
	deleteCmd.Flags().BoolVar(&deleteOpts.Wait, cli.FlagBackupWait, false, "Block until the backup is fully deleted (Ctrl-C cancels only the wait, not the deletion)")
	deleteCmd.Flags().DurationVar(&deleteOpts.Timeout, cli.FlagBackupTimeout, 5*time.Minute, "Maximum time to wait (only valid with --wait); must be positive")

	_ = deleteCmd.MarkFlagRequired(cli.FlagBackupName)
	_ = deleteCmd.MarkFlagRequired(cli.FlagBackupNamespace)
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

	if err := waitcmd.ValidateWaitFlags(deleteOpts.Wait, cmd.Flags().Changed(cli.FlagBackupTimeout), deleteOpts.Timeout); err != nil {
		output.PrintError(err, logger.GetLogger(), deleteCfg.Pretty)
		os.Exit(waitcmd.ExitFailed)
	}

	// Ctrl-C only cancels --wait; the cancellable context is set up inside
	// Deleter.waitForDeletion, not here.
	bd := backupcli.NewDeleter(*deleteCfg, logger.GetLogger())
	runErr := bd.Run(cmd.Context(), *deleteOpts, cfgPath)
	os.Exit(waitcmd.ExitCode(runErr, logger.GetLogger(), deleteCfg.Pretty,
		fmt.Sprintf("wait cancelled; backup %q deletion continues in the background — check with 'everestctl backup list --namespace %s --instance <instance>'", deleteOpts.Name, deleteOpts.Namespace),
		fmt.Sprintf("timed out after %s waiting for backup %q to be deleted; cleanup is still running server-side, check with 'everestctl backup list --namespace %s --instance <instance>'",
			deleteOpts.Timeout, deleteOpts.Name, deleteOpts.Namespace),
	))
}

// GetDeleteCmd returns the delete command.
func GetDeleteCmd() *cobra.Command {
	return deleteCmd
}
