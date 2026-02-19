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

// Package rbac provides RBAC settings CLI commands.
package rbac

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

var (
	settingsRBACValidateCmd = &cobra.Command{
		Use:     "validate [flags]",
		Long:    "Validate RBAC settings",
		Short:   "Validate RBAC settings",
		Example: "everestctl settings rbac validate --policy-file <file_path>",
		PreRun:  settingsRBACValidatePreRun,
		Run:     settingsRBACValidateRun,
	}
	rbacValidatePolicyFilePath string
	rbacValidateKubeconfigPath string
	rbacValidatePretty         bool
)

func init() {
	// local command flags
	settingsRBACValidateCmd.Flags().StringVar(&rbacValidatePolicyFilePath, cli.FlagRBACPolicyFile, "", "Path to the policy file to use, otherwise use policy from Everest deployment.")
}

func settingsRBACValidatePreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	rbacValidatePretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	rbacValidateKubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
}

func settingsRBACValidateRun(cmd *cobra.Command, _ []string) {
	var k kubernetes.KubernetesConnector
	if rbacValidatePolicyFilePath == "" {
		// check over policy in Everest deployment (ConfigMap).
		var l *zap.SugaredLogger
		if rbacValidatePretty {
			l = zap.NewNop().Sugar()
		} else {
			l = logger.GetLogger().With("component", "rbac")
		}

		client, err := kubernetes.New(rbacValidateKubeconfigPath, l)
		if err != nil {
			logger.GetLogger().Error(err)
			os.Exit(1)
		}
		k = client
	}

	err := rbac.ValidatePolicy(cmd.Context(), k, rbacValidatePolicyFilePath)
	if err != nil {
		_, _ = fmt.Fprint(os.Stdout, output.Failure("Invalid"))
		msg := err.Error()
		msg = strings.Join(strings.Split(msg, "\n"), " - ")
		_, _ = fmt.Fprintln(os.Stdout, msg)
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stdout, output.Success("Valid"))
}

// GetSettingsRBACValidateCmd returns the command to validate RBAC policies.
func GetSettingsRBACValidateCmd() *cobra.Command {
	return settingsRBACValidateCmd
}
