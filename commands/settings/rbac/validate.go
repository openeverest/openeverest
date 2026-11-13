// everest
// Copyright (C) 2023 Percona LLC
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

// Package rbac provides RBAC settings CLI commands.
package rbac

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/helm"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

var (
	settingsRBACValidateCmd = &cobra.Command{
		Use:     "validate [flags]",
		Args:    cobra.NoArgs,
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

// ValidateResult represents the result of an RBAC policy validation check.
type ValidateResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

func renderValidate(w io.Writer, err error, pretty bool) {
	if !pretty {
		res := ValidateResult{Valid: err == nil}
		if err != nil {
			res.Error = err.Error()
		}
		_ = json.NewEncoder(w).Encode(res) //nolint:errchkjson
		return
	}
	if err != nil {
		_, _ = fmt.Fprint(w, output.Failure("Invalid"))
		msg := err.Error()
		msg = strings.Join(strings.Split(msg, "\n"), " - ")
		_, _ = fmt.Fprintln(w, msg)
		return
	}
	_, _ = fmt.Fprint(w, output.Success("Valid"))
}

func settingsRBACValidatePreRun(cmd *cobra.Command, _ []string) {
	// Copy global flags to config
	rbacValidatePretty = !isFlagChanged(cmd, cli.FlagVerbose) && !isFlagChanged(cmd, cli.FlagJSON)
	if f := cmd.Flag(cli.FlagKubeconfig); f != nil {
		rbacValidateKubeconfigPath = f.Value.String()
	}
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

		ns, err := helm.DiscoverOpenEverestNamespace(rbacValidateKubeconfigPath)
		if err != nil {
			output.PrintError(err, logger.GetLogger(), rbacValidatePretty)
			os.Exit(1)
		}
		client, err := kubernetes.New(rbacValidateKubeconfigPath, l, ns)
		if err != nil {
			output.PrintError(err, logger.GetLogger(), rbacValidatePretty)
			os.Exit(1)
		}
		k = client
	}

	err := rbac.ValidatePolicy(cmd.Context(), k, rbacValidatePolicyFilePath)
	renderValidate(os.Stdout, err, rbacValidatePretty)
	if err != nil {
		os.Exit(1)
	}
}

// GetSettingsRBACValidateCmd returns the command to validate RBAC policies.
func GetSettingsRBACValidateCmd() *cobra.Command {
	return settingsRBACValidateCmd
}
