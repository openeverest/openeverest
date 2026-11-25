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
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// The tests in this file run against a real API server, as server-side
// apply, CRD defaulting and managed fields are its behaviour. They are
// skipped unless KUBEBUILDER_ASSETS is set, as make test does.
type testAPIServer struct {
	once sync.Once
	env  *envtest.Environment
	cfg  *rest.Config
	err  error
}

// sharedAPIServer is started by the first test that needs it.
var sharedAPIServer testAPIServer //nolint:gochecknoglobals // one API server per test binary

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedAPIServer.env != nil {
		_ = sharedAPIServer.env.Stop()
	}
	os.Exit(code)
}

// apiServerClient returns a client of the shared test API server, which
// serves the CRDs of config/crd/bases, and a namespace of its own.
func apiServerClient(t *testing.T) (client.Client, string) { //nolint:ireturn // the client interface is what tests use
	t.Helper()
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("KUBEBUILDER_ASSETS is not set: run make test")
	}
	s := &sharedAPIServer
	s.once.Do(func() {
		s.env = &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
			ErrorIfCRDPathMissing: true,
		}
		// Without snapshots of the watch cache, a list at an exact resource
		// version is served from etcd: no other list is, as the cache also
		// serves the consistent ones.
		s.env.ControlPlane.GetAPIServer().Configure().Append("feature-gates", "ListFromCacheSnapshot=false")
		s.cfg, s.err = s.env.Start()
	})
	require.NoError(t, s.err)
	scheme := newTestScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	c, err := client.New(s.cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "test-"}}
	require.NoError(t, c.Create(t.Context(), ns))
	return c, ns.Name
}

// apiShippedManifest is a class as provider-sdk generates it: the API
// server defaults its tls.insecureSkipVerify and re-encodes its schemas.
const apiShippedManifest = `
apiVersion: backup.openeverest.io/v1alpha1
kind: BackupClass
metadata:
  name: envtest-pbm
  annotations:
    openeverest.io/provider-version: 1.2.0
spec:
  displayName: Percona Backup for MongoDB
  executionMode: ProviderManaged
  providerManaged:
    providerRef: {name: envtest-provider}
    supportsPITR: true
    limits: {maxPITREnabledStorages: 1}
  backupStorageClasses:
  - name: s3
    authModes: [AccessKey]
    parametersSchema:
      openAPIV3Schema:
        type: object
        additionalProperties: false
        properties:
          region: {type: string}
          implementation: {type: string, enum: [aws, minio]}
    tls: {customCA: Supported}
  parametersSchema:
    openAPIV3Schema:
      type: object
      properties:
        compression: {type: string, enum: [s2, gzip], default: s2}
  uiSchema: {backup: {}}
`

// TestAPIServer_ClaimRoundTrip claims a shipped class the API server stored,
// defaults included, then the same class edited.
func TestAPIServer_ClaimRoundTrip(t *testing.T) {
	t.Parallel()
	c, _ := apiServerClient(t)

	var shipped backupv1alpha1.BackupClass
	require.NoError(t, yaml.Unmarshal([]byte(apiShippedManifest), &shipped))
	require.NoError(t, c.Create(t.Context(), shipped.DeepCopy()))
	t.Cleanup(func() { _ = c.Delete(t.Context(), shipped.DeepCopy()) })

	r := &backupClassClaimReconciler{client: c, provider: "envtest-provider", shipped: []backupv1alpha1.BackupClass{shipped}}
	claim := func() (*backupv1alpha1.BackupClass, *metav1.Condition) {
		t.Helper()
		_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&shipped)})
		require.NoError(t, err)
		got := &backupv1alpha1.BackupClass{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(&shipped), got))
		cond := meta.FindStatusCondition(got.Status.Conditions, backupv1alpha1.BackupClassConditionAccepted)
		require.NotNil(t, cond)
		return got, cond
	}

	got, cond := claim()
	require.NotNil(t, got.Spec.BackupStorageClasses[0].TLS)
	assert.Equal(t, backupv1alpha1.TLSOptionNone, got.Spec.BackupStorageClasses[0].TLS.InsecureSkipVerify, "the API server defaulted it")
	assert.Equal(t, metav1.ConditionTrue, cond.Status, cond.Message)
	assert.Equal(t, backupv1alpha1.ReasonAccepted, cond.Reason)
	assert.Equal(t, got.Generation, cond.ObservedGeneration)
	assert.Contains(t, statusOwners(got), contract.ProviderStatusFieldOwner("envtest-provider")+"/Apply")

	got.Spec.DisplayName = "Nightly"
	require.NoError(t, c.Update(t.Context(), got))
	got, cond = claim()
	assert.Equal(t, metav1.ConditionTrue, cond.Status, "relabelling is no drift")
	assert.Equal(t, got.Generation, cond.ObservedGeneration)

	got.Spec.SupportsImport = true
	require.NoError(t, c.Update(t.Context(), got))
	got, cond = claim()
	assert.Equal(t, metav1.ConditionFalse, cond.Status)
	assert.Equal(t, backupv1alpha1.ReasonSpecDrift, cond.Reason)
	assert.Contains(t, cond.Message, "spec.supportsImport")
	assert.Equal(t, got.Generation, cond.ObservedGeneration)
}

