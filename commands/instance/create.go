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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	instancecli "github.com/openeverest/openeverest/v2/pkg/cli/instance"
	"github.com/openeverest/openeverest/v2/pkg/cli/waitcmd"
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

  # Bootstrap from a preset (provider inferred; version/topology/components from preset)
  everestctl instance create --name my-mongo --namespace everest --preset mongodb-production

  # Preset as base, override a field on top
  everestctl instance create --name my-mongo --namespace everest --preset mongodb-production \
    --set components.engine.replicas=5

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
    --version 8.0.12 --topology sharded

  # Block until the instance is ready (Ctrl-C only stops waiting; the instance keeps provisioning)
  everestctl instance create --name my-mongo --namespace everest --provider percona-server-mongodb --wait

  # Bounded wait for CI, then grab the phase
  everestctl instance create --name my-mongo --namespace everest --provider percona-server-mongodb \
    --wait --timeout 15m --json | jq '.status.phase'`,
		PreRun: createPreRun,
		Run:    createRun,
	}
	createCfg  = &instancecli.Config{}
	createOpts = &instancecli.CreateOptions{}
)

func init() {
	createCmd.Flags().StringVar(&createOpts.Name, cli.FlagInstanceName, "", "Instance name (required)")
	createCmd.Flags().StringVar(&createOpts.Namespace, cli.FlagInstanceNamespace, "", "Namespace to create the instance in (required)")
	createCmd.Flags().StringVar(&createOpts.Provider, cli.FlagInstanceProvider, "", "Provider name, e.g. percona-server-mongodb (required unless --preset is given)")
	createCmd.Flags().StringVar(&createOpts.Preset, cli.FlagInstancePreset, "", "InstancePreset name to bootstrap from; --provider is inferred from the preset when omitted")
	createCmd.Flags().StringVar(&createOpts.Cluster, cli.FlagInstanceCluster, "main", "Cluster name")
	createCmd.Flags().StringVar(&createOpts.Version, cli.FlagInstanceVersion, "", "Provider version bundle to use (default: taken from preset if --preset is given, otherwise provider's default)")
	createCmd.Flags().StringVar(&createOpts.Topology, cli.FlagInstanceTopology, "", "Topology name; cannot be used with --preset (default: provider's first topology)")
	createCmd.Flags().StringVar(&createOpts.Context, cli.FlagInstanceContext, "", "Context to use (default: current context)")
	createCmd.Flags().StringVarP(&createOpts.ValuesFile, cli.FlagInstanceFile, "f", "", "Path to a YAML file with component overrides (--set takes precedence)")
	createCmd.Flags().StringArrayVar(&createOpts.Set, cli.FlagInstanceSet, nil, "Set a spec field: --set components.engine.replicas=3 or --set backup.enabled=true (repeatable)")
	createCmd.Flags().BoolVar(&createOpts.Wait, cli.FlagInstanceWait, false, "Block until the instance reaches the Ready phase; exit non-zero if it fails or times out")
	createCmd.Flags().DurationVar(&createOpts.Timeout, cli.FlagInstanceTimeout, 10*time.Minute, "Maximum time to wait (only valid with --wait); must be positive")

	_ = createCmd.MarkFlagRequired(cli.FlagInstanceName)
	_ = createCmd.MarkFlagRequired(cli.FlagInstanceNamespace)
}

func createPreRun(cmd *cobra.Command, _ []string) { //nolint:revive
	createCfg.Pretty = !(cmd.Flag(cli.FlagVerbose).Changed || cmd.Flag(cli.FlagJSON).Changed)
}

func createRun(cmd *cobra.Command, _ []string) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(waitcmd.ExitFailed)
	}

	if err := waitcmd.ValidateWaitFlags(createOpts.Wait, cmd.Flags().Changed(cli.FlagInstanceTimeout), createOpts.Timeout); err != nil {
		output.PrintError(err, logger.GetLogger(), createCfg.Pretty)
		os.Exit(waitcmd.ExitFailed)
	}

	ctx := cmd.Context()
	stop := func() {}
	if createOpts.Wait {
		// Ctrl-C cancels only the wait; the instance keeps provisioning.
		var cancel context.CancelFunc
		ctx, cancel = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		stop = cancel
	}

	ic := instancecli.NewInstanceCreator(*createCfg, logger.GetLogger())
	runErr := ic.Run(ctx, *createOpts, cfgPath)
	// Release the signal handler before os.Exit (which skips defers).
	stop()
	os.Exit(waitcmd.ExitCode(runErr, logger.GetLogger(), createCfg.Pretty,
		fmt.Sprintf("wait cancelled; instance %q is still provisioning — check with 'everestctl instance status'", createOpts.Name),
		fmt.Sprintf("timed out after %s waiting for instance %q to become ready; it is still provisioning", createOpts.Timeout, createOpts.Name),
	))
}

// GetCreateCmd returns the create command.
func GetCreateCmd() *cobra.Command {
	return createCmd
}
