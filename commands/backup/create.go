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

// Package backup holds commands for backup management.
package backup

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
	createCmd = &cobra.Command{
		Use:   "create [flags]",
		Args:  cobra.NoArgs,
		Short: "Create an on-demand backup",
		Long: `Create an on-demand Backup for an instance via the Everest API.

The instance, namespace, class, and storage flags are required. When --name
is omitted, the server generates a name from the instance name plus a random
suffix (metadata.generateName). Use --wait to block until the backup reaches
a terminal state (Succeeded or Failed); --timeout bounds how long --wait
blocks for. Ctrl-C during --wait stops the CLI but leaves the backup running
server-side.`,
		Example: `  # Take an on-demand backup
  everestctl backup create --instance my-mongo --namespace everest \
    --class percona-backup-mongodb --storage my-s3

  # With explicit name and wait for completion
  everestctl backup create --instance my-mongo --namespace everest \
    --class percona-backup-mongodb --storage my-s3 --name pre-upgrade --wait

  # Wait with a custom timeout
  everestctl backup create --instance my-mongo --namespace everest \
    --class percona-backup-mongodb --storage my-s3 --wait --timeout 10m

  # Retain the underlying data if the Backup CR is later deleted
  everestctl backup create --instance my-mongo --namespace everest \
    --class percona-backup-mongodb --storage my-s3 --deletion-policy Retain

  # Specific cluster and context
  everestctl backup create --instance my-mongo --namespace everest \
    --class percona-backup-mongodb --storage my-s3 --cluster staging --context staging-ctx

  # Bounded wait for CI, then grab the state (Ctrl-C only stops waiting; the backup keeps running)
  everestctl backup create --instance my-mongo --namespace everest \
    --class percona-backup-mongodb --storage my-s3 --wait --timeout 15m --json | jq '.status.state'`,
		PreRun: createPreRun,
		Run:    createRun,
	}
	createCfg  = &backupcli.Config{}
	createOpts = &backupcli.CreateOptions{}
)

func init() {
	createCmd.Flags().StringVar(&createOpts.Instance, cli.FlagBackupInstance, "", "Instance to back up (required)")
	createCmd.Flags().StringVarP(&createOpts.Namespace, cli.FlagBackupNamespace, "n", "", "Namespace the instance is located in (required)")
	createCmd.Flags().StringVar(&createOpts.Class, cli.FlagBackupClass, "", "BackupClass to execute the backup with (required)")
	createCmd.Flags().StringVar(&createOpts.Storage, cli.FlagBackupStorage, "", "BackupStorage to write the backup to (required)")
	createCmd.Flags().StringVar(&createOpts.Name, cli.FlagBackupName, "", "Backup name (default: generated from instance name)")
	createCmd.Flags().StringVar(&createOpts.DeletionPolicy, cli.FlagBackupDeletionPolicy, "", "Delete or Retain the underlying backup data when the Backup CR is deleted (default: Delete)")
	createCmd.Flags().StringVar(&createOpts.Cluster, cli.FlagBackupCluster, "main", "Cluster name")
	createCmd.Flags().StringVar(&createOpts.Context, cli.FlagBackupContext, "", "Context to use (default: current context)")
	createCmd.Flags().BoolVar(&createOpts.Wait, cli.FlagBackupWait, false, "Block until the backup reaches a terminal state; exit non-zero if it fails or times out")
	createCmd.Flags().DurationVar(&createOpts.Timeout, cli.FlagBackupTimeout, 30*time.Minute, "Maximum time to wait (only valid with --wait); must be positive")

	_ = createCmd.MarkFlagRequired(cli.FlagBackupInstance)
	_ = createCmd.MarkFlagRequired(cli.FlagBackupNamespace)
	_ = createCmd.MarkFlagRequired(cli.FlagBackupClass)
	_ = createCmd.MarkFlagRequired(cli.FlagBackupStorage)
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

	if err := waitcmd.ValidateWaitFlags(createOpts.Wait, cmd.Flags().Changed(cli.FlagBackupTimeout), createOpts.Timeout); err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(waitcmd.ExitFailed)
	}

	ctx := cmd.Context()
	var stop context.CancelFunc
	if createOpts.Wait {
		// Ctrl-C cancels only the wait; the backup keeps running server-side.
		ctx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	}

	runner := backupcli.NewCreateRunner(*createCfg, logger.GetLogger())
	runErr := runner.Run(ctx, *createOpts, cfgPath)
	if stop != nil {
		stop()
	}
	// Name omitted from the cancel message: the runner already printed it
	// before waiting began.
	os.Exit(waitcmd.ExitCode(runErr, logger.GetLogger(), createCfg.Pretty,
		"wait cancelled; the backup is still running — check with 'everestctl backup list'",
		fmt.Sprintf("timed out after %s waiting for backup to complete", createOpts.Timeout),
	))
}

// GetCreateCmd returns the create command.
func GetCreateCmd() *cobra.Command {
	return createCmd
}
