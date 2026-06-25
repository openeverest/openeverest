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
	pluginInstallCmd = &cobra.Command{
		Use:  "install [flags]",
		Args: cobra.NoArgs,
		Example: `  # Install from a manifest file:
  everestctl extension install -f plugin.yaml

  # Install from a URL:
  everestctl extension install -f https://raw.githubusercontent.com/author/my-plugin/main/plugin.yaml

  # Install with inline flags:
  everestctl extension install --name hello --backend-url http://hello-plugin.everest-system:3001`,
		Long:   "Install an extension from a manifest file, URL, or inline flags",
		Short:  "Install an extension",
		PreRun: pluginInstallPreRun,
		Run:    pluginInstallRun,
	}
	pluginInstallCfg = &cliext.InstallConfig{
		Enabled: true,
	}
)

func init() {
	pluginInstallCmd.Flags().StringVarP(&pluginInstallCfg.File, "file", "f", "", "Path or URL to a Plugin CR YAML manifest")
	pluginInstallCmd.Flags().StringVar(&pluginInstallCfg.Name, "name", "", "Plugin name (required without -f)")
	pluginInstallCmd.Flags().StringVar(&pluginInstallCfg.DisplayName, "display-name", "", "Human-readable display name (defaults to name)")
	pluginInstallCmd.Flags().StringVar(&pluginInstallCfg.BackendURL, "backend-url", "", "URL of the plugin backend service (required without -f)")
	pluginInstallCmd.Flags().StringVar(&pluginInstallCfg.BundlePath, "bundle-path", "/main.js", "Path to the frontend bundle on the backend")
	pluginInstallCmd.Flags().BoolVar(&pluginInstallCfg.Enabled, "enabled", true, "Whether the plugin is enabled")
}

func pluginInstallPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pluginInstallCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	pluginInstallCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func pluginInstallRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pi, err := cliext.NewPluginInstaller(*pluginInstallCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), pluginInstallCfg.Pretty)
		os.Exit(1)
	}

	if err := pi.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), pluginInstallCfg.Pretty)
		os.Exit(1)
	}
}

// GetInstallCmd returns the command to install an extension.
func GetInstallCmd() *cobra.Command {
	return pluginInstallCmd
}
