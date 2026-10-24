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

package reconciler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestNewManagerOptions(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	t.Run("leader election disabled by default", func(t *testing.T) {
		t.Parallel()

		options := newManagerOptions(scheme, "postgresql", reconcilerOptions{})

		assert.False(t, options.LeaderElection)
		assert.Empty(t, options.LeaderElectionID)
		assert.False(t, options.LeaderElectionReleaseOnCancel)
	})

	t.Run("leader election enabled", func(t *testing.T) {
		t.Parallel()

		options := &reconcilerOptions{}
		WithLeaderElection()(options)

		managerOptions := newManagerOptions(scheme, "postgresql", *options)

		assert.True(t, managerOptions.LeaderElection)
		assert.Equal(t, "postgresql.provider.openeverest.io", managerOptions.LeaderElectionID)
		assert.True(t, managerOptions.LeaderElectionReleaseOnCancel)
	})
}

func TestProviderLeaderElectionID(t *testing.T) {
	t.Parallel()

	postgresqlID := providerLeaderElectionID("postgresql")
	mongodbID := providerLeaderElectionID("percona-server-mongodb")

	assert.Equal(t, postgresqlID, providerLeaderElectionID("postgresql"))
	assert.NotEqual(t, postgresqlID, mongodbID)
}
