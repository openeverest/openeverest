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
	getCmd = &cobra.Command{
		Use:     "get [flags]",
		Aliases: []string{"describe"},
		Args:    cobra.NoArgs,
		Short:   "Get an instance",
		Long: `Fetch a single Instance resource via the Everest API.

Prints the full API object as JSON (default) or YAML (--yaml). For a
human-readable status summary, use "instance status" instead.`,
		Example: `  # Fetch an instance
  everestctl instance get --name my-mongo --namespace everest

  # YAML output
  everestctl instance get --name my-mongo --namespace everest --yaml

  # Scripting — pick a field
  everestctl instance get --name my-mongo --namespace everest --json \
    | jq -r '.status.phase'`,
		PreRun: getPreRun,
		Run:    getRun,
	}
	getCfg  = &instancecli.Config{}
	getOpts = &instancecli.GetOptions{}
)

func init() {
	getCmd.Flags().StringVar(&getOpts.Name, cli.FlagInstanceName, "", "Instance name (required)")
	getCmd.Flags().StringVarP(&getOpts.Namespace, cli.FlagInstanceNamespace, "n", "", "Namespace the instance lives in (required)")
	getCmd.Flags().StringVar(&getOpts.Cluster, cli.FlagInstanceCluster, "main", "Cluster name")
	getCmd.Flags().StringVar(&getOpts.Context, cli.FlagInstanceContext, "", "Context to use (default: current context)")

	_ = getCmd.MarkFlagRequired(cli.FlagInstanceName)
	_ = getCmd.MarkFlagRequired(cli.FlagInstanceNamespace)
}

func getPreRun(cmd *cobra.Command, _ []string) {
	getCfg.Pretty = cli.IsPretty(cmd)
	getCfg.YAML = cli.IsYAML(cmd)
}

func getRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), getCfg.Pretty)
		os.Exit(1)
	}

	runner := instancecli.NewGetRunner(*getCfg, logger.GetLogger())
	if err := runner.Run(cmd.Context(), *getOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), getCfg.Pretty)
		os.Exit(1)
	}
}

// GetGetCmd returns the get command.
func GetGetCmd() *cobra.Command {
	return getCmd
}
