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
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	backupstoragecli "github.com/openeverest/openeverest/v2/pkg/cli/backupstorage"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	updateCmd = &cobra.Command{
		Use:   "update [flags]",
		Args:  cobra.NoArgs,
		Short: "Update a backup storage",
		Long: `Patch an existing BackupStorage via the Everest API.

The name and namespace flags are required, plus at least one of:

  - --parameter key=value (repeatable), or the s3 shorthands --region,
    --endpoint-url, --force-path-style and --implementation. Parameters
    merge key by key: the ones not named keep their value.
  - --auth-mode, to switch to another of the class's authentication modes.
  - --credentials-secret, to point at another existing Secret in the
    namespace. This is how credentials are rotated: create a Secret with
    the new keys, then switch the storage to it.
  - --ca-secret NAME[:KEY] or --insecure-skip-tls-verify, to change how
    the server certificate is verified. Setting one clears the other.

The class and the location (bucket and prefix) cannot change: engines
treat the location as their catalog. Re-homing a storage is create-new +
delete-old, not an update.`,
		Example: `  # Rotate credentials: point at a Secret holding the new keys
  everestctl backup-storage update --name s3-prod -n team-a \
    --credentials-secret s3-prod-credentials-2

  # Move to another endpoint and region
  everestctl backup-storage update --name s3-prod -n team-a \
    --endpoint-url https://s3.eu-west-1.amazonaws.com --region eu-west-1

  # Trust a private CA
  everestctl bs update --name minio -n team-a --ca-secret minio-ca:ca.crt`,
		PreRun: updatePreRun,
		Run:    updateRun,
	}
	updateCfg  = &backupstoragecli.Config{}
	updateOpts = &backupstoragecli.UpdateOptions{}
)

func init() {
	updateCmd.Flags().StringVar(&updateOpts.Name, cli.FlagBackupStorageName, "", "Backup storage name (required)")
	updateCmd.Flags().StringVarP(&updateOpts.Namespace, cli.FlagBackupStorageNamespace, "n", "", "Namespace the backup storage is in (required)")
	updateCmd.Flags().StringVar(&updateOpts.Cluster, cli.FlagBackupStorageCluster, "main", "Cluster name")
	updateCmd.Flags().StringVar(&updateOpts.Context, cli.FlagBackupStorageContext, "", "Context to use (default: current context)")
	addParameterFlags(updateCmd, &updateOpts.Parameters)
	updateCmd.Flags().StringVar(&updateOpts.AuthMode, cli.FlagBackupStorageAuthMode, "", "Authentication mode, one of the class's authModes")
	updateCmd.Flags().StringVar(&updateOpts.CredentialsSecret, cli.FlagBackupStorageCredentialsSecret, "", "Name of an existing Secret holding the credentials")
	updateCmd.Flags().StringVar(&updateOpts.CASecret, cli.FlagBackupStorageCASecret, "", "Existing Secret holding the PEM CA bundle, as NAME[:KEY] (KEY defaults to ca.crt)")
	updateCmd.Flags().Bool(cli.FlagBackupStorageInsecureSkipTLSVerify, false, "Do not verify the server certificate of the store")

	_ = updateCmd.MarkFlagRequired(cli.FlagBackupStorageName)
	_ = updateCmd.MarkFlagRequired(cli.FlagBackupStorageNamespace)
}

func updatePreRun(cmd *cobra.Command, _ []string) {
	updateCfg.Pretty = !cmd.Flag(cli.FlagVerbose).Changed && !cmd.Flag(cli.FlagJSON).Changed
}

// updateRun reads --insecure-skip-tls-verify and the parameter shorthands
// through cmd.Flags().Changed rather than their bound values: an update must
// not name a field in the patch that the caller never passed.
func updateRun(cmd *cobra.Command, _ []string) {
	readParameterShorthands(cmd, &updateOpts.Parameters)
	if cmd.Flags().Changed(cli.FlagBackupStorageInsecureSkipTLSVerify) {
		v, _ := cmd.Flags().GetBool(cli.FlagBackupStorageInsecureSkipTLSVerify)
		updateOpts.InsecureSkipTLSVerify = &v
	}

	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), updateCfg.Pretty)
		os.Exit(1)
	}

	u := backupstoragecli.NewUpdater(*updateCfg, logger.GetLogger())
	if err := u.Run(cmd.Context(), *updateOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), updateCfg.Pretty)
		os.Exit(1)
	}
}

// GetUpdateCmd returns the update command.
func GetUpdateCmd() *cobra.Command {
	return updateCmd
}
