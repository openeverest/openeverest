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

	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	backupstoragecli "github.com/openeverest/openeverest/v2/pkg/cli/backupstorage"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	createCmd = &cobra.Command{
		Use:   "create [flags]",
		Args:  cobra.NoArgs,
		Short: "Create a backup storage",
		Long: `Create a BackupStorage resource via the Everest API.

A backup storage is a bucket, and optionally a --prefix within it, of a
BackupStorageClass: the object storage technology, "s3" by default. The
class and the location cannot change once the storage is created.

Connection parameters are validated by the class. Set them with
--parameter key=value (repeatable; true, false and numbers keep their JSON
type, anything else is a string). For the s3 class, --region,
--endpoint-url, --force-path-style and --implementation set the region,
endpointURL, forcePathStyle and implementation parameters.

Credentials live in a Secret in the same namespace that holds the keys of
the --auth-mode (for s3 and AccessKey: AWS_ACCESS_KEY_ID and
AWS_SECRET_ACCESS_KEY). Exactly one of:

  - --credentials-secret: reference an existing Secret.
  - --credentials-file: a file of KEY=VALUE lines (blank lines and #
    comments are ignored) the CLI creates the Secret "backup-storage-<name>-credentials"
    from. Values are never printed.

The server certificate is verified against the system trust store, unless
--ca-file (the CLI creates the Secret "backup-storage-<name>-ca" from a
PEM bundle),
--ca-secret NAME[:KEY] (an existing Secret; KEY defaults to ca.crt) or
--insecure-skip-tls-verify is given.

Secrets the CLI created are deleted again if the backup storage is not
created. The server checks the storage against its class afterwards: see
the ACCEPTED column of 'everestctl backup-storage list'.`,
		Example: `  # S3, credentials read from a KEY=VALUE file
  everestctl backup-storage create --name s3-prod -n team-a --bucket my-backups \
    --region us-east-1 --endpoint-url https://s3.us-east-1.amazonaws.com \
    --credentials-file ./s3.env

  # Reference an existing credentials Secret, write under a prefix
  everestctl backup-storage create --name s3-prod -n team-a --bucket my-backups \
    --prefix team-a/prod --region us-east-1 \
    --endpoint-url https://s3.us-east-1.amazonaws.com \
    --credentials-secret s3-prod-credentials

  # MinIO with a private CA, parameters spelled out
  everestctl bs create --name minio -n team-a --bucket backups \
    --parameter region=us-east-1 --parameter endpointURL=https://minio.minio.svc:9000 \
    --parameter forcePathStyle=true --parameter implementation=minio \
    --credentials-file ./minio.env --ca-file ./minio-ca.pem

  # Specific cluster and context
  everestctl backup-storage create --name s3-prod -n team-a --bucket my-backups \
    --region us-east-1 --endpoint-url https://s3.us-east-1.amazonaws.com \
    --credentials-secret s3-prod-credentials --cluster staging --context staging-ctx`,
		PreRun: createPreRun,
		Run:    createRun,
	}
	createCfg  = &backupstoragecli.Config{}
	createOpts = &backupstoragecli.CreateOptions{}
)

func init() {
	createCmd.Flags().StringVar(&createOpts.Name, cli.FlagBackupStorageName, "", "Backup storage name (required)")
	createCmd.Flags().StringVarP(&createOpts.Namespace, cli.FlagBackupStorageNamespace, "n", "", "Namespace to create the backup storage in (required)")
	createCmd.Flags().StringVar(&createOpts.Class, cli.FlagBackupStorageClass, storageclasses.ClassS3, "BackupStorageClass, the object storage technology")
	createCmd.Flags().StringVar(&createOpts.Bucket, cli.FlagBackupStorageBucket, "", "Bucket name (required)")
	createCmd.Flags().StringVar(&createOpts.Prefix, cli.FlagBackupStoragePrefix, "", "Path within the bucket to write backups under, without leading or trailing slash")
	addParameterFlags(createCmd, &createOpts.Parameters)
	createCmd.Flags().StringVar(&createOpts.AuthMode, cli.FlagBackupStorageAuthMode, storageclasses.AuthModeAccessKey, "Authentication mode, one of the class's authModes")
	createCmd.Flags().StringVar(&createOpts.CredentialsSecret, cli.FlagBackupStorageCredentialsSecret, "", "Name of an existing Secret holding the credentials")
	createCmd.Flags().StringVar(&createOpts.CredentialsFile, cli.FlagBackupStorageCredentialsFile, "", "File of KEY=VALUE lines to create the Secret backup-storage-<name>-credentials from")
	createCmd.Flags().StringVar(&createOpts.CAFile, cli.FlagBackupStorageCAFile, "", "PEM CA bundle to create the Secret backup-storage-<name>-ca from")
	createCmd.Flags().StringVar(&createOpts.CASecret, cli.FlagBackupStorageCASecret, "", "Existing Secret holding the PEM CA bundle, as NAME[:KEY] (KEY defaults to ca.crt)")
	createCmd.Flags().BoolVar(&createOpts.InsecureSkipTLSVerify, cli.FlagBackupStorageInsecureSkipTLSVerify, false, "Do not verify the server certificate of the store")
	createCmd.Flags().StringVar(&createOpts.Cluster, cli.FlagBackupStorageCluster, "main", "Cluster name")
	createCmd.Flags().StringVar(&createOpts.Context, cli.FlagBackupStorageContext, "", "Context to use (default: current context)")

	_ = createCmd.MarkFlagRequired(cli.FlagBackupStorageName)
	_ = createCmd.MarkFlagRequired(cli.FlagBackupStorageNamespace)
	_ = createCmd.MarkFlagRequired(cli.FlagBackupStorageBucket)
}

func createPreRun(cmd *cobra.Command, _ []string) {
	createCfg.Pretty = !cmd.Flag(cli.FlagVerbose).Changed && !cmd.Flag(cli.FlagJSON).Changed
}

func createRun(cmd *cobra.Command, _ []string) {
	readParameterShorthands(cmd, &createOpts.Parameters)

	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(1)
	}

	runner := backupstoragecli.NewCreateRunner(*createCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *createOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(1)
	}
}

// GetCreateCmd returns the create command.
func GetCreateCmd() *cobra.Command {
	return createCmd
}
