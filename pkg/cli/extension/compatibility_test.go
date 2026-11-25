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
	"errors"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// The cluster in these tests runs OpenEverest 2.1.1, a patch release that
// keeps the window of 2.1.0: it supports extensions built for 2.0.0 to 2.1.0.
const (
	testMin         = "2.0.0"
	testCurrent     = "2.1.0"
	testCoreVersion = "2.1.1"
)

func testPluginsCRD(labels, annotations map[string]string) *apiextv1.CustomResourceDefinition {
	return &apiextv1.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{
		Name:        pluginsCRD,
		Labels:      labels,
		Annotations: annotations,
	}}
}

// testWindowCRD is the Plugin CRD of a core serving [minimum, current] whose
// release is coreVersion, with extra annotations.
func testWindowCRD(minimum, current, coreVersion string, extra map[string]string) *apiextv1.CustomResourceDefinition {
	annotations := map[string]string{
		contract.MinVersionAnnotation: minimum,
		contract.VersionAnnotation:    current,
	}
	maps.Copy(annotations, extra)
	var labels map[string]string
	if coreVersion != "" {
		labels = map[string]string{contract.CoreVersionKey: coreVersion}
	}
	return testPluginsCRD(labels, annotations)
}

func testCRD(extra map[string]string) *apiextv1.CustomResourceDefinition {
	return testWindowCRD(testMin, testCurrent, testCoreVersion, extra)
}

// testPlugin is the Plugin hello 1.2.0, stamped with stamp unless it is
// empty, and carrying the extra annotations.
func testPlugin(stamp string, extra map[string]string) *extensionsv1alpha1.Plugin {
	annotations := make(map[string]string)
	if stamp != "" {
		annotations[contract.VersionAnnotation] = stamp
	}
	maps.Copy(annotations, extra)
	return &extensionsv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: "hello", Annotations: annotations},
		Spec: extensionsv1alpha1.PluginSpec{
			DisplayName: "Hello",
			Version:     "1.2.0",
			Backend:     &extensionsv1alpha1.PluginBackend{ExternalURL: "http://hello.everest-system:3001"},
		},
	}
}

