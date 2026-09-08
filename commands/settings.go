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

// Package commands ...
package commands

import (
	"github.com/spf13/cobra"

	"github.com/percona/everest/commands/settings"
)

var settingsCmd = &cobra.Command{
	Use:   "settings <command> [flags]",
	Args:  cobra.ExactArgs(1),
	Long:  "Configure Everest settings",
	Short: "Configure Everest settings",
	Run:   func(_ *cobra.Command, _ []string) {},
}

func init() {
	rootCmd.AddCommand(settingsCmd)

	settingsCmd.AddCommand(settings.GetSettingsOIDCCmd())
	settingsCmd.AddCommand(settings.GetSettingsRBACCmd())
}
