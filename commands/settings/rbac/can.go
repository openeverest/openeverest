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
	"errors"
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

const canCmdExamples = `
Examples:
# Check if user 'alice' can get 'cluster-1' in namespace 'my-namespace'
$ everestctl settings rbac can alice read database-clusters my-namespace/cluster-1

# Check if user 'alice' can perform all/any actions on 'cluster-1' in namespace 'my-namespace'
$ everestctl settings rbac can alice '*' database-clusters my-namespace/cluster-1

# Check if role 'role:admin' can update backup 'prod-backup-1' in namespace 'prod-namespace'
$ everestctl settings rbac can role:admin update database-cluster-backups prod-namespace/prod-backup-1

# Check if user 'bob' can delete all/any backups in namespace 'prod-namespace'
$ everestctl settings rbac can bob delete database-cluster-backups prod-namespace/*

# Check if user 'alice' can perform all/any actions on all backups in all namespaces
$ everestctl settings rbac can alice '*' database-cluster-backups '*'

NOTE: The asterisk character (*) holds a special meaning in the unix shell.
To prevent misinterpretation, you need to add single quotes around it.
`

var (
	settingsRBACCanCmd = &cobra.Command{
		Use:     "can <subject> <action> <resource> <subresource> [flags]",
		Args:    cobra.ExactArgs(4),
		Long:    `Test RBAC policy.` + "\n" + canCmdExamples,
		Short:   "Test RBAC policy",
		Example: "everestctl settings rbac can alice read database-clusters all",
		PreRunE: settingsRBACCanPreRunE,
		Run:     settingsRBACCanRun,
	}
	rbacCanPolicyFilePath string
	rbacCanKubeconfigPath string
	rbacCanPretty         bool
)

func init() {
	// local command flags
	settingsRBACCanCmd.Flags().StringVar(&rbacCanPolicyFilePath, cli.FlagRBACPolicyFile, "", "Path to the policy file to use, otherwise use policy from Everest deployment.")
}

func settingsRBACCanPreRunE(cmd *cobra.Command, args []string) error { //nolint:revive
	// validate action
	if !rbac.ValidateAction(args[1]) {
		return errors.New(fmt.Sprintf("invalid action '%s'. Supported actions: %s",
			args[1], strings.Join(rbac.SupportedActions, `,`),
		))
	}

	// Copy global flags to config
	rbacCanPretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
	rbacCanKubeconfigPath = cmd.Flag(cli.FlagKubeconfig).Value.String()
	return nil
}

func settingsRBACCanRun(cmd *cobra.Command, args []string) {
	var k kubernetes.KubernetesConnector
	if rbacCanPolicyFilePath == "" {
		// check over policy in Everest deployment (ConfigMap).
		var l *zap.SugaredLogger
		if rbacCanPretty {
			l = zap.NewNop().Sugar()
		} else {
			l = logger.GetLogger().With("component", "rbac")
		}

		client, err := kubernetes.New(rbacCanKubeconfigPath, l)
		if err != nil {
			output.PrintError(err, logger.GetLogger(), rbacCanPretty)
			os.Exit(1)
		}
		k = client
	}

	can, err := rbac.Can(cmd.Context(), rbacCanPolicyFilePath, k, args...)
	if err != nil {
		output.PrintError(err, logger.GetLogger(), rbacCanPretty)
		os.Exit(1)
	}

	if can {
		_, _ = fmt.Fprintln(os.Stdout, "Yes")
		return
	}
	_, _ = fmt.Fprintln(os.Stdout, "No")
}

// GetSettingsRBACCanCmd returns the command to test RBAC policy.
func GetSettingsRBACCanCmd() *cobra.Command {
	return settingsRBACCanCmd
}
