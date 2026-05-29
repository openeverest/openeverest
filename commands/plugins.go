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

// Package commands ...
package commands

import (
	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/commands/plugins"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugin <command> [flags]",
	Args:  cobra.ExactArgs(1),
	Long:  "Manage Everest plugins",
	Short: "Manage Everest plugins",
	Run:   func(_ *cobra.Command, _ []string) {},
}

func init() {
	rootCmd.AddCommand(pluginsCmd)

	pluginsCmd.AddCommand(plugins.GetListCmd())
	pluginsCmd.AddCommand(plugins.GetInstallCmd())
	pluginsCmd.AddCommand(plugins.GetUninstallCmd())
	pluginsCmd.AddCommand(plugins.GetEnableCmd())
	pluginsCmd.AddCommand(plugins.GetDisableCmd())
	pluginsCmd.AddCommand(plugins.GetRunCmd())
}
