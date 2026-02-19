// everest
// Copyright (C) 2025 Percona LLC
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
	"fmt"
	"os"
	"strings"

	"github.com/rodaine/table"
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

func namespacesListPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	namespacesListCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	namespacesListCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func namespacesListRun(cmd *cobra.Command, _ []string) {
	op, err := namespaces.NewNamespaceLister(*namespacesListCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), namespacesListCfg.Pretty)
		os.Exit(1)
	}

	if nsList, err := op.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), namespacesListCfg.Pretty)
		os.Exit(1)
	} else {
		if err := printNamespacesTable(nsList); err != nil {
			output.PrintError(err, logger.GetLogger(), namespacesListCfg.Pretty)
			os.Exit(1)
		}
	}
}

// GetNamespacesListCmd returns the command to list a namespaces.
func GetNamespacesListCmd() *cobra.Command {
	return namespacesListCmd
}

const (
	// columnName is the column name for the namespace.
	columnName = "namespace"
	// columnManagedByEverest is the column name for the namespace managed by Everest.
	columnManagedByEverest = "managed"
	// columnOperators is the column name for the installed Everest operators.
	columnOperators = "operators"
)

// Print namespaces to console.
func printNamespacesTable(nsList []namespaces.NamespaceInfo) error {
	// Prepare table headings.
	headings := []interface{}{columnName, columnManagedByEverest, columnOperators}
	// Prepare table header.
	tbl := table.New(headings...)
	tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
		// Print all in caps.
		return strings.ToUpper(fmt.Sprintf(format, vals...))
	})

	// Return a table row for the given account.
	row := func(ns namespaces.NamespaceInfo) []any {
		var row []any
		for _, heading := range headings {
			switch heading {
			case columnName:
				row = append(row, ns.Name)
			case columnManagedByEverest:
				row = append(row, len(ns.InstalledOperators) > 0)
			case columnOperators:
				row = append(row, strings.Join(ns.InstalledOperators, ", "))
			}
		}
		return row
	}

	for _, ns := range nsList {
		tbl.AddRow(row(ns)...)
	}

	tbl.Print()
	return nil
}
