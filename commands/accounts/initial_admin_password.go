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
	accountsInitAdminPasswdCmd = &cobra.Command{
		Use:     "initial-admin-password [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl accounts initial-admin-password",
		Long:    "Get the initial admin password for Everest",
		Short:   "Get the initial admin password for Everest",
		PreRun:  accountsInitAdminPasswdPreRun,
		Run:     accountsInitAdminPasswdRun,
	}
	accountsInitAdminPasswdCfg = &accountscli.Config{}
)

func accountsInitAdminPasswdPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	accountsInitAdminPasswdCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	accountsInitAdminPasswdCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func accountsInitAdminPasswdRun(cmd *cobra.Command, _ []string) { //nolint:revive
	cliA, err := accountscli.NewAccounts(*accountsInitAdminPasswdCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), accountsInitAdminPasswdCfg.Pretty)
		os.Exit(1)
	}

	passwordHash, err := cliA.GetInitAdminPassword(cmd.Context())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), accountsInitAdminPasswdCfg.Pretty)
		os.Exit(1)
	}

	_, _ = fmt.Fprint(os.Stdout, passwordHash+"\n")
}

// GetInitAdminPasswordCmd returns the command to get the initial admin password.
func GetInitAdminPasswordCmd() *cobra.Command {
	return accountsInitAdminPasswdCmd
}
