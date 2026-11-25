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

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOutputTestCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	var output OutputFormat
	cmd := &cobra.Command{
		Use: "test",
		Run: func(*cobra.Command, []string) {},
	}
	cmd.Flags().Bool(FlagVerbose, false, "")
	cmd.Flags().VarP(&output, FlagOutput, "o", "")
	cmd.SetArgs(args)
	require.NoError(t, cmd.Execute())
	return cmd
}

func TestOutputModeHelpers(t *testing.T) {
	t.Parallel()

	t.Run("default is pretty", func(t *testing.T) {
		t.Parallel()
		cmd := newOutputTestCmd(t)
		assert.True(t, IsPretty(cmd))
		assert.False(t, IsYAML(cmd))
		assert.False(t, IsMachineReadable(cmd))
	})

	t.Run("verbose is not pretty", func(t *testing.T) {
		t.Parallel()
		cmd := newOutputTestCmd(t, "--verbose")
		assert.False(t, IsPretty(cmd))
		assert.False(t, IsMachineReadable(cmd))
	})

	t.Run("json is machine-readable", func(t *testing.T) {
		t.Parallel()
		cmd := newOutputTestCmd(t, "-o", "json")
		assert.False(t, IsPretty(cmd))
		assert.False(t, IsYAML(cmd))
		assert.True(t, IsMachineReadable(cmd))
	})

	t.Run("yaml is machine-readable", func(t *testing.T) {
		t.Parallel()
		cmd := newOutputTestCmd(t, "--output", "yaml")
		assert.False(t, IsPretty(cmd))
		assert.True(t, IsYAML(cmd))
		assert.True(t, IsMachineReadable(cmd))
	})
}
