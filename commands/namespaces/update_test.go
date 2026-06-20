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

package namespaces

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/percona/everest/pkg/cli"
	nscli "github.com/percona/everest/pkg/cli/namespaces"
)

func Test_shouldPromptOperatorsForNamespaceUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		operatorFlagsChanged bool
		skipWizard           bool
		wantPrompt           bool
	}{
		{
			name:                 "skip wizard never prompts",
			operatorFlagsChanged: false,
			skipWizard:           true,
			wantPrompt:           false,
		},
		{
			name:                 "skip wizard does not prompt even if flags were passed",
			operatorFlagsChanged: true,
			skipWizard:           true,
			wantPrompt:           false,
		},
		{
			name:                 "interactive prompts when no operator flags and not skip wizard",
			operatorFlagsChanged: false,
			skipWizard:           false,
			wantPrompt:           true,
		},
		{
			name:                 "no prompt when an operator flag was set",
			operatorFlagsChanged: true,
			skipWizard:           false,
			wantPrompt:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shouldPromptOperatorsForNamespaceUpdate(tt.operatorFlagsChanged, tt.skipWizard)
			assert.Equal(t, tt.wantPrompt, got)
		})
	}
}

func Test_shouldUseInstalledOperatorsForNamespaceUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		skipWizard bool
		want       bool
	}{
		{
			name:       "use installed operators in skip wizard mode",
			skipWizard: true,
			want:       true,
		},
		{
			name:       "do not use installed operators in interactive mode",
			skipWizard: false,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shouldUseInstalledOperatorsForNamespaceUpdate(tt.skipWizard)
			assert.Equal(t, tt.want, got)
		})
	}
}

func newUpdateCmdWithOperatorFlags(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "update"}
	var mongo, pg, pxcLegacy, mysql bool
	cmd.Flags().BoolVar(&mongo, cli.FlagOperatorMongoDB, true, "")
	cmd.Flags().BoolVar(&pg, cli.FlagOperatorPostgresql, true, "")
	cmd.Flags().BoolVar(&pxcLegacy, cli.FlagOperatorXtraDBCluster, true, "")
	cmd.Flags().BoolVar(&mysql, cli.FlagOperatorMySQL, true, "")
	return cmd
}

func Test_operatorFlagsChangedOnUpdateCommand(t *testing.T) {
	t.Parallel()

	t.Run("unchanged defaults", func(t *testing.T) {
		t.Parallel()
		cmd := newUpdateCmdWithOperatorFlags(t)
		err := cmd.ParseFlags([]string{})
		assert.NoError(t, err)
		assert.False(t, operatorFlagsChangedOnUpdateCommand(cmd))
	})

	t.Run("operator.mysql explicitly set", func(t *testing.T) {
		t.Parallel()
		cmd := newUpdateCmdWithOperatorFlags(t)
		err := cmd.ParseFlags([]string{"--" + cli.FlagOperatorMySQL + "=false"})
		assert.NoError(t, err)
		assert.True(t, operatorFlagsChangedOnUpdateCommand(cmd))
	})

	t.Run("operator.postgresql explicitly set", func(t *testing.T) {
		t.Parallel()
		cmd := newUpdateCmdWithOperatorFlags(t)
		err := cmd.ParseFlags([]string{"--" + cli.FlagOperatorPostgresql + "=true"})
		assert.NoError(t, err)
		assert.True(t, operatorFlagsChangedOnUpdateCommand(cmd))
	})
}

func Test_operatorFlagsChangedDetailsOnUpdateCommand(t *testing.T) {
	t.Parallel()

	t.Run("no operator flags changed", func(t *testing.T) {
		t.Parallel()
		cmd := newUpdateCmdWithOperatorFlags(t)
		err := cmd.ParseFlags([]string{})
		assert.NoError(t, err)
		got := operatorFlagsChangedDetailsOnUpdateCommand(cmd)
		assert.Equal(t, nscli.OperatorFlagsChanged{}, got)
	})

	t.Run("single mysql flag changed", func(t *testing.T) {
		t.Parallel()
		cmd := newUpdateCmdWithOperatorFlags(t)
		err := cmd.ParseFlags([]string{"--" + cli.FlagOperatorMySQL + "=true"})
		assert.NoError(t, err)
		got := operatorFlagsChangedDetailsOnUpdateCommand(cmd)
		assert.Equal(t, nscli.OperatorFlagsChanged{PXC: true}, got)
	})

	t.Run("mysql via deprecated flag still tracked as pxc override", func(t *testing.T) {
		t.Parallel()
		cmd := newUpdateCmdWithOperatorFlags(t)
		err := cmd.ParseFlags([]string{"--" + cli.FlagOperatorXtraDBCluster + "=false"})
		assert.NoError(t, err)
		got := operatorFlagsChangedDetailsOnUpdateCommand(cmd)
		assert.Equal(t, nscli.OperatorFlagsChanged{PXC: true}, got)
	})
}
