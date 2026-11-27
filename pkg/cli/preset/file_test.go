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

package preset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/client"
)

func TestApplyIdentity_FlagsWin(t *testing.T) {
	t.Parallel()

	preset := &client.InstancePreset{
		Metadata: &client.ObjectMeta{Name: "from-file", Namespace: "ns-file"},
	}
	name, ns, err := applyIdentity(preset, "from-flag", "ns-flag")
	require.NoError(t, err)
	assert.Equal(t, "from-flag", name)
	assert.Equal(t, "ns-flag", ns)
	assert.Equal(t, "from-flag", preset.Metadata.Name)
	assert.Equal(t, "ns-flag", preset.Metadata.Namespace)
}

func TestApplyIdentity_RequiresNameAndNamespace(t *testing.T) {
	t.Parallel()

	_, _, err := applyIdentity(&client.InstancePreset{}, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")

	_, _, err = applyIdentity(&client.InstancePreset{
		Metadata: &client.ObjectMeta{Name: "only-name"},
	}, "", "",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace")
}

func TestLoadPresetFile(t *testing.T) {
	t.Parallel()

	path := writeTempFile(t, "preset.yaml", minimalPresetYAML("mongodb-production", "everest"))
	preset, err := loadPresetFile(path)
	require.NoError(t, err)
	require.NotNil(t, preset.Metadata)
	assert.Equal(t, "mongodb-production", preset.Metadata.Name)
	assert.Equal(t, "percona-server-mongodb", preset.Spec.ProviderRef.Name)
}
