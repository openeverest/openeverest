// everest
// Copyright (C) 2025 Percona LLC
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

// Package namespaces provides the namespaces CLI command.
package namespaces

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/namespaces"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	namespacesListCmd = &cobra.Command{
		Use:     "list [flags]",
		Args:    cobra.NoArgs,
		Long:    "List namespaces managed by Everest.",
		Short:   "List namespaces managed by Everest.",
		Example: `everestctl namespaces list --all`,
		PreRun:  namespacesListPreRun,
		Run:     namespacesListRun,
	}
	namespacesListCfg = &namespaces.NamespaceListConfig{}
)

func init() {
	// local command flags
	namespacesListCmd.Flags().BoolVarP(&namespacesListCfg.ListAllNamespaces, cli.FlagNamespaceAll, "a", false, "If set, returns all namespaces in kubernetes cluster (excludes system and Everest core namespaces)")
}

func namespacesListPreRun(cmd *cobra.Command, _ []string) {
	// Copy global flags to config
	namespacesListCfg.Pretty = !cmd.Flag(cli.FlagVerbose).Changed && !cmd.Flag(cli.FlagJSON).Changed
	namespacesListCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func namespacesListRun(cmd *cobra.Command, _ []string) {
	op, err := namespaces.NewNamespaceLister(*namespacesListCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), namespacesListCfg.Pretty)
		os.Exit(1)
	}

	nsList, err := op.Run(cmd.Context())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), namespacesListCfg.Pretty)
		os.Exit(1)
	}

	op.Render(os.Stdout, nsList)
}

// GetNamespacesListCmd returns the command to list a namespaces.
func GetNamespacesListCmd() *cobra.Command {
	return namespacesListCmd
}
