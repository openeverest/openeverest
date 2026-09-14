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

// Package common contains common types for all commands.
package common

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NoSubcommandArgs rejects positional arguments on commands that exist only to
// group subcommands, mirroring the "unknown command" error Cobra produces at the root,
// suggestions included.
func NoSubcommandArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "unknown command %q for %q", args[0], cmd.CommandPath())
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
		sb.WriteString("\n\nDid you mean this?\n")
		for _, s := range suggestions {
			fmt.Fprintf(&sb, "\t%s %s\n", cmd.CommandPath(), s)
		}
	}
	return errors.New(sb.String())
}
