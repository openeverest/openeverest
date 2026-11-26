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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/provider-runtime/server"
)

// TestAPIServer_StartupGate starts a provider whose build the installed
// core does not support. It stays alive and unready, tells why on its
// Provider, and starts by itself, recording the version it links, once
// core is upgraded. It does not run in parallel: it points the manager at
// the test API server through KUBECONFIG, and edits the window on the
// Provider CRD, which it restores before the parallel tests resume.
func TestAPIServer_StartupGate(t *testing.T) {
	c, _ := apiServerClient(t)
	user, err := sharedAPIServer.env.ControlPlane.AddUser(envtest.User{Name: "startup-gate", Groups: []string{"system:masters"}}, nil)
	require.NoError(t, err)
	kubeconfig, err := user.KubeConfig()
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(path, kubeconfig, 0o600))
	t.Setenv("KUBECONFIG", path)

	provider := testProviderCR()
	provider.Name = "envtest-gate-provider"
	require.NoError(t, c.Create(t.Context(), provider))
	t.Cleanup(func() { _ = c.Delete(context.WithoutCancel(t.Context()), provider) })

	// A build for a contract no core serves yet is held.
	const linked = "999.0.0"
	port := freePort(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	r, err := New(ctx, &namedTestProvider{name: provider.Name},
		WithServer(server.ServerConfig{Port: port}),
		WithMetrics("0"),
		withStartupGate(startupGateOptions{interval: 50 * time.Millisecond, linked: linked}),
	)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- r.Start(ctx) }()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	alive := func() bool { return statusCode(base+"/healthz") == http.StatusOK }
	ready := func() bool { return statusCode(base+"/readyz") == http.StatusOK }
	held := func() bool { return gateEvent(t, c, provider.Name, contract.ReasonCoreTooOld) != nil }
	require.Eventually(t, alive, 10*time.Second, 20*time.Millisecond, "a held build is alive")
	require.Eventually(t, held, 10*time.Second, 50*time.Millisecond, "a held build tells why on its Provider")
	want := "a new build (for OpenEverest 999.0.0) is waiting until OpenEverest 999.0.0 is installed"
	assert.Equal(t, want, gateEvent(t, c, provider.Name, contract.ReasonCoreTooOld).Message)
	require.Never(t, ready, time.Second, 50*time.Millisecond, "a held build is not ready")
	assert.Empty(t, gateObserved(t, c, provider.Name), "a held build records nothing")

	// Core is upgraded to one whose window serves the build.
	setProviderCRDVersion(t, c, linked)

	recorded := func() bool { return gateObserved(t, c, provider.Name) == linked }
	require.Eventually(t, recorded, 10*time.Second, 50*time.Millisecond, "the build records the version it links once it starts")
	require.Eventually(t, ready, 30*time.Second, 50*time.Millisecond, "the build is ready once it starts")
	require.NotNil(t, gateEvent(t, c, provider.Name, contract.ReasonSupported), "the build tells it started")

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(30 * time.Second):
		require.FailNow(t, "the provider did not stop")
	}
}

// TestAPIServer_ContractVersionNeedsAProvider records the contract version
// on a Provider that does not exist yet: the apply fails rather than create
// one, so the gate tries again until the chart has created it.
func TestAPIServer_ContractVersionNeedsAProvider(t *testing.T) {
	t.Parallel()
	c, _ := apiServerClient(t)

	r := &ProviderReconciler{provider: &namedTestProvider{name: "envtest-gate-missing"}, Client: c}
	err := r.applyContractVersion(t.Context(), "2.1.0")
	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err), "%v", err)
	got := &v1alpha1.Provider{}
	assert.True(t, apierrors.IsNotFound(c.Get(t.Context(), client.ObjectKey{Name: "envtest-gate-missing"}, got)))
}

func gateObserved(t *testing.T, c client.Client, name string) string {
	t.Helper()
	p := &v1alpha1.Provider{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: name}, p))
	return p.Status.ContractVersion
}

// gateEvent returns the startup gate's Event with reason on the Provider
// named name, or nil.
func gateEvent(t *testing.T, c client.Client, name, reason string) *corev1.Event {
	t.Helper()
	list := &corev1.EventList{}
	require.NoError(t, c.List(t.Context(), list, client.InNamespace(metav1.NamespaceDefault)))
	i := slices.IndexFunc(list.Items, func(e corev1.Event) bool {
		return e.InvolvedObject.Kind == providerKind && e.InvolvedObject.Name == name && e.Reason == reason
	})
	if i < 0 {
		return nil
	}
	return &list.Items[i]
}

// setProviderCRDVersion publishes version as the top of the window on the
// Provider CRD, and restores the window when the test ends.
func setProviderCRDVersion(t *testing.T, c client.Client, version string) {
	t.Helper()
	crd := &metav1.PartialObjectMetadata{}
	crd.SetGroupVersionKind(apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition"))
	require.NoError(t, c.Get(t.Context(), client.ObjectKey{Name: providerCRD}, crd))
	was := crd.Annotations[contract.VersionAnnotation]
	set := func(ctx context.Context, from, to string) error {
		base := crd.DeepCopy()
		base.Annotations[contract.VersionAnnotation] = from
		patched := crd.DeepCopy()
		patched.Annotations[contract.VersionAnnotation] = to
		return c.Patch(ctx, patched, client.MergeFrom(base))
	}
	require.NoError(t, set(t.Context(), was, version))
	t.Cleanup(func() { require.NoError(t, set(context.WithoutCancel(t.Context()), version, was)) })
}
