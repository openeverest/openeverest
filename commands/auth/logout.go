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

package auth

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	logoutCmd = &cobra.Command{
		Use:     "logout",
		Args:    cobra.NoArgs,
		Example: "everestctl auth logout",
		Short:   "Log out of the current Everest server",
		Long: `Revoke the current session on the Everest server and remove its credentials
from the local config (~/.config/everest/config.yaml).`,
		PreRun: logoutPreRun,
		Run:    logoutRun,
	}
	logoutCfg = &authcli.Config{}
)

func logoutPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	logoutCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
}

func logoutRun(cmd *cobra.Command, _ []string) { //nolint:revive
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), logoutCfg.Pretty)
		os.Exit(1)
	}

	lo := authcli.NewLogin(*logoutCfg, logger.GetLogger())
	if err := lo.Logout(cmd.Context(), cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), logoutCfg.Pretty)
		os.Exit(1)
	}
}

// GetLogoutCmd returns the logout command.
func GetLogoutCmd() *cobra.Command {
	return logoutCmd
}
