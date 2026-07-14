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
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
)

func TestHelm_RenderTemplates(t *testing.T) {
	t.Parallel()

	testNs := "test-ns"

	// initialise a mock configuration.
	cfg := action.Configuration{}
	err := cfg.Init(nil, testNs, "memory", nil)
	require.NoError(t, err)
	cfg.KubeClient = &kubefake.PrintingKubeClient{Out: io.Discard}
	cfg.Capabilities = chartutil.DefaultCapabilities
	cfg.Log = func(_ string, _ ...any) {}

	installer := Installer{
		ReleaseName:      "test-release",
		ReleaseNamespace: testNs,
	}
	err = installer.Init("", ChartOptions{
		Directory: "../../../data/testchart",
		Version:   "0.1.0",
	})
	require.NoError(t, err)
	installer.cfg = &cfg

	ctx := context.Background()

	rendered, err := installer.RenderTemplates(ctx)
	require.NoError(t, err)

	allFiles := rendered.Strings()
	require.NoError(t, err)
	assert.Len(t, allFiles, 7)

	allFiles, err = rendered.GetUninstallManifests()
	require.NoError(t, err)
	assert.Len(t, allFiles, 7)

	crds, err := rendered.GetCRDs()
	require.NoError(t, err)
	assert.Len(t, crds, 2)
}
