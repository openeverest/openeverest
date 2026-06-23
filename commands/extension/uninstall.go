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

package extension

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	cliext "github.com/openeverest/openeverest/v2/pkg/cli/extension"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	pluginUninstallCmd = &cobra.Command{
		Use:     "uninstall [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl plugin uninstall --name hello",
		Long:    "Uninstall a plugin by deleting its Plugin custom resource",
		Short:   "Uninstall a plugin",
		PreRun:  pluginUninstallPreRun,
		Run:     pluginUninstallRun,
	}
	pluginUninstallCfg = &cliext.UninstallConfig{}
)

func init() {
	pluginUninstallCmd.Flags().StringVar(&pluginUninstallCfg.Name, "name", "", "Plugin name (required)")
	_ = pluginUninstallCmd.MarkFlagRequired("name")
}

func pluginUninstallPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pluginUninstallCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	pluginUninstallCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func pluginUninstallRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pu, err := cliext.NewPluginUninstaller(*pluginUninstallCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), pluginUninstallCfg.Pretty)
		os.Exit(1)
	}

	if err := pu.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), pluginUninstallCfg.Pretty)
		os.Exit(1)
	}
}

// GetUninstallCmd returns the command to uninstall a plugin.
func GetUninstallCmd() *cobra.Command {
	return pluginUninstallCmd
}
