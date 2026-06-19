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

	"github.com/openeverest/openeverest/v2/commands/extension"
)

var extensionCmd = &cobra.Command{
	Use:   "extension <command> [flags]",
	Args:  cobra.ExactArgs(1),
	Long:  "Manage Everest extensions (generic plugins and providers)",
	Short: "Manage Everest extensions",
	Run:   func(_ *cobra.Command, _ []string) {},
}

func init() {
	rootCmd.AddCommand(extensionCmd)

	extensionCmd.AddCommand(extension.GetListCmd())
	extensionCmd.AddCommand(extension.GetInstallCmd())
	extensionCmd.AddCommand(extension.GetUninstallCmd())
	extensionCmd.AddCommand(extension.GetNamespaceCmd())
	extensionCmd.AddCommand(extension.GetRunCmd())
}
