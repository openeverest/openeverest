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

package commands

import (
	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/commands/backupstorage"
)

var backupStorageCmd = &cobra.Command{
	Use:     "backup-storage <command> [flags]",
	Aliases: []string{"backupstorage", "bs"},
	Short:   "Manage Everest backup storages",
	Long:    "Manage Everest backup storages",
	RunE:    func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
}

func init() {
	rootCmd.AddCommand(backupStorageCmd)
	backupStorageCmd.AddCommand(backupstorage.GetListCmd())
	backupStorageCmd.AddCommand(backupstorage.GetCreateCmd())
}
