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

// Package provider holds commands for provider management.
package provider

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	providercli "github.com/openeverest/openeverest/v2/pkg/cli/provider"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	listCmd = &cobra.Command{
		Use:     "list [flags]",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Short:   "List providers",
		Long: `List Provider resources via the Everest API.

Displays a table with NAME and AGE. Details such as supported topologies,
version catalog, and schemas remain available via --json / --verbose.`,
		Example: `  # List installed providers
  everestctl provider list

  # Using aliases
  everestctl prov ls

  # JSON output — inspect the full version catalog
  everestctl provider list --json | jq '.[].spec.versions'

  # Specific cluster and context
  everestctl provider list --cluster staging --context staging-ctx`,
		PreRun: listPreRun,
		Run:    listRun,
	}
	listCfg  = &providercli.Config{}
	listOpts = &providercli.ListOptions{}
)

func init() {
	listCmd.Flags().StringVar(&listOpts.Cluster, cli.FlagProviderCluster, "main", "Cluster name")
	listCmd.Flags().StringVar(&listOpts.Context, cli.FlagProviderContext, "", "Context to use (default: current context)")
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

	runner := providercli.NewListRunner(*listCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *listOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), listCfg.Pretty)
		os.Exit(1)
	}
}

// GetListCmd returns the list command.
func GetListCmd() *cobra.Command {
	return listCmd
}