// TestAPIServer_InstanceStatus checks, on a real API server, that the
// runtime's status apply leaves core's fields alone and takes over the
// fields an earlier runtime wrote with Update, so it removes them once it
// stops setting them.
func TestAPIServer_InstanceStatus(t *testing.T) {
	t.Parallel()
	c, ns := apiServerClient(t)

	provider := testProviderCR()
	provider.Name = "envtest-status-provider"
	require.NoError(t, c.Create(t.Context(), provider))
	t.Cleanup(func() { _ = c.Delete(t.Context(), provider) })
	p := &namedTestProvider{readyTestProvider: readyTestProvider{status: controller.Ready()}, name: provider.Name}

	in := readyTestInstance(func(in *v1alpha1.Instance) {
		in.Namespace = ns
		in.Generation = 0
		in.Spec.ProviderRef.Name = provider.Name
		in.Spec.Version = "7.0"
		in.Spec.Components = map[string]v1alpha1.ComponentSpec{"engine": {Replicas: new(int32(3))}}
		in.Labels[contract.ProviderLabel] = provider.Name
	})
	require.NoError(t, c.Create(t.Context(), in))
	key := client.ObjectKeyFromObject(in)

	// The status an earlier runtime wrote, with Update under the client's
	// default field manager, including a condition this runtime no longer
	// writes.
	in.Status.Phase = v1alpha1.InstancePhasePending
	in.Status.Conditions = []metav1.Condition{{
		Type: v1alpha1.ConditionReady, Status: metav1.ConditionFalse, Reason: "Provisioning",
		Message: "creating pods", LastTransitionTime: metav1.Now(),
	}, {
		Type: retiredCondition, Status: metav1.ConditionTrue, Reason: "Provisioning",
		Message: "creating pods", LastTransitionTime: metav1.Now(),
	}}
	require.NoError(t, c.Status().Update(t.Context(), in, client.FieldOwner("legacy-provider")))
	applyCoreCondition(t, c, in, coreCondition())

	r := &ProviderReconciler{provider: p, Client: c, legacyFieldManager: "legacy-provider"}
	_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
	require.NoError(t, err)

	got := &v1alpha1.Instance{}
	require.NoError(t, c.Get(t.Context(), key, got))
	assert.Equal(t, v1alpha1.InstancePhaseProvisioned, got.Status.Phase)
	assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, retiredCondition),
		"the earlier runtime's condition is gone with its owner",
	)
	core := meta.FindStatusCondition(got.Status.Conditions, coreCondition().Type)
	require.NotNil(t, core, "core's condition survives")
	assert.Equal(t, "set by core", core.Message)
	wantOwners := []string{coreFieldOwner + "/Apply", contract.ProviderStatusFieldOwner(provider.Name) + "/Apply"}
	assert.Equal(t, wantOwners, statusOwners(got))

	// A write from a stale read loses, as Update did.
	stale := got.DeepCopy()
	changed := coreCondition()
	changed.Message = "changed by core"
	applyCoreCondition(t, c, got, changed)
	require.NoError(t, c.Get(t.Context(), key, got))
	require.NotEqual(t, stale.ResourceVersion, got.ResourceVersion)
	stale.Status.Version = "stale"
	require.Error(t, r.applyStatus(t.Context(), stale))

	applyCoreCondition(t, c, got)
	require.NoError(t, c.Get(t.Context(), key, got))
	assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, coreCondition().Type), "core removes its own condition")
	assert.NotNil(t, meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionReady))
}

