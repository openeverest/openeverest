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

package helm

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func newTestActionCfg(t *testing.T, ns string) *action.Configuration {
	t.Helper()
	cfg := &action.Configuration{}
	require.NoError(t, cfg.Init(nil, ns, "memory", nil))
	cfg.KubeClient = &kubefake.PrintingKubeClient{Out: io.Discard}
	return cfg
}

func storeRelease(t *testing.T, cfg *action.Configuration, rel *release.Release) {
	t.Helper()
	mem := driver.NewMemory()
	mem.SetNamespace(rel.Namespace)
	cfg.Releases = storage.Init(mem)
	require.NoError(t, cfg.Releases.Create(rel))
}

func TestDiscoverOpenEverestNamespace(t *testing.T) {
	t.Parallel()

	t.Run("finds everest release", func(t *testing.T) {
		t.Parallel()
		cfg := newTestActionCfg(t, "custom-everest")
		storeRelease(t, cfg, &release.Release{
			Name:      "custom-everest",
			Namespace: "custom-everest",
			Version:   1,
			Info:      &release.Info{Status: release.StatusDeployed},
			Chart:     &chart.Chart{Metadata: &chart.Metadata{Name: EverestChartName}},
		})

		ns, err := discoverOpenEverestNamespace(cfg)
		require.NoError(t, err)
		assert.Equal(t, "custom-everest", ns)
	})

	t.Run("no matching release", func(t *testing.T) {
		t.Parallel()
		cfg := newTestActionCfg(t, "default")
		storeRelease(t, cfg, &release.Release{
			Name:      "other-app",
			Namespace: "default",
			Version:   1,
			Info:      &release.Info{Status: release.StatusDeployed},
			Chart:     &chart.Chart{Metadata: &chart.Metadata{Name: "other-chart"}},
		})

		_, err := discoverOpenEverestNamespace(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no OpenEverest Helm release found")
	})

	t.Run("empty release store", func(t *testing.T) {
		t.Parallel()
		cfg := newTestActionCfg(t, "")

		_, err := discoverOpenEverestNamespace(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no OpenEverest Helm release found")
	})
}
