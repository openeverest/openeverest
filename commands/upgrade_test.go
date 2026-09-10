// everest
// Copyright (C) 2023 Percona LLC
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

	"github.com/openeverest/openeverest/v2/pkg/cli/helm"
)

//nolint:paralleltest
func TestUpgradeCmdHelmFlagsBinding(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		expectedReuse      bool
		expectedReset      bool
		expectedResetReuse bool
	}{
		{
			name:               "helm.reuse-values flag sets ReuseValues",
			args:               []string{"--" + helm.FlagHelmReuseValues},
			expectedReuse:      true,
			expectedReset:      false,
			expectedResetReuse: false,
		},
		{
			name:               "helm.reset-values flag sets ResetValues",
			args:               []string{"--" + helm.FlagHelmResetValues},
			expectedReuse:      false,
			expectedReset:      true,
			expectedResetReuse: false,
		},
		{
			name:               "helm.reset-then-reuse-values flag sets ResetThenReuseValues",
			args:               []string{"--" + helm.FlagHelmResetThenReuseValues},
			expectedReuse:      false,
			expectedReset:      false,
			expectedResetReuse: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset configuration state before parsing flags
			upgradeCfg.ReuseValues = false
			upgradeCfg.ResetValues = false
			upgradeCfg.ResetThenReuseValues = false

			err := upgradeCmd.ParseFlags(tc.args)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedReuse, upgradeCfg.ReuseValues)
			assert.Equal(t, tc.expectedReset, upgradeCfg.ResetValues)
			assert.Equal(t, tc.expectedResetReuse, upgradeCfg.ResetThenReuseValues)
		})
	}
}
