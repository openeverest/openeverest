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

package commands

import (
	"github.com/spf13/cobra"

	"github.com/openeverest/openeverest/v2/commands/clusterpreset"
	"github.com/openeverest/openeverest/v2/commands/common"
)

var clusterPresetCmd = &cobra.Command{
	Use:     "cluster-preset <command> [flags]",
	Aliases: []string{"cluster-instance-preset", "clusterpreset", "cip"},
	Args:    common.NoSubcommandArgs,
	Short:   "Manage Everest cluster instance presets",
	Long: `Manage cluster-scoped ClusterInstancePreset resources.

The Everest API currently exposes list and get for cluster instance presets.
Create, update and delete are available for namespace-scoped InstancePresets
via 'everestctl preset'.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(clusterPresetCmd)
	clusterPresetCmd.AddCommand(clusterpreset.GetListCmd())
	clusterPresetCmd.AddCommand(clusterpreset.GetGetCmd())
}
