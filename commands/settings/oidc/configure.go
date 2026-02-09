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

// Package oidc provides OIDC settings CLI commands.
package oidc

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/oidc"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	settingsOIDCConfigureCmd = &cobra.Command{
		Use:     "configure [flags]",
		Args:    cobra.NoArgs,
		Long:    "Configure OIDC settings",
		Short:   "Configure OIDC settings",
		Example: `everestctl settings oidc configure --issuer-url https://example.com --client-id 123456 --scopes openid,profile,email,groups`,
		PreRun:  settingsOIDCConfigurePreRun,
		Run:     settingsOIDCConfigureRun,
	}
	settingsOIDCConfigureCfg = &oidc.Config{}
	scopes                   string
)

func init() {
	// local command flags
	settingsOIDCConfigureCmd.Flags().StringVar(&settingsOIDCConfigureCfg.IssuerURL, cli.FlagOIDCIssuerURL, "", "OIDC issuer url")
	settingsOIDCConfigureCmd.Flags().StringVar(&settingsOIDCConfigureCfg.ClientID, cli.FlagOIDCClientID, "", "OIDC application client ID")
	settingsOIDCConfigureCmd.Flags().StringVar(&scopes, cli.FlagOIDCScopes, strings.Join(common.DefaultOIDCScopes, ","), "Comma-separated list of scopes")
}

func settingsOIDCConfigurePreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	settingsOIDCConfigureCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	settingsOIDCConfigureCfg.KubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()

	// Check if issuer URL is provided
	if settingsOIDCConfigureCfg.IssuerURL == "" {
		// Ask user to provide issuer URL in interactive mode
		if err := settingsOIDCConfigureCfg.PopulateIssuerURL(cmd.Context()); err != nil {
			output.PrintError(err, logger.GetLogger(), settingsOIDCConfigureCfg.Pretty)
			os.Exit(1)
		}
	} else {
		// Validate issuer URL provided by user in flags
		if err := oidc.ValidateURL(settingsOIDCConfigureCfg.IssuerURL); err != nil {
			output.PrintError(err, logger.GetLogger(), settingsOIDCConfigureCfg.Pretty)
			os.Exit(1)
		}
	}

	// Check if Client ID is provided
	if settingsOIDCConfigureCfg.ClientID == "" {
		// Ask user to provide client ID in interactive mode
		if err := settingsOIDCConfigureCfg.PopulateClientID(cmd.Context()); err != nil {
			output.PrintError(err, logger.GetLogger(), settingsOIDCConfigureCfg.Pretty)
			os.Exit(1)
		}
	} else {
		// Validate client ID provided by user in flags
		if err := oidc.ValidateClientID(settingsOIDCConfigureCfg.ClientID); err != nil {
			output.PrintError(err, logger.GetLogger(), settingsOIDCConfigureCfg.Pretty)
			os.Exit(1)
		}
	}

	// Validate scopes (default or provided by user in flags)
	scopesList := strings.Split(scopes, ",")
	if err := oidc.ValidateScopes(scopesList); err != nil {
		output.PrintError(err, logger.GetLogger(), settingsOIDCConfigureCfg.Pretty)
		os.Exit(1)
	}
	settingsOIDCConfigureCfg.Scopes = scopesList
}

func settingsOIDCConfigureRun(cmd *cobra.Command, _ []string) {
	op, err := oidc.NewOIDC(*settingsOIDCConfigureCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), settingsOIDCConfigureCfg.Pretty)
		os.Exit(1)
	}

	if err := op.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), settingsOIDCConfigureCfg.Pretty)
		os.Exit(1)
	}
}

// GetSettingsOIDCConfigureCmd returns the command to configure OIDC settings.
func GetSettingsOIDCConfigureCmd() *cobra.Command {
	return settingsOIDCConfigureCmd
}
