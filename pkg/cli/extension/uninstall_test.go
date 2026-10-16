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

package extension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// newUninstaller builds a PluginUninstaller backed by a fake Kubernetes client
// holding the given objects.
func newUninstaller(t *testing.T, name string, objects ...ctrlclient.Object) *PluginUninstaller {
	t.Helper()

	k := kubernetes.NewEmpty(zap.NewNop().Sugar(), "everest-system").
		WithKubernetesClient(
			fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(objects...).
				Build(),
		)

	return &PluginUninstaller{
		cfg:        UninstallConfig{Name: name},
		kubeClient: k,
		l:          zap.NewNop().Sugar(),
	}
}

// testPlugin mirrors what `extension install` creates for a plugin.
func testPlugin(name string) *extensionsv1alpha1.Plugin {
	return &extensionsv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: extensionsv1alpha1.PluginSpec{
			DisplayName: name,
			Backend:     &extensionsv1alpha1.PluginBackend{ExternalURL: "http://example.svc:8080"},
		},
	}
}

// testInstalledExtension mirrors the InstalledExtension `extension install`
// creates next to the Plugin — cluster-scoped and with no owner reference.
func testInstalledExtension(name string) *extensionsv1alpha1.InstalledExtension {
	return &extensionsv1alpha1.InstalledExtension{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: extensionsv1alpha1.InstalledExtensionSpec{
			Type: extensionsv1alpha1.InstalledExtensionTypePlugin,
			Plugin: &extensionsv1alpha1.PluginInstall{
				PluginRef: common.ObjectRef{Name: name},
			},
		},
	}
}

func TestUninstall_DeletesPluginAndInstalledExtension(t *testing.T) {
	t.Parallel()

	const name = "leak-probe"
	pu := newUninstaller(t, name, testPlugin(name), testInstalledExtension(name))

	require.NoError(t, pu.Run(context.Background()))

	_, err := pu.kubeClient.GetPlugin(context.Background(), ctrlclient.ObjectKey{Name: name})
	assert.True(t, apierrors.IsNotFound(err), "Plugin should be gone, got %v", err)

	_, err = pu.kubeClient.GetInstalledExtension(context.Background(), ctrlclient.ObjectKey{Name: name})
	assert.True(t, apierrors.IsNotFound(err), "InstalledExtension should be gone, got %v", err)
}

func TestUninstall_NoInstalledExtension(t *testing.T) {
	t.Parallel()

	const name = "manual-plugin"
	pu := newUninstaller(t, name, testPlugin(name))

	require.NoError(t, pu.Run(context.Background()))

	_, err := pu.kubeClient.GetPlugin(context.Background(), ctrlclient.ObjectKey{Name: name})
	assert.True(t, apierrors.IsNotFound(err), "Plugin should be gone, got %v", err)
}

func TestUninstall_MissingPlugin(t *testing.T) {
	t.Parallel()

	const name = "absent"
	pu := newUninstaller(t, name)

	err := pu.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `plugin "absent" not found`)
}
