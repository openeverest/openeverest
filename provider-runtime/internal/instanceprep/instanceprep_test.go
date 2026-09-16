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

package instanceprep_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/internal/instanceprep"
)

func bundleSpec() *v1alpha1.ProviderSpec {
	return &v1alpha1.ProviderSpec{
		Versions: []v1alpha1.VersionBundle{
			{
				Name:       "8.0",
				Default:    true,
				Components: map[string]string{"engine": "8.0.12", "proxy": "8.0.12"},
			},
			{
				Name:       "7.0",
				Components: map[string]string{"engine": "7.0.18", "proxy": "7.0.18"},
			},
		},
	}
}

func instanceWith(components map[string]v1alpha1.ComponentSpec) *v1alpha1.Instance {
	return &v1alpha1.Instance{Spec: v1alpha1.InstanceSpec{Components: components}}
}

func TestPrepareForSyncFillsOnlyUnpinnedComponents(t *testing.T) {
	t.Parallel()

	in := instanceWith(map[string]v1alpha1.ComponentSpec{
		"engine": {},
		"proxy":  {Version: "6.0.19"},
	})
	in.Spec.Version = "7.0"

	prepared, bundle, err := instanceprep.PrepareForSync(bundleSpec(), in)
	require.NoError(t, err)

	assert.Equal(t, "7.0", bundle)
	assert.Equal(t, "7.0.18", prepared.Spec.Components["engine"].Version)
	assert.Equal(t, "6.0.19", prepared.Spec.Components["proxy"].Version, "an explicit choice wins over the bundle")
}

func TestPrepareForSyncIgnoresComponentsNotInTheInstance(t *testing.T) {
	t.Parallel()

	prepared, _, err := instanceprep.PrepareForSync(bundleSpec(), instanceWith(map[string]v1alpha1.ComponentSpec{
		"engine": {},
	}))
	require.NoError(t, err)

	assert.Len(t, prepared.Spec.Components, 1)
}

func TestPrepareForSyncLeavesTheInstanceAloneWithoutABundle(t *testing.T) {
	t.Parallel()

	in := instanceWith(map[string]v1alpha1.ComponentSpec{"engine": {}})
	prepared, bundle, err := instanceprep.PrepareForSync(&v1alpha1.ProviderSpec{}, in)
	require.NoError(t, err)

	assert.Empty(t, bundle)
	assert.Same(t, in, prepared)
}

func TestPrepareForSyncRejectsAnUnknownBundle(t *testing.T) {
	t.Parallel()

	in := instanceWith(nil)
	in.Spec.Version = "9.9"

	_, _, err := instanceprep.PrepareForSync(bundleSpec(), in)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `version bundle "9.9" not found`)
}

// The caller's Instance must not be mutated: the runtime resolves a bundle for
// the provider's benefit without writing it back to the stored object.
func TestPrepareForSyncDoesNotMutateTheInput(t *testing.T) {
	t.Parallel()

	in := instanceWith(map[string]v1alpha1.ComponentSpec{"engine": {}})
	_, _, err := instanceprep.PrepareForSync(bundleSpec(), in)
	require.NoError(t, err)

	assert.Empty(t, in.Spec.Components["engine"].Version)
}
