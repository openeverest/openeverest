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

// Package cli contains interactive CLI helpers.
package cli

import "github.com/spf13/cobra"

// ShouldAskOperators determines if the CLI should prompt for operator selection.
// It returns true if skipWizard is false and no operator flags were explicitly set.
func ShouldAskOperators(cmd *cobra.Command, skipWizard bool) bool {
	return !skipWizard && !(cmd.Flags().Lookup(FlagOperatorMongoDB).Changed ||
		cmd.Flags().Lookup(FlagOperatorPostgresql).Changed ||
		cmd.Flags().Lookup(FlagOperatorXtraDBCluster).Changed ||
		cmd.Flags().Lookup(FlagOperatorMySQL).Changed)
}