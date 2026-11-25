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

package cli

import "github.com/spf13/cobra"

// IsPretty is true when neither --verbose nor -o/--output was passed.
func IsPretty(cmd *cobra.Command) bool {
	return !cmd.Flag(FlagVerbose).Changed && !cmd.Flag(FlagOutput).Changed
}

// IsYAML is true when -o/--output yaml was passed.
func IsYAML(cmd *cobra.Command) bool {
	return cmd.Flag(FlagOutput).Changed && cmd.Flag(FlagOutput).Value.String() == string(OutputYAML)
}

// IsMachineReadable is true when -o/--output was passed (json or yaml).
func IsMachineReadable(cmd *cobra.Command) bool {
	return cmd.Flag(FlagOutput).Changed
}
