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

// Package commands ...
package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli/status"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	statusCmd = &cobra.Command{
		Use:   "status [flags]",
		Args:  cobra.NoArgs,
		Short: "Show health status of Everest components",
		Long:  "Show health status of all Everest components including core services, OLM, monitoring, and database operators",
		Run:   statusRun,
	}
	statusCfg = status.StatusConfig{
		Pretty: true,
	}
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

func statusRun(cmd *cobra.Command, _ []string) { //nolint:revive
	statusCfg.Pretty = rootCmdFlags.Pretty
	statusCfg.JSON = rootCmdFlags.JSON
	statusCfg.KubeconfigPath = rootCmdFlags.KubeconfigPath

	op, err := status.NewStatus(statusCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), statusCfg.Pretty)
		os.Exit(1)
	}

	if err := op.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), statusCfg.Pretty)
		os.Exit(1)
	}
}
