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

// Package commands ...
package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/uninstall"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	uninstallCmd = &cobra.Command{
		Use:    "uninstall [flags]",
		Args:   cobra.NoArgs,
		Long:   "Uninstall Percona Everest",
		Short:  "Uninstall Percona Everest",
		PreRun: uninstallPreRun,
		Run:    uninstallRun,
	}
	uninstallCfg = &uninstall.Config{}
)

func init() {
	rootCmd.AddCommand(uninstallCmd)

	// local command flags
	uninstallCmd.Flags().BoolVarP(&uninstallCfg.AssumeYes, "assume-yes", "y", false, "Assume yes to all questions")
	uninstallCmd.Flags().BoolVarP(&uninstallCfg.Force, "force", "f", false, "Force removal in case there are database clusters running")
	uninstallCmd.Flags().BoolVar(&uninstallCfg.SkipEnvDetection, cli.FlagSkipEnvDetection, false, "Skip detecting Kubernetes environment where Everest is installed")
}

func uninstallPreRun(_ *cobra.Command, _ []string) { //nolint:revive
	// Copy global flags to config
	uninstallCfg.Pretty = rootCmdFlags.Pretty
	uninstallCfg.KubeconfigPath = rootCmdFlags.KubeconfigPath
}

func uninstallRun(cmd *cobra.Command, _ []string) { //nolint:revive
	op, err := uninstall.NewUninstall(*uninstallCfg, logger.GetLogger())
	if err != nil {
		output.PrintError(err, logger.GetLogger(), uninstallCfg.Pretty)
		os.Exit(1)
	}

	if err := op.Run(cmd.Context()); err != nil {
		output.PrintError(err, logger.GetLogger(), uninstallCfg.Pretty)
		os.Exit(1)
	}
}