// TestAPIServer_InstanceStatusDefaults checks that an Instance no provider
// has reconciled reads as such: the CRD defaults its status on every read,
// field selectors match the defaulted phase whether etcd or the watch cache
// serves the list, and the defaults stay once another writer applies a
// status of its own, as core does.
func TestAPIServer_InstanceStatusDefaults(t *testing.T) {
	t.Parallel()
	c, ns := apiServerClient(t)

	in := readyTestInstance(func(in *v1alpha1.Instance) {
		in.Namespace = ns
		in.Generation = 0
		in.Finalizers = nil
		in.Spec.ProviderRef.Name = "envtest-defaults-provider"
		in.Spec.Version = "7.0"
		in.Spec.Components = map[string]v1alpha1.ComponentSpec{"engine": {Replicas: new(int32(3))}}
	})
	ctx := t.Context()
	require.NoError(t, c.Create(ctx, in))
	key := client.ObjectKeyFromObject(in)

	pending := client.MatchingFields{"status.phase": string(v1alpha1.InstancePhasePending)}
	// listed returns the name and resource version of each Instance in the
	// namespace whose phase is Pending.
	listed := func(t require.TestingT, raw *metav1.ListOptions) []string {
		list := &v1alpha1.InstanceList{}
		require.NoError(t, c.List(ctx, list, client.InNamespace(ns), pending, &client.ListOptions{Raw: raw}))
		found := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			found = append(found, item.Name+"@"+item.ResourceVersion)
		}
		return found
	}
	requireNeverReconciled := func(t *testing.T) *v1alpha1.Instance {
		t.Helper()
		got := &v1alpha1.Instance{}
		require.NoError(t, c.Get(ctx, key, got))
		assert.Equal(t, int64(-1), got.Status.ObservedGeneration)
		assert.Equal(t, v1alpha1.InstancePhasePending, got.Status.Phase)

		want := []string{got.Name + "@" + got.ResourceVersion}
		assert.Equal(t, want, listed(t, &metav1.ListOptions{
			ResourceVersion: got.ResourceVersion, ResourceVersionMatch: metav1.ResourceVersionMatchExact,
		}), "from etcd",
		)
		assert.Equal(t, want, listed(t, &metav1.ListOptions{}), "the latest")
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			assert.Equal(t, want, listed(t, &metav1.ListOptions{ResourceVersion: "0"}))
		}, 10*time.Second, 100*time.Millisecond, "from the watch cache",
		)
		return got
	}

	created := requireNeverReconciled(t)

	applyCoreCondition(t, c, in, metav1.Condition{
		Type: v1alpha1.ConditionUnattended, Status: metav1.ConditionUnknown, Reason: v1alpha1.ReasonNotResponding,
		Message: "The provider has not reported on generation 1", ObservedGeneration: 1, LastTransitionTime: metav1.Now(),
	})
	got := requireNeverReconciled(t)
	require.NotEqual(t, created.ResourceVersion, got.ResourceVersion, "core's status was written")
	require.Len(t, got.Status.Conditions, 1)
	assert.Equal(t, v1alpha1.ConditionUnattended, got.Status.Conditions[0].Type)
}

// retiredCondition is a condition type an earlier runtime wrote and this one
// does not.
const retiredCondition = "Initializing"

// namedTestProvider is readyTestProvider under another name.
type namedTestProvider struct {
	readyTestProvider

	name string
}

func (p *namedTestProvider) Name() string { return p.name }
