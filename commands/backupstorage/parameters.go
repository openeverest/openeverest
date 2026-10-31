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
	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	backupstoragecli "github.com/openeverest/openeverest/v2/pkg/cli/backupstorage"
)

// addParameterFlags registers --parameter and the s3 shorthands, which
// readParameterShorthands reads back.
func addParameterFlags(cmd *cobra.Command, opts *backupstoragecli.ParameterOptions) {
	cmd.Flags().StringArrayVar(&opts.Parameters, cli.FlagBackupStorageParameter, nil,
		"Set a connection parameter as key=value (repeatable); true, false and numbers keep their JSON type",
	)
	cmd.Flags().String(cli.FlagBackupStorageRegion, "", "s3: shorthand for --parameter region=...")
	cmd.Flags().String(cli.FlagBackupStorageEndpointURL, "", "s3: shorthand for --parameter endpointURL=...")
	cmd.Flags().Bool(cli.FlagBackupStorageForcePathStyle, false, "s3: shorthand for --parameter forcePathStyle=true (bucket in the URL path, not the host name)")
	cmd.Flags().String(cli.FlagBackupStorageImplementation, "", "s3: shorthand for --parameter implementation=... (aws, minio, ceph, gcs or other)")
}

// readParameterShorthands sets the shorthands that were passed, leaving the
// others nil so they name no parameter.
func readParameterShorthands(cmd *cobra.Command, opts *backupstoragecli.ParameterOptions) {
	flags := cmd.Flags()
	if flags.Changed(cli.FlagBackupStorageRegion) {
		v, _ := flags.GetString(cli.FlagBackupStorageRegion)
		opts.Region = &v
	}
	if flags.Changed(cli.FlagBackupStorageEndpointURL) {
		v, _ := flags.GetString(cli.FlagBackupStorageEndpointURL)
		opts.EndpointURL = &v
	}
	if flags.Changed(cli.FlagBackupStorageForcePathStyle) {
		v, _ := flags.GetBool(cli.FlagBackupStorageForcePathStyle)
		opts.ForcePathStyle = &v
	}
	if flags.Changed(cli.FlagBackupStorageImplementation) {
		v, _ := flags.GetString(cli.FlagBackupStorageImplementation)
		opts.Implementation = &v
	}
}
