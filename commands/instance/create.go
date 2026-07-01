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

// Package instance holds commands for the instance subcommand group.
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
	createCmd = &cobra.Command{
		Use:   "create [flags]",
		Args:  cobra.NoArgs,
		Short: "Create a new instance",
		Long: `Provision a new instance through the Everest API.

The provider, name, and namespace flags are required. Version and topology are
resolved automatically from the provider's defaults if not specified. Use --set
to override any Instance spec field using dot-notation paths rooted at the spec
(e.g. --set components.engine.replicas=3 or --set backup.enabled=true).`,
		Example: `  # Minimal: server defaults for all components
  everestctl instance create --name my-mongo --namespace everest --provider percona-server-mongodb

  # Override component fields via --set
  everestctl instance create --name my-mongo --namespace everest --provider percona-server-mongodb \
    --set components.engine.replicas=3 \
    --set components.engine.storage.size=50Gi

  # Enable backup alongside component config
  everestctl instance create --name my-mongo --namespace everest --provider percona-server-mongodb \
    --set components.engine.replicas=3 \
    --set backup.enabled=true

  # Complex config via values file (like helm -f values.yaml)
  everestctl instance create --name my-mongo --namespace everest --provider percona-server-mongodb \
    -f my-values.yaml

  # File as base, --set overrides specific fields on top
  everestctl instance create --name my-mongo --namespace everest --provider percona-server-mongodb \
    -f my-values.yaml --set components.engine.replicas=5

  # Explicit version and topology
  everestctl instance create --name my-mongo --namespace everest --provider percona-server-mongodb \
    --version 8.0.12 --topology sharded`,
		PreRun: createPreRun,
		Run:    createRun,
	}
	createCfg  = &instancecli.Config{}
	createOpts = &instancecli.CreateOptions{}
)

func init() {
	createCmd.Flags().StringVar(&createOpts.Name, cli.FlagInstanceName, "", "Instance name (required)")
	createCmd.Flags().StringVar(&createOpts.Namespace, cli.FlagInstanceNamespace, "", "Namespace to create the instance in (required)")
	createCmd.Flags().StringVar(&createOpts.Provider, cli.FlagInstanceProvider, "", "Provider name, e.g. percona-server-mongodb, percona-xtradb-cluster (required)")
	createCmd.Flags().StringVar(&createOpts.Cluster, cli.FlagInstanceCluster, "main", "Cluster name")
	createCmd.Flags().StringVar(&createOpts.Version, cli.FlagInstanceVersion, "", "Version bundle name (default: provider's default bundle)")
	createCmd.Flags().StringVar(&createOpts.Topology, cli.FlagInstanceTopology, "", "Topology name (default: provider's first topology)")
	createCmd.Flags().StringVar(&createOpts.Context, cli.FlagInstanceContext, "", "Context to use (default: current context)")
	createCmd.Flags().StringVarP(&createOpts.ValuesFile, cli.FlagInstanceFile, "f", "", "Path to a YAML file with component overrides (--set takes precedence)")
	createCmd.Flags().StringArrayVar(&createOpts.Set, cli.FlagInstanceSet, nil, "Set a spec field: --set components.engine.replicas=3 or --set backup.enabled=true (repeatable)")

	_ = createCmd.MarkFlagRequired(cli.FlagInstanceName)
	_ = createCmd.MarkFlagRequired(cli.FlagInstanceNamespace)
	_ = createCmd.MarkFlagRequired(cli.FlagInstanceProvider)
}

func createPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	createCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
}

func createRun(cmd *cobra.Command, _ []string) { //nolint:revive
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(1)
	}

	ic := instancecli.NewInstanceCreator(*createCfg, logger.GetLogger())
	if err := ic.Run(cmd.Context(), *createOpts, cfgPath); err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(1)
	}
}

// GetCreateCmd returns the create command.
func GetCreateCmd() *cobra.Command {
	return createCmd
}
