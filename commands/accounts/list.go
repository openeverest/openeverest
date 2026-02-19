// everest
// Copyright (C) 2023 Percona LLC
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

// Package accounts holds commands for accounts command.
package accounts

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	accountscli "github.com/openeverest/openeverest/v2/pkg/accounts/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	accountsListCmd = &cobra.Command{
		Use:     "list [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl accounts list --no-headers",
		Long:    "List all Everest user accounts",
		Short:   "List all Everest user accounts",
		PreRun:  accountsListPreRun,
		Run:     accountsListRun,
	}
	accountsListCfg  = &accountscli.Config{}
	accountsListOpts = &accountscli.ListOptions{}
)

func init() {
	// local command flags
	accountsListCmd.Flags().BoolVar(&accountsListOpts.NoHeaders, "no-headers", false, "If set, hide table headers")
	accountsListCmd.Flags().StringSliceVar(&accountsListOpts.Columns, "columns", nil,
		fmt.Sprintf("Comma-separated list of column names to display. Supported columns: %s, %s, %s.",
			accountscli.ColumnUser, accountscli.ColumnCapabilities, accountscli.ColumnEnabled,
		),
	)
}

func accountsListPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	accountsListCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	accountsListCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func accountsListRun(cmd *cobra.Command, _ []string) { //nolint:revive
	cliA, err := accountscli.NewAccounts(*accountsListCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), accountsListCfg.Pretty)
		os.Exit(1)
	}

	if err := cliA.List(cmd.Context(), *accountsListOpts); err != nil {
		output.PrintError(err, logger.GetLogger(), accountsListCfg.Pretty)
		os.Exit(1)
	}
}

// GetListCmd returns the command to list all user accounts.
func GetListCmd() *cobra.Command {
	return accountsListCmd
}
