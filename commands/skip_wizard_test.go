package commands

import (
	"testing"

	"github.com/percona/everest/pkg/cli"
	"github.com/percona/everest/pkg/common"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func newSkipWizardTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String(cli.FlagNamespaces, common.DefaultDBNamespaceName, "Comma-separated namespaces list")
	cmd.Flags().Bool(cli.FlagOperatorMongoDB, true, "Install MongoDB operator")
	cmd.Flags().Bool(cli.FlagOperatorPostgresql, true, "Install PostgreSQL operator")
	cmd.Flags().Bool(cli.FlagOperatorXtraDBCluster, true, "Install XtraDB Cluster operator")
	cmd.Flags().Bool(cli.FlagOperatorMySQL, true, "Install MySQL operator")
	return cmd
}

func TestShouldAskNamespaces(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name       string
		skipWizard bool
		setFlag    bool
		flagValue  string
		expected   bool
	}

	tcases := []tcase{
		{
			name:       "skip-wizard disabled, flag not set - should ask",
			skipWizard: false,
			setFlag:    false,
			expected:   true,
		},
		{
			name:       "skip-wizard enabled, flag not set - should not ask",
			skipWizard: true,
			setFlag:    false,
			expected:   false,
		},
		{
			name:       "skip-wizard disabled, flag explicitly set to custom - should not ask",
			skipWizard: false,
			setFlag:    true,
			flagValue:  "custom-ns",
			expected:   false,
		},
		{
			name:       "skip-wizard enabled, flag explicitly set - should not ask",
			skipWizard: true,
			setFlag:    true,
			flagValue:  "custom-ns",
			expected:   false,
		},
		{
			name:       "skip-wizard disabled, flag set to empty string - should not ask",
			skipWizard: false,
			setFlag:    true,
			flagValue:  "",
			expected:   false,
		},
		{
			name:       "skip-wizard disabled, flag set to default value - should not ask",
			skipWizard: false,
			setFlag:    true,
			flagValue:  common.DefaultDBNamespaceName,
			expected:   false,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := newSkipWizardTestCommand()
			if tc.setFlag {
				assert.NoError(t, cmd.Flags().Set(cli.FlagNamespaces, tc.flagValue))
			}
			result := shouldAskNamespaces(cmd, tc.skipWizard)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestShouldAskOperators(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name           string
		skipWizard     bool
		flagsToSet     map[string]string // flag name -> value
		expectedResult bool
	}

	tcases := []tcase{
		{
			name:           "skip-wizard disabled, no flags set - should ask",
			skipWizard:     false,
			flagsToSet:     map[string]string{},
			expectedResult: true,
		},
		{
			name:           "skip-wizard enabled, no flags set - should not ask",
			skipWizard:     true,
			flagsToSet:     map[string]string{},
			expectedResult: false,
		},
		{
			name:       "skip-wizard disabled, MongoDB operator flag set to false - should not ask",
			skipWizard: false,
			flagsToSet: map[string]string{
				cli.FlagOperatorMongoDB: "false",
			},
			expectedResult: false,
		},
		{
			name:       "skip-wizard disabled, MongoDB operator flag set to true - should not ask",
			skipWizard: false,
			flagsToSet: map[string]string{
				cli.FlagOperatorMongoDB: "true",
			},
			expectedResult: false,
		},
		{
			name:       "skip-wizard disabled, PostgreSQL operator flag set - should not ask",
			skipWizard: false,
			flagsToSet: map[string]string{
				cli.FlagOperatorPostgresql: "true",
			},
			expectedResult: false,
		},
		{
			name:       "skip-wizard disabled, MySQL operator flag set - should not ask",
			skipWizard: false,
			flagsToSet: map[string]string{
				cli.FlagOperatorMySQL: "false",
			},
			expectedResult: false,
		},
		{
			name:       "skip-wizard disabled, XtraDBCluster operator flag set - should not ask",
			skipWizard: false,
			flagsToSet: map[string]string{
				cli.FlagOperatorXtraDBCluster: "true",
			},
			expectedResult: false,
		},
		{
			name:       "skip-wizard disabled, multiple operator flags set - should not ask",
			skipWizard: false,
			flagsToSet: map[string]string{
				cli.FlagOperatorMongoDB:    "true",
				cli.FlagOperatorPostgresql: "false",
				cli.FlagOperatorMySQL:      "true",
			},
			expectedResult: false,
		},
		{
			name:       "skip-wizard enabled, one operator flag set - should not ask",
			skipWizard: true,
			flagsToSet: map[string]string{
				cli.FlagOperatorMongoDB: "true",
			},
			expectedResult: false,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := newSkipWizardTestCommand()
			for flag, value := range tc.flagsToSet {
				assert.NoError(t, cmd.Flags().Set(flag, value))
			}
			result := cli.ShouldAskOperators(cmd, tc.skipWizard)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
