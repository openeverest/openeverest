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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
	createCmd = &cobra.Command{
		Use:   "create [flags]",
		Args:  cobra.NoArgs,
		Short: "Restore an instance from a backup",
		Long: `Create a Restore for an instance from an existing Backup via the Everest API.

The instance, namespace, and backup flags are required. The backup must be an
existing Backup CR in the same namespace. Only full-backup restores are
supported; point-in-time recovery is a follow-up. When --name is omitted, the
server generates a name from the instance name plus a random suffix
(metadata.generateName). Use --wait to block until the restore reaches a
terminal state (Succeeded or Failed); --timeout bounds how long --wait blocks
for. Ctrl-C during --wait stops the CLI but leaves the restore running
server-side.`,
		Example: `  # Restore an instance from a backup
  everestctl restore create --instance my-mongo --namespace everest \
    --backup pre-upgrade

  # Wait for the restore to finish, bounded for CI
  everestctl restore create --instance my-mongo --namespace everest \
    --backup nightly-20260727 --wait --timeout 30m

  # Specific cluster and context
  everestctl restore create --instance my-mongo --namespace everest \
    --backup pre-upgrade --cluster staging --context staging-ctx

  # Bounded wait for CI, then grab the state (Ctrl-C only stops waiting; the restore keeps running)
  everestctl restore create --instance my-mongo --namespace everest \
    --backup pre-upgrade --wait --timeout 15m --json | jq '.status.state'`,
		PreRun: createPreRun,
		Run:    createRun,
	}
	createCfg  = &restorecli.Config{}
	createOpts = &restorecli.CreateOptions{}
)

func init() {
	createCmd.Flags().StringVar(&createOpts.Instance, cli.FlagRestoreInstance, "", "Instance to restore into (required)")
	createCmd.Flags().StringVarP(&createOpts.Namespace, cli.FlagRestoreNamespace, "n", "", "Namespace the instance is located in (required)")
	createCmd.Flags().StringVar(&createOpts.Backup, cli.FlagRestoreBackup, "", "Backup to restore from (required)")
	createCmd.Flags().StringVar(&createOpts.Name, cli.FlagRestoreName, "", "Restore name (default: generated from instance name)")
	createCmd.Flags().StringVar(&createOpts.Cluster, cli.FlagRestoreCluster, "main", "Cluster name")
	createCmd.Flags().StringVar(&createOpts.Context, cli.FlagRestoreContext, "", "Context to use (default: current context)")
	createCmd.Flags().BoolVar(&createOpts.Wait, cli.FlagRestoreWait, false, "Block until the restore reaches a terminal state; exit non-zero if it fails or times out")
	createCmd.Flags().DurationVar(&createOpts.Timeout, cli.FlagRestoreTimeout, 30*time.Minute, "Maximum time to wait (only valid with --wait); must be positive")

	_ = createCmd.MarkFlagRequired(cli.FlagRestoreInstance)
	_ = createCmd.MarkFlagRequired(cli.FlagRestoreNamespace)
	_ = createCmd.MarkFlagRequired(cli.FlagRestoreBackup)
}

func createPreRun(cmd *cobra.Command, _ []string) {
	createCfg.Pretty = !cmd.Flag(cli.FlagVerbose).Changed && !cmd.Flag(cli.FlagJSON).Changed
}

func createRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(waitcmd.ExitFailed)
	}

	if err := waitcmd.ValidateWaitFlags(createOpts.Wait, cmd.Flags().Changed(cli.FlagRestoreTimeout), createOpts.Timeout); err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(waitcmd.ExitFailed)
	}

	ctx := cmd.Context()
	var stop context.CancelFunc
	if createOpts.Wait {
		// Ctrl-C cancels only the wait; the restore keeps running server-side.
		ctx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	}

	runner := restorecli.NewCreateRunner(*createCfg, logger.GetLogger())
	runErr := runner.Run(ctx, *createOpts, cfgPath)
	if stop != nil {
		stop()
	}
	// Name omitted from the cancel message: the runner already printed it
	// before waiting began.
	os.Exit(waitcmd.ExitCode(runErr, logger.GetLogger(), createCfg.Pretty,
		"wait cancelled; the restore is still running — check with 'everestctl restore list'",
		fmt.Sprintf("timed out after %s waiting for restore to complete", createOpts.Timeout),
	))
}

// GetCreateCmd returns the create command.
func GetCreateCmd() *cobra.Command {
	return createCmd
}
