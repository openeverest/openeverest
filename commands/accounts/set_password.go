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
	accountsSetPasswordCmd = &cobra.Command{
		Use:     "set-password [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl accounts set-password --username user1 --new-password $USER_PASS",
		Long:    "Set a new password for an existing Everest user account",
		Short:   "Set a new password for an existing Everest user account",
		PreRun:  accountsSetPasswordPreRun,
		Run:     accountsSetPasswordRun,
	}
	accountsSetPasswordCfg  = &accountscli.Config{}
	accountsSetPasswordOpts = &accountscli.SetPasswordOptions{}
)

func init() {
	// local command flags
	accountsSetPasswordCmd.Flags().StringVarP(&accountsSetPasswordOpts.Username, cli.FlagAccountsUsername, "u", "", "Username of the account")
	accountsSetPasswordCmd.Flags().StringVarP(&accountsSetPasswordOpts.NewPassword, cli.FlagAccountsNewPassword, "p", "", "New password for the account")
}

func accountsSetPasswordPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	accountsSetPasswordCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	accountsSetPasswordCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()

	// Check username
	if accountsSetPasswordOpts.Username != "" {
		// Validate provided username for whom password shall be changed.
		if err := accountscli.ValidateUsername(accountsSetPasswordOpts.Username); err != nil {
			output.PrintError(err, logger.GetLogger(), accountsSetPasswordCfg.Pretty)
			os.Exit(1)
		}
	} else {
		// Ask user in interactive mode to provide username for whom password shall be changed.
		if username, err := accountscli.PopulateUsername(cmd.Context()); err != nil {
			output.PrintError(err, logger.GetLogger(), accountsSetPasswordCfg.Pretty)
			os.Exit(1)
		} else {
			accountsSetPasswordOpts.Username = username
		}
	}

	// Check password
	if accountsSetPasswordOpts.NewPassword != "" {
		// Validate provided a new password.
		if err := accountscli.ValidatePassword(accountsSetPasswordOpts.NewPassword); err != nil {
			output.PrintError(err, logger.GetLogger(), accountsSetPasswordCfg.Pretty)
			os.Exit(1)
		}
	} else {
		// Ask user in interactive mode to provide a new password.
		if password, err := accountscli.PopulateNewPassword(cmd.Context()); err != nil {
			output.PrintError(err, logger.GetLogger(), accountsSetPasswordCfg.Pretty)
			os.Exit(1)
		} else {
			accountsSetPasswordOpts.NewPassword = password
		}
	}
}

func accountsSetPasswordRun(cmd *cobra.Command, _ []string) { //nolint:revive
	cliA, err := accountscli.NewAccounts(*accountsSetPasswordCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), accountsSetPasswordCfg.Pretty)
		os.Exit(1)
	}

	if err := cliA.SetPassword(cmd.Context(), *accountsSetPasswordOpts); err != nil {
		output.PrintError(err, logger.GetLogger(), accountsSetPasswordCfg.Pretty)
		os.Exit(1)
	}
}

// GetSetPasswordCmd returns the command to set password for an account.
func GetSetPasswordCmd() *cobra.Command {
	return accountsSetPasswordCmd
}
