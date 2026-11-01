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
		Long: `Patch an existing BackupStorage's safe-to-edit fields via the Everest API.

The name and namespace flags are required, plus at least one of:

  - --access-key-id, with the secret access key supplied the same way as
    'backup-storage create' (the EVEREST_BACKUP_STORAGE_SECRET_ACCESS_KEY
    environment variable, or an interactive prompt).
  - --credentials-secret, to re-point at a different existing Secret.
    Mutually exclusive with --access-key-id.
  - --verify-tls / --force-path-style, to change connection settings.

The identity fields (--type, --bucket, --region, --endpoint-url) are
deliberately not exposed here, even though the API does not itself reject a
change to them: re-pointing a storage that already holds backups orphans
every Backup CR referencing it by name and every engine already registered
against the old coordinates. Re-homing a storage is create-new + delete-old,
not an update.`,
		Example: `  # Rotate credentials
  everestctl backup-storage update --name my-s3 --namespace everest \
    --access-key-id AKIA...

  # Point at an externally managed Secret instead
  everestctl backup-storage update --name my-s3 --namespace everest \
    --credentials-secret my-rotated-creds

  # Toggle TLS verification for a self-signed MinIO
  everestctl bs update --name my-minio --namespace everest --verify-tls=false`,
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
	updateCmd.Flags().StringVar(&updateOpts.CredentialsSecret, cli.FlagBackupStorageCredentialsSecret, "", "Name of an existing Secret holding the storage credentials")
	updateCmd.Flags().StringVar(&updateOpts.AccessKeyID, cli.FlagBackupStorageAccessKeyID, "", "Access key ID (the secret access key is never passed as a flag — see --help)")
	updateCmd.Flags().Bool(cli.FlagBackupStorageVerifyTLS, false, "Verify TLS certificates when connecting to the endpoint")
	updateCmd.Flags().Bool(cli.FlagBackupStorageForcePathStyle, false, "Force path-style storage URLs (bucket name in the path instead of the host)")

	_ = updateCmd.MarkFlagRequired(cli.FlagBackupStorageName)
	_ = updateCmd.MarkFlagRequired(cli.FlagBackupStorageNamespace)

	updateCmd.MarkFlagsMutuallyExclusive(cli.FlagBackupStorageCredentialsSecret, cli.FlagBackupStorageAccessKeyID)
}

func updatePreRun(cmd *cobra.Command, _ []string) {
	updateCfg.Pretty = !cmd.Flag(cli.FlagVerbose).Changed && !cmd.Flag(cli.FlagJSON).Changed
}

// updateRun reads --verify-tls/--force-path-style through cmd.Flags().Changed
// rather than their bound values: unlike `create`, an update must not name a
// field in the patch that the caller never passed.
func updateRun(cmd *cobra.Command, _ []string) {
	if cmd.Flags().Changed(cli.FlagBackupStorageVerifyTLS) {
		v, _ := cmd.Flags().GetBool(cli.FlagBackupStorageVerifyTLS)
		updateOpts.VerifyTLS = &v
	}
	if cmd.Flags().Changed(cli.FlagBackupStorageForcePathStyle) {
		v, _ := cmd.Flags().GetBool(cli.FlagBackupStorageForcePathStyle)
		updateOpts.ForcePathStyle = &v
	}

	if updateOpts.AccessKeyID != "" {
		secret, err := resolveSecretAccessKey(cmd)
		if err != nil {
			output.PrintError(err, logger.GetLogger(), updateCfg.Pretty)
			os.Exit(1)
		}
		updateOpts.SecretAccessKey = secret
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
