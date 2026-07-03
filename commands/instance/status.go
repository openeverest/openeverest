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

package instance

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	instancecli "github.com/openeverest/openeverest/v2/pkg/cli/instance"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

var (
	statusCmd = &cobra.Command{
		Use:   "status [flags]",
		Args:  cobra.NoArgs,
		Short: "Show the status of an instance",
		Long: `Fetch and display the current status of an instance.

Shows the phase, version, per-component ready/total counts, and Kubernetes
conditions reported by the Everest API. Pass --json / --verbose to get the
raw API response instead of the formatted table.`,
		Example: `  # Status of an instance in the default cluster
  everestctl instance status --name my-mongo --namespace everest

  # Raw JSON output
  everestctl instance status --name my-mongo --namespace everest --json

  # Specific cluster and context
  everestctl instance status --name my-mongo --namespace everest \
    --cluster staging --context staging-ctx`,
		PreRun: statusPreRun,
		Run:    statusRun,
	}
	statusCfg  = &instancecli.Config{}
	statusOpts = &instancecli.StatusOptions{}
)

func init() {
	statusCmd.Flags().StringVar(&statusOpts.Name, cli.FlagInstanceName, "", "Instance name (required)")
	statusCmd.Flags().StringVar(&statusOpts.Namespace, cli.FlagInstanceNamespace, "", "Namespace the instance lives in (required)")
	statusCmd.Flags().StringVar(&statusOpts.Cluster, cli.FlagInstanceCluster, "main", "Cluster name")
	statusCmd.Flags().StringVar(&statusOpts.Context, cli.FlagInstanceContext, "", "Context to use (default: current context)")

	_ = statusCmd.MarkFlagRequired(cli.FlagInstanceName)
	_ = statusCmd.MarkFlagRequired(cli.FlagInstanceNamespace)
}

func statusPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	statusCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
}

func statusRun(cmd *cobra.Command, _ []string) { //nolint:revive
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), statusCfg.Pretty)
		os.Exit(1)
	}

	runner := instancecli.NewInstanceStatusRunner(*statusCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *statusOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), statusCfg.Pretty)
		os.Exit(1)
	}
}

// GetStatusCmd returns the status command.
func GetStatusCmd() *cobra.Command {
	return statusCmd
}
