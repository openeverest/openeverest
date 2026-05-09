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

package plugins

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	cliplugins "github.com/openeverest/openeverest/v2/pkg/cli/plugins"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	pluginEnableCmd = &cobra.Command{
		Use:     "enable [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl plugin enable --name hello -n my-namespace",
		Long:    "Enable a plugin in a namespace by creating a PluginInstallation CR",
		Short:   "Enable a plugin in a namespace",
		PreRun:  pluginEnablePreRun,
		Run:     pluginEnableRun,
	}
	pluginEnableCfg = &cliplugins.EnableConfig{}
)

func init() {
	pluginEnableCmd.Flags().StringVar(&pluginEnableCfg.Name, "name", "", "Plugin name (required)")
	pluginEnableCmd.Flags().StringVarP(&pluginEnableCfg.Namespace, "namespace", "n", "", "Target namespace (required)")
	_ = pluginEnableCmd.MarkFlagRequired("name")
	_ = pluginEnableCmd.MarkFlagRequired("namespace")
}

func pluginEnablePreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pluginEnableCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	pluginEnableCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func pluginEnableRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pe, err := cliplugins.NewPluginEnabler(*pluginEnableCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), pluginEnableCfg.Pretty)
		os.Exit(1)
	}

	if err := pe.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), pluginEnableCfg.Pretty)
		os.Exit(1)
	}
}

// GetEnableCmd returns the command to enable a plugin.
func GetEnableCmd() *cobra.Command {
	return pluginEnableCmd
}
