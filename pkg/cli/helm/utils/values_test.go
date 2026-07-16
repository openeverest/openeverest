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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValues(t *testing.T) {
	t.Parallel()
	in := map[string]any{
		"server": map[string]any{
			"tls": map[string]any{
				"enabled": true,
			},
			"service": map[string]any{
				"name": "test",
				"port": 8080,
			},
		},
	}

	result, err := ParseValues(in)
	require.NoError(t, err)
	assert.True(t, result.Server.TLS.Enabled)
	assert.Equal(t, "test", result.Server.Service.Name)
	assert.Equal(t, 8080, result.Server.Service.Port)
}