func TestJudgeCompatibility(t *testing.T) {
	t.Parallel()

	noVersion := testPlugin("2.2.0", nil)
	noVersion.Spec.Version = ""

	tests := []struct {
		name   string
		plugin *extensionsv1alpha1.Plugin
		crd    *apiextv1.CustomResourceDefinition
		accept bool

		wantErr     string
		wantNotice  string
		wantAck     string
		notContains []string
	}{
		{
			name:   "supported",
			plugin: testPlugin("2.0.1", nil),
			crd:    testCRD(nil),
		},
		{
			name:   "supported at the bottom of the window",
			plugin: testPlugin(testMin, nil),
			crd:    testCRD(nil),
		},
		{
			name:   "supported at the top of the window",
			plugin: testPlugin(testCurrent, nil),
			crd:    testCRD(nil),
		},
		{
			name:   "supported: --accept-incompatible records nothing",
			plugin: testPlugin(testCurrent, nil),
			crd:    testCRD(nil),
			accept: true,
		},
		{
			name:    "core too old",
			plugin:  testPlugin("2.2.0", nil),
			crd:     testCRD(nil),
			wantErr: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later (this cluster runs OpenEverest 2.1.1); pass --accept-incompatible to install it anyway`,
		},
		{
			name:    "core too old: the extension declares no version of its own",
			plugin:  noVersion,
			crd:     testCRD(nil),
			wantErr: `extension "hello" needs OpenEverest 2.2.0 or later (this cluster runs OpenEverest 2.1.1); pass --accept-incompatible to install it anyway`,
		},
		{
			name:        "core too old for a build against an unreleased OpenEverest",
			plugin:      testPlugin("2.2.0-alpha.1.next", nil),
			crd:         testCRD(nil),
			wantErr:     `extension "hello" 1.2.0 needs a newer, unreleased OpenEverest build; install a version of it built for a released OpenEverest; pass --accept-incompatible to install it anyway`,
			notContains: []string{"next"},
		},
		{
			name:    "extension too old",
			plugin:  testPlugin("1.9.0", nil),
			crd:     testCRD(nil),
			wantErr: `extension "hello" 1.2.0 was built for an older OpenEverest; update it to a release built for OpenEverest 2.0.0 or later; pass --accept-incompatible to install it anyway`,
		},
		{
			name:        "extension too old for an unreleased OpenEverest build",
			plugin:      testPlugin("2.0.0", nil),
			crd:         testWindowCRD("2.1.0-alpha.1.next", "2.1.0-alpha.1.next", "", nil),
			wantErr:     `extension "hello" 1.2.0 was built for an older OpenEverest; update it to a version built for this unreleased OpenEverest build`,
			notContains: []string{"next", "accept-incompatible"},
		},
		{
			name:    "malformed stamp",
			plugin:  testPlugin("2.0", nil),
			wantErr: `extension "hello" 1.2.0 declares an invalid OpenEverest version "2.0", so its compatibility cannot be checked`,
		},
		{
			name:    "malformed stamp: --accept-incompatible does not help",
			plugin:  testPlugin("v2.1.0", nil),
			accept:  true,
			wantErr: `extension "hello" 1.2.0 declares an invalid OpenEverest version "v2.1.0", so its compatibility cannot be checked`,
		},
		{
			name:       "unstamped",
			plugin:     testPlugin("", nil),
			wantNotice: `extension "hello" 1.2.0 does not declare the OpenEverest it was built for, so its compatibility cannot be checked; installing it anyway`,
		},
		{
			name:    "no window: OpenEverest predates compatibility checks",
			plugin:  testPlugin(testCurrent, nil),
			crd:     testPluginsCRD(map[string]string{"everest.percona.com/version": "2.0.0-alpha.1"}, nil),
			wantErr: `extension "hello" 1.2.0 cannot be checked for compatibility: the OpenEverest in this cluster predates compatibility checks and does not publish the versions it supports; upgrade OpenEverest, then install the extension again`,
		},
		{
			name:    "no window: --accept-incompatible does not help",
			plugin:  testPlugin(testCurrent, nil),
			crd:     testPluginsCRD(map[string]string{contract.CoreVersionKey: testCoreVersion}, nil),
			accept:  true,
			wantErr: `extension "hello" 1.2.0 cannot be checked for compatibility: the OpenEverest in this cluster predates compatibility checks and does not publish the versions it supports; upgrade OpenEverest, then install the extension again`,
		},
		{
			name:       "acknowledged on the Plugin",
			plugin:     testPlugin("2.2.0", map[string]string{contract.AcknowledgedAnnotation: testCoreVersion}),
			crd:        testCRD(nil),
			wantNotice: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later; allowed anyway because the incompatibility was acknowledged; the acknowledgement lasts until an OpenEverest upgrade changes compatibility`,
		},
		{
			name:       "acknowledged on the Plugin on the release that set the window",
			plugin:     testPlugin("1.9.0", map[string]string{contract.AcknowledgedAnnotation: testCurrent}),
			crd:        testCRD(nil),
			wantNotice: `extension "hello" 1.2.0 was built for an older OpenEverest; update it to a release built for OpenEverest 2.0.0 or later; allowed anyway because the incompatibility was acknowledged; the acknowledgement lasts until an OpenEverest upgrade changes compatibility`,
		},
		{
			name:    "acknowledgement on the Plugin expired when the window moved",
			plugin:  testPlugin("2.2.0", map[string]string{contract.AcknowledgedAnnotation: "2.0.5"}),
			crd:     testCRD(nil),
			wantErr: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later (this cluster runs OpenEverest 2.1.1); pass --accept-incompatible to install it anyway`,
		},
		{
			name:    "acknowledgement on the Plugin for a release not installed yet",
			plugin:  testPlugin("2.2.0", map[string]string{contract.AcknowledgedAnnotation: "2.2.0"}),
			crd:     testCRD(nil),
			wantErr: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later (this cluster runs OpenEverest 2.1.1); pass --accept-incompatible to install it anyway`,
		},
		{
			name:   "acknowledged in the list core publishes",
			plugin: testPlugin("2.2.0", nil),
			crd: testCRD(map[string]string{
				contract.AcknowledgedExtensionsAnnotation: "other@2.1.0, hello@2.1.1",
			}),
			wantNotice: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later; allowed anyway because the incompatibility was acknowledged; the acknowledgement lasts until an OpenEverest upgrade changes compatibility`,
		},
		{
			name:    "the list core publishes acknowledges another extension",
			plugin:  testPlugin("2.2.0", nil),
			crd:     testCRD(map[string]string{contract.AcknowledgedExtensionsAnnotation: "other@2.1.1"}),
			wantErr: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later (this cluster runs OpenEverest 2.1.1); pass --accept-incompatible to install it anyway`,
		},
		{
			name:    "a list core publishes that does not parse acknowledges nothing",
			plugin:  testPlugin("2.2.0", nil),
			crd:     testCRD(map[string]string{contract.AcknowledgedExtensionsAnnotation: "hello@2.1.1,hello@2.1.0"}),
			wantErr: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later (this cluster runs OpenEverest 2.1.1); pass --accept-incompatible to install it anyway`,
		},
		{
			name:       "--accept-incompatible: core too old",
			plugin:     testPlugin("2.2.0", nil),
			crd:        testCRD(nil),
			accept:     true,
			wantNotice: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later; allowed anyway because the incompatibility was acknowledged; recording the acknowledgement on the Plugin as openeverest.io/compatibility-acknowledged=2.1.1, where it lasts until an OpenEverest upgrade changes compatibility`,
			wantAck:    testCoreVersion,
		},
		{
			name:       "--accept-incompatible: extension too old",
			plugin:     testPlugin("1.9.0", nil),
			crd:        testCRD(nil),
			accept:     true,
			wantNotice: `extension "hello" 1.2.0 was built for an older OpenEverest; update it to a release built for OpenEverest 2.0.0 or later; allowed anyway because the incompatibility was acknowledged; recording the acknowledgement on the Plugin as openeverest.io/compatibility-acknowledged=2.1.1, where it lasts until an OpenEverest upgrade changes compatibility`,
			wantAck:    testCoreVersion,
		},
		{
			name:       "--accept-incompatible replaces an expired acknowledgement",
			plugin:     testPlugin("2.2.0", map[string]string{contract.AcknowledgedAnnotation: "2.0.5"}),
			crd:        testCRD(nil),
			accept:     true,
			wantNotice: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later; allowed anyway because the incompatibility was acknowledged; recording the acknowledgement on the Plugin as openeverest.io/compatibility-acknowledged=2.1.1, where it lasts until an OpenEverest upgrade changes compatibility`,
			wantAck:    testCoreVersion,
		},
		{
			name:    "--accept-incompatible: OpenEverest does not publish its release",
			plugin:  testPlugin("2.2.0", nil),
			crd:     testWindowCRD(testMin, testCurrent, "", nil),
			accept:  true,
			wantErr: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later; --accept-incompatible cannot acknowledge it, because the OpenEverest in this cluster does not publish which release it is`,
		},
		{
			name:    "--accept-incompatible: unreleased OpenEverest build",
			plugin:  testPlugin("1.9.0", nil),
			crd:     testWindowCRD(testMin, "2.1.0-alpha.1.next", "2.1.0-alpha.1", nil),
			accept:  true,
			wantErr: `extension "hello" 1.2.0 was built for an older OpenEverest; update it to a release built for OpenEverest 2.0.0 or later; --accept-incompatible cannot acknowledge it, because this cluster runs an unreleased OpenEverest build`,
		},
		{
			name:   "extension too old where core keeps incompatible plugins",
			plugin: testPlugin("1.9.0", nil),
			crd: testCRD(map[string]string{
				contract.IncompatiblePluginsAnnotation: contract.IncompatiblePluginsContinue,
			}),
			wantNotice: `extension "hello" 1.2.0 was built for an older OpenEverest; update it to a release built for OpenEverest 2.0.0 or later; installing it anyway, because this OpenEverest keeps incompatible plugins installed but does not load them`,
		},
		{
			name:   "core too old where core keeps incompatible plugins",
			plugin: testPlugin("2.2.0", nil),
			crd: testCRD(map[string]string{
				contract.IncompatiblePluginsAnnotation: contract.IncompatiblePluginsContinue,
			}),
			wantErr: `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later (this cluster runs OpenEverest 2.1.1); pass --accept-incompatible to install it anyway`,
		},
		{
			name:   "extension too old where core blocks incompatible plugins",
			plugin: testPlugin("1.9.0", nil),
			crd: testCRD(map[string]string{
				contract.IncompatiblePluginsAnnotation: contract.IncompatiblePluginsBlock,
			}),
			wantErr: `extension "hello" 1.2.0 was built for an older OpenEverest; update it to a release built for OpenEverest 2.0.0 or later; pass --accept-incompatible to install it anyway`,
		},
		{
			name:    "core version skew during an upgrade",
			plugin:  testPlugin("2.1.0", nil),
			crd:     testWindowCRD("2.2.0", testCurrent, testCoreVersion, nil),
			wantErr: `extension "hello" 1.2.0 cannot be checked for compatibility while the parts of this OpenEverest disagree on the versions they support, as they do during an upgrade; try again once the upgrade has finished; pass --accept-incompatible to install it anyway`,
		},
		{
			name:    "invalid window",
			plugin:  testPlugin("2.1.0", nil),
			crd:     testWindowCRD(testMin, "2.1", testCoreVersion, nil),
			accept:  true,
			wantErr: `extension "hello" 1.2.0 cannot be checked for compatibility because this OpenEverest publishes invalid supported versions; --accept-incompatible cannot acknowledge it`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := judgeCompatibility(tt.plugin, tt.crd, tt.accept)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				for _, s := range tt.notContains {
					assert.NotContains(t, err.Error(), s)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantNotice, got.notice)
			assert.Equal(t, tt.wantAck, got.acknowledge)
			for _, s := range tt.notContains {
				assert.NotContains(t, got.notice, s)
			}
		})
	}
}

