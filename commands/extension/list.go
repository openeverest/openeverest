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

// Package plugins holds commands for the plugin command.
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
	pluginListCmd = &cobra.Command{
		Use:     "list [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl plugin list",
		Long:    "List all installed plugins",
		Short:   "List all installed plugins",
		PreRun:  pluginListPreRun,
		Run:     pluginListRun,
	}
	pluginListCfg = &cliext.ListConfig{}
)

func pluginListPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pluginListCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	pluginListCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func pluginListRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pl, err := cliext.NewPluginLister(*pluginListCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), pluginListCfg.Pretty)
		os.Exit(1)
	}

	if err := pl.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), pluginListCfg.Pretty)
		os.Exit(1)
	}
}

// GetListCmd returns the command to list plugins.
func GetListCmd() *cobra.Command {
	return pluginListCmd
}
