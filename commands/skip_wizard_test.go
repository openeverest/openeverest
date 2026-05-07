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

	cmd := newSkipWizardTestCommand()
	assert.True(t, shouldAskNamespaces(cmd, false), "should ask namespaces when skip-wizard is disabled and namespaces flag was not explicitly set")
	assert.False(t, shouldAskNamespaces(cmd, true), "should not ask namespaces when skip-wizard is enabled")

	assert.NoError(t, cmd.Flags().Set(cli.FlagNamespaces, "custom-ns"))
	assert.False(t, shouldAskNamespaces(cmd, false), "should not ask namespaces when namespaces flag is explicitly set")
}

func TestShouldAskOperators(t *testing.T) {
	t.Parallel()

	cmd := newSkipWizardTestCommand()
	assert.True(t, cli.ShouldAskOperators(cmd, false), "should ask operators when skip-wizard is disabled and no operator flag was explicitly set")
	assert.False(t, cli.ShouldAskOperators(cmd, true), "should not ask operators when skip-wizard is enabled")

	assert.NoError(t, cmd.Flags().Set(cli.FlagOperatorMongoDB, "false"))
	assert.False(t, cli.ShouldAskOperators(cmd, false), "should not ask operators when any operator flag was explicitly set")
}