func TestPluginInstaller_install(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plugin   *extensionsv1alpha1.Plugin
		accept   bool
		objects  []ctrlclient.Object
		getErr   error
		wantErr  func(t *testing.T, err error)
		wantAck  string
		noPlugin bool
	}{
		{
			name:    "supported",
			plugin:  testPlugin(testCurrent, nil),
			objects: []ctrlclient.Object{testCRD(nil)},
		},
		{
			name:    "refused: nothing is applied",
			plugin:  testPlugin("2.2.0", nil),
			objects: []ctrlclient.Object{testCRD(nil)},
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				assert.EqualError(t, err, `extension "hello" 1.2.0 needs OpenEverest 2.2.0 or later (this cluster runs OpenEverest 2.1.1); pass --accept-incompatible to install it anyway`)
			},
			noPlugin: true,
		},
		{
			name:    "--accept-incompatible records the acknowledgement on the Plugin",
			plugin:  testPlugin("2.2.0", nil),
			accept:  true,
			objects: []ctrlclient.Object{testCRD(nil)},
			wantAck: testCoreVersion,
		},
		{
			name:   "--accept-incompatible leaves a Plugin already installed unchanged",
			plugin: testPlugin("2.2.0", nil),
			accept: true,
			objects: []ctrlclient.Object{
				testCRD(nil),
				testPlugin(testCurrent, nil),
			},
		},
		{
			name:   "unstamped: the CRD is not read",
			plugin: testPlugin("", nil),
			getErr: errors.New("the CRD must not be read"),
		},
		{
			name:   "stamped, without OpenEverest",
			plugin: testPlugin(testCurrent, nil),
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				assert.EqualError(t, err, `cannot install extension "hello": OpenEverest is not installed in this cluster (the plugins.extensions.openeverest.io CRD is missing)`)
			},
			noPlugin: true,
		},
		{
			name:   "stamped, the CRD cannot be read",
			plugin: testPlugin(testCurrent, nil),
			getErr: k8serrors.NewForbidden(apiextv1.Resource("customresourcedefinitions"), pluginsCRD, errors.New("no access")),
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				assert.True(t, k8serrors.IsForbidden(err), "unexpected error: %v", err)
			},
			noPlugin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(tt.objects...)
			if tt.getErr != nil {
				builder = builder.WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, c ctrlclient.WithWatch, key ctrlclient.ObjectKey, obj ctrlclient.Object, opts ...ctrlclient.GetOption) error {
						if _, ok := obj.(*apiextv1.CustomResourceDefinition); ok {
							return tt.getErr
						}
						return c.Get(ctx, key, obj, opts...)
					},
				})
			}
			c := builder.Build()
			pi := &PluginInstaller{
				cfg:        InstallConfig{AcceptIncompatible: tt.accept},
				kubeClient: kubernetes.NewEmpty(zap.NewNop().Sugar(), "everest-system").WithKubernetesClient(c),
				l:          zap.NewNop().Sugar(),
			}

			err := pi.install(context.Background(), tt.plugin)
			if tt.wantErr != nil {
				require.Error(t, err)
				tt.wantErr(t, err)
			} else {
				require.NoError(t, err)
			}

			plugin := &extensionsv1alpha1.Plugin{}
			ie := &extensionsv1alpha1.InstalledExtension{}
			pluginErr := c.Get(context.Background(), ctrlclient.ObjectKey{Name: "hello"}, plugin)
			ieErr := c.Get(context.Background(), ctrlclient.ObjectKey{Name: "hello"}, ie)
			if tt.noPlugin {
				assert.True(t, k8serrors.IsNotFound(pluginErr), "the Plugin was applied: %v", pluginErr)
				assert.True(t, k8serrors.IsNotFound(ieErr), "the InstalledExtension was applied: %v", ieErr)
				return
			}
			require.NoError(t, pluginErr)
			require.NoError(t, ieErr)
			assert.Equal(t, tt.wantAck, plugin.GetAnnotations()[contract.AcknowledgedAnnotation])
		})
	}
}
