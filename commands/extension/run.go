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
	pluginRunCmd = &cobra.Command{
		Use:   "run <plugin-name> [-- <args>...]",
		Args:  cobra.MinimumNArgs(1),
		Long:  "Run a plugin's CLI container, passing extra arguments after '--'",
		Short: "Run a plugin CLI",
		Example: `  # Run the sql-explorer plugin CLI:
  everestctl plugin run sql-explorer -- query --db my-db "SELECT 1"

  # Run with a custom container runtime:
  everestctl plugin run sql-explorer --runtime podman -- --help`,
		PreRun: pluginRunPreRun,
		Run:    pluginRunRun,
	}
	pluginRunCfg = &cliext.RunConfig{}
)

func init() {
	pluginRunCmd.Flags().StringVar(&pluginRunCfg.Runtime, "runtime", "docker", "Container runtime to use (docker, nerdctl, podman)")
}

func pluginRunPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pluginRunCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	pluginRunCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func pluginRunRun(cmd *cobra.Command, args []string) { //nolint:revive
	pluginRunCfg.PluginName = args[0]
	// Everything after the first arg (the plugin name) is passed to the container.
	if len(args) > 1 {
		pluginRunCfg.ExtraArgs = args[1:]
	}

	runner, err := cliext.NewPluginRunner(*pluginRunCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), pluginRunCfg.Pretty)
		os.Exit(1)
	}

	if err := runner.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), pluginRunCfg.Pretty)
		os.Exit(1)
	}
}

// GetRunCmd returns the run command.
func GetRunCmd() *cobra.Command {
	return pluginRunCmd
}
