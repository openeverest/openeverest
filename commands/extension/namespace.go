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
	namespaceCmd = &cobra.Command{
		Use:   "namespace <command> [flags]",
		Args:  cobra.ExactArgs(1),
		Long:  "Manage which namespaces an extension is enabled in",
		Short: "Manage extension namespaces",
		Run:   func(_ *cobra.Command, _ []string) {},
	}

	namespaceAddCmd = &cobra.Command{
		Use:     "add [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl extension namespace add --name hello -n my-namespace",
		Long:    "Enable an extension in a namespace by appending it to spec.plugin.namespaces[]",
		Short:   "Enable an extension in a namespace",
		PreRun:  namespaceAddPreRun,
		Run:     namespaceAddRun,
	}
	namespaceAddCfg = &cliext.EnableConfig{}

	namespaceRemoveCmd = &cobra.Command{
		Use:     "remove [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl extension namespace remove --name hello -n my-namespace",
		Long:    "Disable an extension in a namespace by removing it from spec.plugin.namespaces[]",
		Short:   "Disable an extension in a namespace",
		PreRun:  namespaceRemovePreRun,
		Run:     namespaceRemoveRun,
	}
	namespaceRemoveCfg = &cliext.DisableConfig{}
)

func init() {
	namespaceAddCmd.Flags().StringVar(&namespaceAddCfg.Name, "name", "", "Extension name (required)")
	namespaceAddCmd.Flags().StringVarP(&namespaceAddCfg.Namespace, "namespace", "n", "", "Target namespace (required)")
	_ = namespaceAddCmd.MarkFlagRequired("name")
	_ = namespaceAddCmd.MarkFlagRequired("namespace")

	namespaceRemoveCmd.Flags().StringVar(&namespaceRemoveCfg.Name, "name", "", "Extension name (required)")
	namespaceRemoveCmd.Flags().StringVarP(&namespaceRemoveCfg.Namespace, "namespace", "n", "", "Target namespace (required)")
	_ = namespaceRemoveCmd.MarkFlagRequired("name")
	_ = namespaceRemoveCmd.MarkFlagRequired("namespace")

	namespaceCmd.AddCommand(namespaceAddCmd)
	namespaceCmd.AddCommand(namespaceRemoveCmd)
}

func namespaceAddPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	namespaceAddCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	namespaceAddCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func namespaceAddRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pe, err := cliext.NewPluginEnabler(*namespaceAddCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), namespaceAddCfg.Pretty)
		os.Exit(1)
	}
	if err := pe.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), namespaceAddCfg.Pretty)
		os.Exit(1)
	}
}

func namespaceRemovePreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	namespaceRemoveCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	namespaceRemoveCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func namespaceRemoveRun(cmd *cobra.Command, _ []string) { //nolint:revive
	pd, err := cliext.NewPluginDisabler(*namespaceRemoveCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), namespaceRemoveCfg.Pretty)
		os.Exit(1)
	}
	if err := pd.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), namespaceRemoveCfg.Pretty)
		os.Exit(1)
	}
}

// GetNamespaceCmd returns the "namespace" command tree for managing extension namespaces.
func GetNamespaceCmd() *cobra.Command {
	return namespaceCmd
}
