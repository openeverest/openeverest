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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/contract/gate"
)

// gateTestCRD returns the Provider CRD as core 2.1.0 publishes it, with the
// window [2.0.0, 2.1.0].
func gateTestCRD() *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{
		Name:   providerCRD,
		Labels: map[string]string{contract.CoreVersionKey: "2.1.0"},
		Annotations: map[string]string{
			contract.MinVersionAnnotation: "2.0.0",
			contract.VersionAnnotation:    "2.1.0",
		},
	}}
}

func gateTestProvider(observed string) *v1alpha1.Provider {
	p := testProviderCR()
	p.UID = "test-provider-uid"
	p.Status.ContractVersion = observed
	p.Status.Conditions = []metav1.Condition{{
		Type: "ContractCompatible", Status: metav1.ConditionTrue, Reason: "Supported", LastTransitionTime: metav1.Now(),
	}}
	return p
}

// newGateTestReconciler returns a reconciler of the provider "test-provider"
// over a fake cluster holding objs, whose build links linked.
func newGateTestReconciler(t *testing.T, linked string, objs ...client.Object) (*ProviderReconciler, *events.FakeRecorder) {
	t.Helper()
	s := newTestScheme()
	require.NoError(t, apiextensionsv1.AddToScheme(s))
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objs...).
		WithStatusSubresource(&v1alpha1.Provider{}).
		Build()
	recorder := events.NewFakeRecorder(10)
	return &ProviderReconciler{
		provider:    &readyTestProvider{},
		Client:      c,
		apiReader:   c,
		recorder:    recorder,
		startupGate: startupGateOptions{interval: time.Millisecond, linked: linked},
	}, recorder
}

func TestStartupGateReads(t *testing.T) {
	t.Parallel()

	acknowledged := gateTestProvider("2.0.0")
	acknowledged.Annotations = map[string]string{contract.AcknowledgedAnnotation: "2.1.0"}
	listed := gateTestCRD()
	listed.Annotations[contract.AcknowledgedExtensionsAnnotation] = "test-provider@2.1.0"

	tests := []struct {
		name string
		objs []client.Object
		root gate.Root
		core gate.Core
	}{
		{
			name: "no Provider and no core",
		},
		{
			name: "a Provider and a core",
			objs: []client.Object{acknowledged, listed},
			root: gate.Root{Observed: "2.0.0", Acknowledged: "2.1.0"},
			core: gate.Core{
				Window:                 contract.Window{Min: "2.0.0", Current: "2.1.0"},
				HasWindow:              true,
				Version:                "2.1.0",
				AcknowledgedExtensions: "test-provider@2.1.0",
			},
		},
		{
			name: "a core that predates published windows",
			objs: []client.Object{&apiextensionsv1.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: providerCRD}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, _ := newGateTestReconciler(t, "2.1.0", tt.objs...)
			runner := r.startupGateRunner(func(*v1alpha1.Provider) {})
			root, err := runner.ReadRoot(t.Context())
			require.NoError(t, err)
			assert.Equal(t, tt.root, root)
			core, err := runner.ReadCore(t.Context())
			require.NoError(t, err)
			assert.Equal(t, tt.core, core)
		})
	}
}

func TestApplyContractVersion(t *testing.T) {
	t.Parallel()

	r, _ := newGateTestReconciler(t, "2.1.0", gateTestProvider("2.0.0"))
	require.NoError(t, r.applyContractVersion(t.Context(), "2.1.0"))

	got := &v1alpha1.Provider{}
	require.NoError(t, r.Get(t.Context(), client.ObjectKey{Name: "test-provider"}, got))
	assert.Equal(t, "2.1.0", got.Status.ContractVersion)
	assert.Len(t, got.Status.Conditions, 1, "the apply leaves the conditions other writers own")
	assert.Equal(t, testProviderCR().Spec.Versions, got.Spec.Versions)
}

// awaitInBackground runs awaitCompatibleCore and returns the channel its
// result arrives on.
func awaitInBackground(ctx context.Context, r *ProviderReconciler) <-chan bool {
	done := make(chan bool, 1)
	go func() { done <- r.awaitCompatibleCore(ctx) }()
	return done
}

func observedContractVersion(t *testing.T, r *ProviderReconciler) string {
	t.Helper()
	p := &v1alpha1.Provider{}
	require.NoError(t, r.Get(t.Context(), client.ObjectKey{Name: "test-provider"}, p))
	return p.Status.ContractVersion
}

