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
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	backupstoragecli "github.com/openeverest/openeverest/v2/pkg/cli/backupstorage"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/cli/tui"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// envSecretAccessKey is the environment variable checked for the secret access key
// when --access-key-id is used, before falling back to an interactive prompt.
const envSecretAccessKey = "EVEREST_BACKUP_STORAGE_SECRET_ACCESS_KEY"

var (
	createCmd = &cobra.Command{
		Use:   "create [flags]",
		Args:  cobra.NoArgs,
		Short: "Create a backup storage",
		Long: `Create a BackupStorage resource via the Everest API.

Only "s3" is supported as --type today. Provide credentials one of two ways:

  - --credentials-secret: reference an existing Secret (in the same
    namespace) that already holds the storage credentials.
  - --access-key-id plus a secret access key supplied via the
    EVEREST_BACKUP_STORAGE_SECRET_ACCESS_KEY environment variable, or an
    interactive prompt if the env var is unset and stdin is a terminal
    (skipped for --json, or when stdin is piped/redirected, e.g. in CI).
    The secret access key is never accepted as a plain flag. The CLI
    points credentialsSecretRef at "backup-storage-<name>-credentials"
    (the same name the UI uses); the server writes the credentials into
    that Secret and never persists them on the BackupStorage object.

These two modes are mutually exclusive; exactly one is required.`,
		Example: `  # Reference an existing credentials Secret
  everestctl backup-storage create --name my-s3 --namespace everest \
    --type s3 --bucket my-bucket --region us-east-1 \
    --endpoint-url https://s3.amazonaws.com --credentials-secret my-s3-creds

  # Provide an access key id; secret access key via env var
  EVEREST_BACKUP_STORAGE_SECRET_ACCESS_KEY=*** everestctl bs create \
    --name my-s3 --namespace everest --type s3 --bucket my-bucket \
    --region us-east-1 --endpoint-url https://s3.amazonaws.com \
    --access-key-id AKIA...

  # Provide an access key id; interactive prompt for the secret
  everestctl backup-storage create --name my-s3 --namespace everest \
    --type s3 --bucket my-bucket --region us-east-1 \
    --endpoint-url https://s3.amazonaws.com --access-key-id AKIA...

  # Path-style URLs, TLS verification disabled (e.g. a local MinIO)
  everestctl backup-storage create --name minio --namespace everest \
    --type s3 --bucket backups --region us-east-1 \
    --endpoint-url http://minio.everest.svc:9000 \
    --force-path-style --verify-tls=false --credentials-secret minio-creds

  # Specific cluster and context
  everestctl backup-storage create --name my-s3 --namespace everest \
    --type s3 --bucket my-bucket --region us-east-1 \
    --endpoint-url https://s3.amazonaws.com --credentials-secret my-s3-creds \
    --cluster staging --context staging-ctx`,
		PreRun: createPreRun,
		Run:    createRun,
	}
	createCfg  = &backupstoragecli.Config{}
	createOpts = &backupstoragecli.CreateOptions{}
)

func init() {
	createCmd.Flags().StringVar(&createOpts.Name, cli.FlagBackupStorageName, "", "Backup storage name (required)")
	createCmd.Flags().StringVarP(&createOpts.Namespace, cli.FlagBackupStorageNamespace, "n", "", "Namespace to create the backup storage in (required)")
	createCmd.Flags().StringVar(&createOpts.Type, cli.FlagBackupStorageType, "", "Storage type (required)")
	createCmd.Flags().StringVar(&createOpts.Bucket, cli.FlagBackupStorageBucket, "", "Bucket name (required)")
	createCmd.Flags().StringVar(&createOpts.Region, cli.FlagBackupStorageRegion, "", "Storage region (required)")
	createCmd.Flags().StringVar(&createOpts.EndpointURL, cli.FlagBackupStorageEndpointURL, "", "Storage endpoint URL (required)")
	createCmd.Flags().BoolVar(&createOpts.VerifyTLS, cli.FlagBackupStorageVerifyTLS, true, "Verify TLS certificates when connecting to the endpoint")
	createCmd.Flags().BoolVar(&createOpts.ForcePathStyle, cli.FlagBackupStorageForcePathStyle, false, "Force path-style storage URLs (bucket name in the path instead of the host)")
	createCmd.Flags().StringVar(&createOpts.CredentialsSecret, cli.FlagBackupStorageCredentialsSecret, "", "Name of an existing Secret holding the storage credentials")
	createCmd.Flags().StringVar(&createOpts.AccessKeyID, cli.FlagBackupStorageAccessKeyID, "", "Access key ID (the secret access key is never passed as a flag — see --help)")
	createCmd.Flags().StringVar(&createOpts.Cluster, cli.FlagBackupStorageCluster, "main", "Cluster name")
	createCmd.Flags().StringVar(&createOpts.Context, cli.FlagBackupStorageContext, "", "Context to use (default: current context)")

	_ = createCmd.MarkFlagRequired(cli.FlagBackupStorageName)
	_ = createCmd.MarkFlagRequired(cli.FlagBackupStorageNamespace)
	_ = createCmd.MarkFlagRequired(cli.FlagBackupStorageType)
	_ = createCmd.MarkFlagRequired(cli.FlagBackupStorageBucket)
	_ = createCmd.MarkFlagRequired(cli.FlagBackupStorageRegion)
	_ = createCmd.MarkFlagRequired(cli.FlagBackupStorageEndpointURL)

	createCmd.MarkFlagsOneRequired(cli.FlagBackupStorageCredentialsSecret, cli.FlagBackupStorageAccessKeyID)
	createCmd.MarkFlagsMutuallyExclusive(cli.FlagBackupStorageCredentialsSecret, cli.FlagBackupStorageAccessKeyID)
}

func createPreRun(cmd *cobra.Command, _ []string) {
	createCfg.Pretty = !cmd.Flag(cli.FlagVerbose).Changed && !cmd.Flag(cli.FlagJSON).Changed
}

// createRun validates --type before resolving any credential, so an unsupported type
// fails. It then resolves the secret access key (env var, else interactive
// prompt) before building the request.
func createRun(cmd *cobra.Command, _ []string) {
	if err := backupstoragecli.ValidateType(createOpts.Type); err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(1)
	}

	if createOpts.AccessKeyID != "" {
		secret, err := resolveSecretAccessKey(cmd)
		if err != nil {
			output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
			os.Exit(1)
		}
		createOpts.SecretAccessKey = secret
	}

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

// resolveSecretAccessKey returns the secret access key from the environment, falling
// back to an interactive prompt when running on a real terminal.
func resolveSecretAccessKey(cmd *cobra.Command) (string, error) {
	if v := os.Getenv(envSecretAccessKey); v != "" {
		return v, nil
	}
	if cmd.Flag(cli.FlagJSON).Changed || !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("%s must be set: not running interactively", envSecretAccessKey)
	}
	return tui.NewInputPassword(cmd.Context(), "Secret access key",
		tui.WithPasswordValidation(func(s string) error {
			if s == "" {
				return errors.New("secret access key cannot be empty")
			}
			return nil
		}),
	).Run()
}

// GetCreateCmd returns the create command.
func GetCreateCmd() *cobra.Command {
	return createCmd
}
