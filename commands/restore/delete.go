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
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	restorecli "github.com/openeverest/openeverest/v2/pkg/cli/restore"
	"github.com/openeverest/openeverest/v2/pkg/cli/waitcmd"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	deleteCmd = &cobra.Command{
		Use:   "delete [flags]",
		Args:  cobra.NoArgs,
		Short: "Delete a restore",
		Long: `Delete a Restore record through the Everest API.

The name and namespace flags are required. Deleting a Restore only removes
the history record; no stored backup data is touched.

Interactively (a terminal, no --yes), you'll be asked to confirm with y/N.
Pass --yes/-y to skip the prompt; in a non-interactive context (no terminal,
or --json) omitting --yes fails immediately instead of hanging on a prompt
nobody can answer.

If the restore is still Pending or Running, or hasn't reported a status
yet, the command refuses by default, since deleting an in-flight restore
interrupts the provider mid-operation. Pass --force to override the guard;
forcing still shows the confirmation prompt (with an added warning), it
does not skip it — --force is orthogonal to --yes, scripted force-deletes
need both. A restore in the Error state is never guarded: deleting a stuck
restore is the normal way to clear it.

--ignore-not-found treats an already-absent restore as success and skips
both the confirmation prompt and --wait entirely, since there's nothing to
confirm or wait for.`,
		Example: `  # Delete a finished restore record (prompts y/N)
  everestctl restore delete --name my-mongo-restore-x7k2q --namespace everest

  # Scripted cleanup of finished restores
  everestctl restore delete --name my-mongo-restore-x7k2q --namespace everest -y --ignore-not-found

  # Abort and delete a restore that is still running
  everestctl restore delete --name my-mongo-restore-x7k2q --namespace everest -y --force

  # Scripted teardown, wait until fully removed
  everestctl restore delete --name my-mongo-restore-x7k2q --namespace everest --yes --wait --timeout 5m`,
		PreRun: deletePreRun,
		Run:    deleteRun,
	}
	deleteCfg  = &restorecli.Config{}
	deleteOpts = &restorecli.DeleteOptions{}
)

func init() {
	deleteCmd.Flags().StringVar(&deleteOpts.Name, cli.FlagRestoreName, "", "Restore name (required)")
	deleteCmd.Flags().StringVarP(&deleteOpts.Namespace, cli.FlagRestoreNamespace, "n", "", "Namespace the restore is in (required)")
	deleteCmd.Flags().StringVar(&deleteOpts.Cluster, cli.FlagRestoreCluster, "main", "Cluster name")
	deleteCmd.Flags().StringVar(&deleteOpts.Context, cli.FlagRestoreContext, "", "Context to use (default: current context)")
	deleteCmd.Flags().BoolVarP(&deleteOpts.Yes, cli.FlagYes, "y", false, "Skip the confirmation prompt")
	deleteCmd.Flags().BoolVar(&deleteOpts.Force, cli.FlagRestoreForce, false, "Override the in-flight (Pending/Running/no status yet) delete guard; does not skip the confirmation prompt")
	deleteCmd.Flags().BoolVar(&deleteOpts.IgnoreNotFound, cli.FlagRestoreIgnoreNotFound, false, "Treat \"restore already gone\" as a successful delete instead of an error")
	deleteCmd.Flags().BoolVar(&deleteOpts.Wait, cli.FlagRestoreWait, false, "Block until the restore is fully deleted (Ctrl-C cancels only the wait, not the deletion)")
	deleteCmd.Flags().DurationVar(&deleteOpts.Timeout, cli.FlagRestoreTimeout, 5*time.Minute, "Maximum time to wait (only valid with --wait); must be positive")

	_ = deleteCmd.MarkFlagRequired(cli.FlagRestoreName)
	_ = deleteCmd.MarkFlagRequired(cli.FlagRestoreNamespace)
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

	if err := waitcmd.ValidateWaitFlags(deleteOpts.Wait, cmd.Flags().Changed(cli.FlagRestoreTimeout), deleteOpts.Timeout); err != nil {
		output.PrintError(err, logger.GetLogger(), deleteCfg.Pretty)
		os.Exit(waitcmd.ExitFailed)
	}

	// Ctrl-C only cancels --wait; the cancellable context is set up inside
	// Deleter.waitForDeletion, not here.
	rd := restorecli.NewDeleter(*deleteCfg, logger.GetLogger())
	runErr := rd.Run(cmd.Context(), *deleteOpts, cfgPath)
	os.Exit(waitcmd.ExitCode(runErr, logger.GetLogger(), deleteCfg.Pretty,
		fmt.Sprintf("wait cancelled; restore %q deletion continues in the background — check with 'everestctl restore list --namespace %s --instance <instance>'", deleteOpts.Name, deleteOpts.Namespace),
		fmt.Sprintf("timed out after %s waiting for restore %q to be deleted; deletion is still running server-side, check with 'everestctl restore list --namespace %s --instance <instance>'",
			deleteOpts.Timeout, deleteOpts.Name, deleteOpts.Namespace),
	))
}

// GetDeleteCmd returns the delete command.
func GetDeleteCmd() *cobra.Command {
	return deleteCmd
}
