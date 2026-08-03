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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/everest/pkg/cli/upgrade"
)

// TestUpgradeCmd_HelmValueFlagBindings guards the binding of the --helm.reuse-values
// and --helm.reset-then-reuse-values flags to the Helm CLIOptions fields.
// Regression test for #2199 where --helm.reuse-values was incorrectly bound to
// ResetThenReuseValues, silently changing Helm upgrade precedence semantics.
func TestUpgradeCmd_HelmValueFlagBindings(t *testing.T) {
	tests := []struct {
		name                     string
		args                     []string
		wantReuseValues          bool
		wantResetThenReuseValues bool
	}{
		{
			name:                     "--helm.reuse-values sets only ReuseValues",
			args:                     []string{"--helm.reuse-values"},
			wantReuseValues:          true,
			wantResetThenReuseValues: false,
		},
		{
			name:                     "--helm.reset-then-reuse-values sets only ResetThenReuseValues",
			args:                     []string{"--helm.reset-then-reuse-values"},
			wantReuseValues:          false,
			wantResetThenReuseValues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*upgradeCfg = upgrade.Config{}

			require.NoError(t, upgradeCmd.ParseFlags(tt.args))

			assert.Equal(t, tt.wantReuseValues, upgradeCfg.ReuseValues, "ReuseValues")
			assert.Equal(t, tt.wantResetThenReuseValues, upgradeCfg.ResetThenReuseValues, "ResetThenReuseValues")
		})
	}
}
