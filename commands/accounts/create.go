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
//
//nolint:dupl
package accounts

import (
	"os"

	"github.com/spf13/cobra"

	accountscli "github.com/openeverest/openeverest/v2/pkg/accounts/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	accountsCreateCmd = &cobra.Command{
		Use:     "create [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl accounts create --username user1 --password $USER_PASS",
		Short:   "Create a new Everest user account",
		Long:    "Create a new Everest user account",
		PreRun:  accountsCreatePreRun,
		Run:     accountsCreateRun,
	}

	accountsCreateCfg  = &accountscli.Config{}
	accountsCreateOpts = &accountscli.CreateOptions{}
)

func init() {
	// local command flags
	accountsCreateCmd.Flags().StringVarP(&accountsCreateOpts.Username, cli.FlagAccountsUsername, "u", "", "Username of the account")
	accountsCreateCmd.Flags().StringVarP(&accountsCreateOpts.Password, cli.FlagAccountsCreatePassword, "p", "", "Password of the account")
}

func accountsCreatePreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	accountsCreateCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	accountsCreateCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()

	// Check username
	if accountsCreateOpts.Username != "" {
		// Validate provided username for new account.
		if err := accountscli.ValidateUsername(accountsCreateOpts.Username); err != nil {
			output.PrintError(err, logger.GetLogger(), accountsCreateCfg.Pretty)
			os.Exit(1)
		}
	} else {
		// Ask user in interactive mode to provide username for new account.
		if username, err := accountscli.PopulateUsername(cmd.Context()); err != nil {
			output.PrintError(err, logger.GetLogger(), accountsCreateCfg.Pretty)
			os.Exit(1)
		} else {
			accountsCreateOpts.Username = username
		}
	}

	// Check password
	if accountsCreateOpts.Password != "" {
		// Validate provided password for new account.
		if err := accountscli.ValidatePassword(accountsCreateOpts.Password); err != nil {
			output.PrintError(err, logger.GetLogger(), accountsCreateCfg.Pretty)
			os.Exit(1)
		}
	} else {
		// Ask user in interactive mode to provide password for new account.
		if password, err := accountscli.PopulatePassword(cmd.Context()); err != nil {
			output.PrintError(err, logger.GetLogger(), accountsCreateCfg.Pretty)
			os.Exit(1)
		} else {
			accountsCreateOpts.Password = password
		}
	}
}

func accountsCreateRun(cmd *cobra.Command, _ []string) { //nolint:revive
	cliA, err := accountscli.NewAccounts(*accountsCreateCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), accountsCreateCfg.Pretty)
		os.Exit(1)
	}

	if err := cliA.Create(cmd.Context(), *accountsCreateOpts); err != nil {
		output.PrintError(err, logger.GetLogger(), accountsCreateCfg.Pretty)
		os.Exit(1)
	}
}

// GetCreateCmd returns the command to create a new user account.
func GetCreateCmd() *cobra.Command {
	return accountsCreateCmd
}
