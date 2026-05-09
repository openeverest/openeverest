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
	pluginDisableCmd = &cobra.Command{
		Use:     "disable [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl plugin disable --name hello -n my-namespace",
		Long:    "Disable a plugin in a namespace by deleting its PluginInstallation CR",
		Short:   "Disable a plugin in a namespace",
		PreRun:  pluginDisablePreRun,
		Run:     pluginDisableRun,
	}
	pluginDisableCfg = &cliplugins.DisableConfig{}
)

func init() {
	pluginDisableCmd.Flags().StringVar(&pluginDisableCfg.Name, "name", "", "Plugin name (required)")
	pluginDisableCmd.Flags().StringVarP(&pluginDisableCfg.Namespace, "namespace", "n", "", "Target namespace (required)")
	_ = pluginDisableCmd.MarkFlagRequired("name")
	_ = pluginDisableCmd.MarkFlagRequired("namespace")
}

func pluginDisablePreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pluginDisableCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	pluginDisableCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func pluginDisableRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pd, err := cliplugins.NewPluginDisabler(*pluginDisableCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), pluginDisableCfg.Pretty)
		os.Exit(1)
	}

	if err := pd.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), pluginDisableCfg.Pretty)
		os.Exit(1)
	}
}

// GetDisableCmd returns the command to disable a plugin.
func GetDisableCmd() *cobra.Command {
	return pluginDisableCmd
}
