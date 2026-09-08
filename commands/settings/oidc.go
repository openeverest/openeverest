// everest
// Copyright (C) 2023 Percona LLC
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

// Package settings provides the Everest settings CLI commands.
package settings

import (
	"github.com/spf13/cobra"

	"github.com/percona/everest/commands/settings/oidc"
)

var settingsOIDCCmd = &cobra.Command{
	Use:   "oidc <command> [flags]",
	Args:  cobra.ExactArgs(1),
	Long:  "Manage settings related to OIDC",
	Short: "Manage settings related to OIDC",
}

func init() {
	settingsOIDCCmd.AddCommand(oidc.GetSettingsOIDCConfigureCmd())
}

// GetSettingsOIDCCmd returns the command to manage OIDC settings.
func GetSettingsOIDCCmd() *cobra.Command {
	return settingsOIDCCmd
}