func TestAwaitCompatibleCoreHoldsANewBuildUntilCoreSupportsIt(t *testing.T) {
	t.Parallel()

	r, recorder := newGateTestReconciler(t, "2.2.0", gateTestProvider("2.1.0"), gateTestCRD())
	done := awaitInBackground(t.Context(), r)

	select {
	case e := <-recorder.Events:
		assert.Equal(t, "Warning CoreTooOld a new build (for OpenEverest 2.2.0) is waiting until OpenEverest 2.2.0 is installed", e)
	case <-time.After(10 * time.Second):
		require.FailNow(t, "the held build recorded no Event")
	}
	require.Never(t, func() bool { return len(done) > 0 }, 100*time.Millisecond, 10*time.Millisecond, "a held build started")
	assert.Equal(t, "2.1.0", observedContractVersion(t, r), "a held build records nothing")

	crd := &apiextensionsv1.CustomResourceDefinition{}
	require.NoError(t, r.Get(t.Context(), client.ObjectKey{Name: providerCRD}, crd))
	crd.Annotations[contract.VersionAnnotation] = "2.2.0"
	crd.Labels[contract.CoreVersionKey] = "2.2.0"
	require.NoError(t, r.Update(t.Context(), crd))

	select {
	case started := <-done:
		require.True(t, started)
	case <-time.After(10 * time.Second):
		require.FailNow(t, "the build did not start once core supported it")
	}
	assert.Equal(t, "Normal Supported a new build (for OpenEverest 2.2.0) starts: it works with this version of OpenEverest", <-recorder.Events)
	require.Eventually(t, func() bool { return observedContractVersion(t, r) == "2.2.0" }, 10*time.Second, 10*time.Millisecond)
}

func TestAwaitCompatibleCoreStartsTheBuildThatLastStarted(t *testing.T) {
	t.Parallel()

	// No CRD: the core was rolled back to one that publishes no window.
	r, recorder := newGateTestReconciler(t, "2.1.0", gateTestProvider("2.1.0"))
	require.True(t, r.awaitCompatibleCore(t.Context()))
	assert.Empty(t, recorder.Events)
	assert.Equal(t, "2.1.0", observedContractVersion(t, r))
}

func TestAwaitCompatibleCoreStartsAFirstSupportedBuildQuietly(t *testing.T) {
	t.Parallel()

	r, recorder := newGateTestReconciler(t, "2.1.0", gateTestProvider(""), gateTestCRD())
	require.True(t, r.awaitCompatibleCore(t.Context()))
	assert.Empty(t, recorder.Events)
	require.Eventually(t, func() bool { return observedContractVersion(t, r) == "2.1.0" }, 10*time.Second, 10*time.Millisecond)
}

func TestAwaitCompatibleCoreStartsAnAcknowledgedBuildVisibly(t *testing.T) {
	t.Parallel()

	p := gateTestProvider("2.1.0")
	p.Annotations = map[string]string{contract.AcknowledgedAnnotation: "2.1.0"}
	r, recorder := newGateTestReconciler(t, "2.2.0", p, gateTestCRD())
	require.True(t, r.awaitCompatibleCore(t.Context()))
	require.Len(t, recorder.Events, 1)
	assert.Equal(t,
		"Warning CoreTooOldAcknowledged a new build (for OpenEverest 2.2.0) starts: it needs OpenEverest 2.2.0 or later; allowed anyway because the incompatibility was acknowledged",
		<-recorder.Events,
	)
	require.Eventually(t, func() bool { return observedContractVersion(t, r) == "2.2.0" }, 10*time.Second, 10*time.Millisecond)
}

func TestAwaitCompatibleCoreStopsWithTheContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	r, recorder := newGateTestReconciler(t, "2.2.0", gateTestProvider("2.1.0"))
	done := awaitInBackground(ctx, r)
	<-recorder.Events
	cancel()
	select {
	case started := <-done:
		require.False(t, started)
	case <-time.After(10 * time.Second):
		require.FailNow(t, "the gate did not stop with its context")
	}
}

func TestAwaitCompatibleCoreCanBeSkippedInTests(t *testing.T) {
	t.Parallel()

	r, recorder := newGateTestReconciler(t, "2.2.0")
	r.startupGate.skip = true
	require.True(t, r.awaitCompatibleCore(t.Context()))
	assert.Empty(t, recorder.Events)
}
