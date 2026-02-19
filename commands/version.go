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

package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/logger"
	"github.com/openeverest/openeverest/v2/pkg/version"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version [flags]",
		Args:  cobra.NoArgs,
		Long:  "Print Everest components version info",
		Short: "Print Everest components version info",
		RunE:  versionRunE,
	}

	// Command flag values
	clientOnlyFlag bool // Means only info about the client shall be printed
)

func init() {
	rootCmd.AddCommand(versionCmd)

	// local command flags
	versionCmd.Flags().BoolVar(&clientOnlyFlag, "client-only", false, "Print client version only")
}

func versionRunE(cmd *cobra.Command, _ []string) error { //nolint:revive
	v := version.Info{
		ProjectName: version.ProjectName,
		Version:     version.Version,
		FullCommit:  version.FullCommit,
	}
	cmdLogger := logger.GetLogger().With("component", "version")

	if !clientOnlyFlag {
		k, err := utils.NewKubeConnector(cmdLogger, rootCmdFlags.KubeconfigPath)
		if err != nil {
			return err
		}

		ev, err := version.EverestVersionFromDeployment(cmd.Context(), k)
		if client.IgnoreNotFound(err) != nil {
			cmdLogger.Error(err)
			return err
		}
		sv := "[NOT INSTALLED]"
		if ev != nil {
			sv = fmt.Sprintf("v%s", ev.String())
		}
		v.ServerVersion = &sv
	}

	if rootCmdFlags.JSON {
		_, _ = fmt.Fprintln(os.Stdout, v.JSONString())
		return nil
	}
	_, _ = fmt.Fprintln(os.Stdout, v)
	return nil
}
