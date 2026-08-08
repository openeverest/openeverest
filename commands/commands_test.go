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
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// parentCommands are the commands that exist only to group subcommands.
var parentCommands = [][]string{
	{"namespaces"},
	{"settings"},
	{"accounts"},
	{"settings", "rbac"},
	{"settings", "oidc"},
}

// execRoot runs the root command with the given arguments and returns the error
// main() would turn into a non-zero exit code. The commands share the package-level
// rootCmd, so these must not run in parallel.
func execRoot(t *testing.T, args ...string) error {
	t.Helper()

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})
	return rootCmd.Execute()
}

// Cobra checks Runnable() before it validates Args, so a parent command with no
// Run at all returns flag.ErrHelp and exits 0 whatever it is given, and one with
// an empty Run exits 0 silently. Either way a typo looks like success, which
// hides mistakes from CI scripts that gate on the exit code.
//
//nolint:paralleltest // execRoot mutates the package-level rootCmd.
func TestParentCommandsRejectUnknownSubcommand(t *testing.T) {
	for _, parent := range parentCommands {
		path := strings.Join(parent, " ")
		t.Run(path, func(t *testing.T) {
			err := execRoot(t, append(parent, "bogus-subcommand")...)
			require.Error(t, err, "%s accepted an unknown subcommand", path)
			require.Contains(t, err.Error(), `unknown command "bogus-subcommand"`)
		})
	}
}

// Invoked with no subcommand, a parent command should print its help and succeed.
//
//nolint:paralleltest // execRoot mutates the package-level rootCmd.
func TestParentCommandsWithoutArgsPrintHelp(t *testing.T) {
	for _, parent := range parentCommands {
		path := strings.Join(parent, " ")
		t.Run(path, func(t *testing.T) {
			require.NoError(t, execRoot(t, parent...))
		})
	}
}

// Rejecting positional args on the parent must not stop it from routing to a
// real subcommand.
//
//nolint:paralleltest // reads the package-level rootCmd.
func TestParentCommandsRouteToSubcommands(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"namespaces", "list"})
	require.NoError(t, err)
	require.Equal(t, "list", cmd.Name())
}
