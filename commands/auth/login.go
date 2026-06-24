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

// Package auth holds commands for the auth subcommand.
package auth

import (
	"os"

	"github.com/spf13/cobra"

	accountscli "github.com/openeverest/openeverest/v2/pkg/accounts/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/cli/tui"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	loginCmd = &cobra.Command{
		Use:     "login [flags]",
		Args:    cobra.NoArgs,
		Example: "everestctl auth login --server http://localhost:8080 --username admin",
		Short:   "Log in to an Everest server",
		Long: `Authenticate with an Everest server and store credentials locally.

Exchanges your username and password for an access token and refresh token,
persisted to ~/.config/everest/config.yaml. The context is named
username@host:port by default; use --context-name to override.`,
		PreRun:  loginPreRun,
		Run:     loginRun,
	}
	loginCfg  = &authcli.Config{}
	loginOpts = &authcli.LoginOptions{}
)

func init() {
	loginCmd.Flags().StringVar(&loginOpts.Server, cli.FlagAuthServer, "", "Everest server URL (e.g. http://localhost:8080)")
	loginCmd.Flags().StringVarP(&loginOpts.Username, cli.FlagAccountsUsername, "u", "", "Username")
	loginCmd.Flags().StringVarP(&loginOpts.Password, cli.FlagAccountsCreatePassword, "p", "", "Password")
	loginCmd.Flags().StringVar(&loginOpts.ContextName, cli.FlagAuthContextName, "", "Context name (default: server hostname)")
}

func loginPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	loginCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)

	if loginOpts.Server == "" {
		server, err := tui.NewInput(cmd.Context(), "Everest server URL").Run()
		if err != nil {
			output.PrintError(err, logger.GetLogger(), loginCfg.Pretty)
			os.Exit(1)
		}
		loginOpts.Server = server
	}

	if loginOpts.Username == "" {
		username, err := accountscli.PopulateUsername(cmd.Context())
		if err != nil {
			output.PrintError(err, logger.GetLogger(), loginCfg.Pretty)
			os.Exit(1)
		}
		loginOpts.Username = username
	}

	if loginOpts.Password == "" {
		password, err := tui.NewInputPassword(cmd.Context(), "Provide password").Run()
		if err != nil {
			output.PrintError(err, logger.GetLogger(), loginCfg.Pretty)
			os.Exit(1)
		}
		loginOpts.Password = password
	}
}

func loginRun(cmd *cobra.Command, _ []string) { //nolint:revive
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), loginCfg.Pretty)
		os.Exit(1)
	}

	lo := authcli.NewLogin(*loginCfg, logger.GetLogger())
	if err := lo.Run(cmd.Context(), *loginOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), loginCfg.Pretty)
		os.Exit(1)
	}
}

// GetLoginCmd returns the login command.
func GetLoginCmd() *cobra.Command {
	return loginCmd
}
