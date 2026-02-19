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
	"os"

	"github.com/spf13/cobra"

	accountscli "github.com/openeverest/openeverest/v2/pkg/accounts/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	accountsResetJWTKeysCmd = &cobra.Command{
		Use:     "reset-jwt-keys [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl accounts reset-jwt-keys",
		Long:    "Reset the JWT keys used for Everest user authentication",
		Short:   "Reset the JWT keys used for Everest user authentication",
		PreRun:  accountsResetJWTKeysPreRun,
		Run:     accountsResetJWTKeysRun,
	}
	accountsResetJWTKeysCfg = &accountscli.Config{}
)

func accountsResetJWTKeysPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	accountsResetJWTKeysCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	accountsResetJWTKeysCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func accountsResetJWTKeysRun(cmd *cobra.Command, _ []string) { //nolint:revive
	cliA, err := accountscli.NewAccounts(*accountsResetJWTKeysCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), accountsResetJWTKeysCfg.Pretty)
		os.Exit(1)
	}

	if err := cliA.CreateRSAKeyPair(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), accountsResetJWTKeysCfg.Pretty)
		os.Exit(1)
	}
}

// GetResetJWTKeysCmd returns the command to reset the JWT keys used for Everest user authentication.
func GetResetJWTKeysCmd() *cobra.Command {
	return accountsResetJWTKeysCmd
}
