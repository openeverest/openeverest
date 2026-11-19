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

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewRemote(t *testing.T) {
	t.Parallel()

	connector, err := NewRemote(RemoteClusterConfig{
		Endpoint:      "https://agent.example.test",
		BearerToken:   "token",
		TLSServerName: "agent.example.test",
	}, zap.NewNop().Sugar(), "everest-system")
	require.NoError(t, err)
	require.Equal(t, "https://agent.example.test", connector.Config().Host)
	require.Equal(t, "token", connector.Config().BearerToken)
	require.Equal(t, "agent.example.test", connector.Config().TLSClientConfig.ServerName)
	require.Equal(t, "everest-system", connector.Namespace())
}

func TestNewRemoteRequiresConnectionData(t *testing.T) {
	t.Parallel()

	_, err := NewRemote(RemoteClusterConfig{BearerToken: "token"}, zap.NewNop().Sugar(), "system")
	require.EqualError(t, err, "remote endpoint is required")

	_, err = NewRemote(RemoteClusterConfig{Endpoint: "https://agent.example.test"}, zap.NewNop().Sugar(), "system")
	require.EqualError(t, err, "remote bearer token is required")
}
