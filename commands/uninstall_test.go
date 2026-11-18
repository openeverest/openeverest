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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/everest/pkg/cli"
)

// TestUninstallCmdSkipEnvDetectionDeprecated is a regression test for
// issue #2888: --skip-env-detection must remain accepted by `everestctl
// uninstall` (so existing scripts don't fail with an "unknown flag" error),
// but must now be treated as a deprecated no-op.
//
//nolint:paralleltest
func TestUninstallCmdSkipEnvDetectionDeprecated(t *testing.T) {
	// Still parses without error, preserving compatibility with existing
	// invocations of --skip-env-detection.
	err := uninstallCmd.ParseFlags([]string{"--" + cli.FlagSkipEnvDetection})
	require.NoError(t, err)
	assert.True(t, uninstallCfg.SkipEnvDetection)

	// The flag is marked deprecated, so cobra will print a warning on use
	// and omit it from `--help` output.
	flag := uninstallCmd.Flags().Lookup(cli.FlagSkipEnvDetection)
	require.NotNil(t, flag)
	assert.NotEmpty(t, flag.Deprecated, "--skip-env-detection should be marked deprecated")
}
