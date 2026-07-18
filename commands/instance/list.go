// everest
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

package instance

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	instancecli "github.com/openeverest/openeverest/v2/pkg/cli/instance"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	listCmd = &cobra.Command{
		Use:     "list [flags]",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List instances",
		Long: `List Instance resources via the Everest API.

Displays a table with NAME, NAMESPACE, PROVIDER, VERSION, PHASE, and AGE.
Pass --json / --verbose to get the raw API response instead of the formatted
table.`,
		Example: `  # List instances in a namespace
  everestctl instance list --namespace everest

  # List across all namespaces
  everestctl instance ls --all-namespaces

  # JSON output for scripting
  everestctl instance list --namespace everest --json | jq '.[].metadata.name'

  # Specific cluster and context
  everestctl instance list --namespace everest --cluster staging --context staging-ctx`,
		PreRun: listPreRun,
		Run:    listRun,
	}
	listCfg  = &instancecli.Config{}
	listOpts = &instancecli.ListOptions{}
)

func init() {
	listCmd.Flags().StringVarP(&listOpts.Namespace, cli.FlagInstanceNamespace, "n", "", "Namespace to list instances in")
	listCmd.Flags().BoolVarP(&listOpts.AllNamespaces, cli.FlagInstanceAllNamespaces, "A", false, "List instances across all namespaces")
	listCmd.Flags().StringVar(&listOpts.Cluster, cli.FlagInstanceCluster, "main", "Cluster name")
	listCmd.Flags().StringVar(&listOpts.Context, cli.FlagInstanceContext, "", "Context to use (default: current context)")

	listCmd.MarkFlagsOneRequired(cli.FlagInstanceNamespace, cli.FlagInstanceAllNamespaces)
	listCmd.MarkFlagsMutuallyExclusive(cli.FlagInstanceNamespace, cli.FlagInstanceAllNamespaces)
}

func listPreRun(cmd *cobra.Command, _ []string) {
	listCfg.Pretty = !cmd.Flag(cli.FlagVerbose).Changed && !cmd.Flag(cli.FlagJSON).Changed
}

func listRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}

	runner := instancecli.NewListRunner(*listCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *listOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}
}

// GetListCmd returns the list command.
func GetListCmd() *cobra.Command {
	return listCmd
}
