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

// Package commands implements main logic for cli commands.
package commands

import (
	"github.com/go-logr/zapr"
	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/cobra"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/logger"
)

type (
	globalFlags struct {
		Verbose bool // Enable Verbose mode
		JSON    bool // Set output type to JSON
		// If set, we will print the Pretty output.
		Pretty bool
		// Path to a kubeconfig
		KubeconfigPath string `default:"~/.kube/config" envconfig:"KUBECONFIG"`
	}
)

var (
	rootCmd = &cobra.Command{
		Use:              "everestctl <command> [flags]",
		Long:             "CLI for managing Percona Everest",
		Short:            "CLI for managing Percona Everest",
		PersistentPreRun: rootPersistentPreRun,
	}
	// Contains global variables applied to all commands and subcommands.
	rootCmdFlags = &globalFlags{}
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&rootCmdFlags.Verbose, cli.FlagVerbose, "v", false, "Enable Verbose mode")
	rootCmd.PersistentFlags().BoolVar(&rootCmdFlags.JSON, cli.FlagJSON, false, "Set output type to JSON")

	// Read KUBECONFIG ENV var first.
	_ = envconfig.Process("", rootCmdFlags)
	// if kubeconfig is passed explicitly via CLI - use it instead of ENV var.
	rootCmd.PersistentFlags().StringVarP(&rootCmdFlags.KubeconfigPath, cli.FlagKubeconfig, "k", rootCmdFlags.KubeconfigPath, "Path to a kubeconfig. If not set, will use KUBECONFIG env var")
}

func rootPersistentPreRun(_ *cobra.Command, _ []string) { //nolint:revive
	logger.InitLoggerInRootCmd(rootCmdFlags.Verbose, rootCmdFlags.JSON, "everestctl")

	// This is required because controller-runtime requires a logger
	// to be set within 30 seconds of the program initialization.
	ctrlruntimelog.SetLogger(zapr.NewLogger(logger.GetLogger().Desugar()))

	rootCmdFlags.Pretty = !(rootCmdFlags.Verbose || rootCmdFlags.JSON)
	logger.GetLogger().Debug("Debug logging enabled")
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
