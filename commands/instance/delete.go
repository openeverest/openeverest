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

package instance

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	instancecli "github.com/openeverest/openeverest/v2/pkg/cli/instance"
	"github.com/openeverest/openeverest/v2/pkg/cli/waitcmd"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	deleteCmd = &cobra.Command{
		Use:   "delete [flags]",
		Args:  cobra.NoArgs,
		Short: "Delete an instance",
		Long: `Delete an instance through the Everest API.

The name and namespace flags are required. What happens to dependent Backup
and Restore objects is governed by the instance's own spec.deletionPolicy
(Cascade deletes them, Orphan leaves them behind) unless overridden for this
delete with --deletion-policy.

Interactively (a terminal, no --yes), you'll be asked to type the instance
name back to confirm — this is irreversible. Pass --yes/-y to skip the
prompt; in a non-interactive context (no terminal, or --json) omitting --yes
fails immediately instead of hanging on a prompt nobody can answer.

--ignore-not-found treats an already-absent instance as success and skips
both the confirmation prompt and --wait entirely, since there's nothing to
confirm or wait for. If --wait times out, the deletion is still running
server-side, and a stuck cascade is almost always a Backup/Restore that
won't finalize, so point at 'everestctl backup list'/'everestctl restore
list' for the namespace.`,
		Example: `  # Delete an instance (states blast radius, asks to type the name)
  everestctl instance delete --name my-mongo --namespace everest

  # Keep backups around despite the instance's Cascade policy
  everestctl instance delete --name my-mongo --namespace everest --deletion-policy Orphan

  # Scripted teardown, skip the prompt, wait until fully removed
  everestctl instance delete --name ci-run-421 --namespace ci --yes --wait --timeout 10m

  # Idempotent teardown: treat "already gone" as success
  everestctl instance delete --name ci-run-421 --namespace ci --yes --ignore-not-found`,
		PreRun: deletePreRun,
		Run:    deleteRun,
	}
	deleteCfg  = &instancecli.Config{}
	deleteOpts = &instancecli.DeleteOptions{}
)

func init() {
	deleteCmd.Flags().StringVar(&deleteOpts.Name, cli.FlagInstanceName, "", "Instance name (required)")
	deleteCmd.Flags().StringVar(&deleteOpts.Namespace, cli.FlagInstanceNamespace, "", "Namespace the instance is in (required)")
	deleteCmd.Flags().StringVar(&deleteOpts.Cluster, cli.FlagInstanceCluster, "main", "Cluster name")
	deleteCmd.Flags().StringVar(&deleteOpts.Context, cli.FlagInstanceContext, "", "Context to use (default: current context)")
	deleteCmd.Flags().StringVar(&deleteOpts.DeletionPolicy, cli.FlagInstanceDeletionPolicy, "",
		"Override the instance's deletion policy for this delete: Cascade or Orphan (default: the instance's own spec.deletionPolicy)")
	deleteCmd.Flags().BoolVarP(&deleteOpts.Yes, cli.FlagYes, "y", false, "Skip the confirmation prompt")
	deleteCmd.Flags().BoolVar(&deleteOpts.IgnoreNotFound, cli.FlagInstanceIgnoreNotFound, false, "Treat \"instance already gone\" as a successful delete instead of an error")
	deleteCmd.Flags().BoolVar(&deleteOpts.Wait, cli.FlagInstanceWait, false, "Block until the instance is fully deleted (Ctrl-C cancels only the wait, not the deletion)")
	deleteCmd.Flags().DurationVar(&deleteOpts.Timeout, cli.FlagInstanceTimeout, 10*time.Minute, "Maximum time to wait (only valid with --wait); must be positive")

	_ = deleteCmd.MarkFlagRequired(cli.FlagInstanceName)
	_ = deleteCmd.MarkFlagRequired(cli.FlagInstanceNamespace)
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

	if err := waitcmd.ValidateWaitFlags(deleteOpts.Wait, cmd.Flags().Changed(cli.FlagInstanceTimeout), deleteOpts.Timeout); err != nil {
		output.PrintError(err, logger.GetLogger(), deleteCfg.Pretty)
		os.Exit(waitcmd.ExitFailed)
	}

	// Ctrl-C only cancels --wait; the cancellable context is set up inside
	// Deleter.waitForDeletion, not here.
	id := instancecli.NewDeleter(*deleteCfg, logger.GetLogger())
	runErr := id.Run(cmd.Context(), *deleteOpts, cfgPath)
	os.Exit(waitcmd.ExitCode(runErr, logger.GetLogger(), deleteCfg.Pretty,
		fmt.Sprintf("wait cancelled; instance %q deletion continues in the background — check with 'everestctl instance status'", deleteOpts.Name),
		fmt.Sprintf("timed out after %s waiting for instance %q to be deleted; deletion is still running server-side — a stuck cascade is almost always a Backup or Restore that won't finalize, check 'everestctl backup list --namespace %s' / 'everestctl restore list --namespace %s'",
			deleteOpts.Timeout, deleteOpts.Name, deleteOpts.Namespace, deleteOpts.Namespace),
	))
}

// GetDeleteCmd returns the delete command.
func GetDeleteCmd() *cobra.Command {
	return deleteCmd
}
