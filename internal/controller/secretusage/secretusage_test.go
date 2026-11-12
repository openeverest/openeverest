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

package secretusage

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	pkgcommon "github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	testNamespace  = "team-a"
	testHold       = "test.openeverest.io/hold"
	podsCountKey   = "test.openeverest.io/attached-pods"
	destinationKey = pkgcommon.AttachedDestinationsAnnotation
)

// destinations mirrors the MonitoringDestination entry the monitoring
// package registers, which this package cannot import.
func destinations() Referrer {
	return Referrer{
		Kind:       "MonitoringDestination",
		CountKey:   destinationKey,
		Object:     &monitoringv1alpha1.MonitoringDestination{},
		NewList:    func() client.ObjectList { return &monitoringv1alpha1.MonitoringDestinationList{} },
		IndexField: ".spec.secretNames",
		Names: func(obj client.Object) []string {
			dst, ok := obj.(*monitoringv1alpha1.MonitoringDestination)
			if !ok {
				return nil
			}
			return dst.Spec.SecretNames()
		},
	}
}

// pods is a second referrer kind, as BackupStorage will be: Pods naming a
// Secret in a volume.
func pods() Referrer {
	return Referrer{
		Kind:       "Pod",
		CountKey:   podsCountKey,
		Object:     &corev1.Pod{},
		NewList:    func() client.ObjectList { return &corev1.PodList{} },
		IndexField: ".spec.volumes.secret.secretName",
		Names: func(obj client.Object) []string {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				return nil
			}
			var names []string
			for _, v := range pod.Spec.Volumes {
				if v.Secret != nil {
					names = append(names, v.Secret.SecretName)
				}
			}
			return names
		},
	}
}

func newClient(t *testing.T, funcs interceptor.Funcs, objs ...client.Object) client.WithWatch { //nolint:ireturn
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, monitoringv1alpha1.AddToScheme(scheme))
	seeded := make([]client.Object, 0, len(objs))
	for _, obj := range objs {
		copied, ok := obj.DeepCopyObject().(client.Object)
		require.True(t, ok)
		seeded = append(seeded, copied)
	}
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(seeded...).
		WithIndex(&monitoringv1alpha1.MonitoringDestination{}, destinations().IndexField, destinations().Names).
		WithIndex(&corev1.Pod{}, pods().IndexField, pods().Names).
		WithInterceptorFuncs(funcs).
		Build()
}

// world is a Reconciler over a cache and an API server that may disagree,
// counting the patches it sends.
type world struct {
	r       *Reconciler
	cache   client.Client
	rec     *events.FakeRecorder
	patches *atomic.Int32
}

// newWorld seeds the cache with secret and cached, and the API server with
// secret and live; live nil means the API server agrees with the cache.
func newWorld(t *testing.T, secret *corev1.Secret, cached, live []client.Object, refs ...Referrer) world {
	t.Helper()
	patches := &atomic.Int32{}
	countPatches := interceptor.Funcs{
		Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			patches.Add(1)
			return c.Patch(ctx, obj, patch, opts...)
		},
	}
	cache := newClient(t, countPatches, append([]client.Object{secret}, cached...)...)
	apiServer := client.Reader(cache)
	if live != nil {
		apiServer = newClient(t, interceptor.Funcs{}, append([]client.Object{secret}, live...)...)
	}
	if len(refs) == 0 {
		refs = []Referrer{destinations()}
	}
	rec := events.NewFakeRecorder(10)
	return world{
		r:       &Reconciler{Client: cache, APIReader: apiServer, Recorder: rec, Referrers: refs},
		cache:   cache,
		rec:     rec,
		patches: patches,
	}
}

func (w world) reconcile(t *testing.T, name string) {
	t.Helper()
	_, err := w.r.Reconcile(t.Context(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: name}})
	require.NoError(t, err)
}

// stored returns the Secret as the cache holds it after the reconcile, or
// nil once it is gone.
func (w world) stored(t *testing.T, name string) *corev1.Secret {
	t.Helper()
	secret := &corev1.Secret{}
	err := w.cache.Get(t.Context(), types.NamespacedName{Namespace: testNamespace, Name: name}, secret)
	if apierrors.IsNotFound(err) {
		return nil
	}
	require.NoError(t, err)
	return secret
}

func newSecret(name string, labels map[string]string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace, Labels: labels},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"token": []byte("glsa_token")},
	}
}

// credentials is a tracked Secret: the credentials of a monitoring class
// mode.
func credentials(name string) *corev1.Secret {
	return newSecret(name, map[string]string{
		pkgcommon.OpenEverestMonitoringClassLabel: "pmm",
		pkgcommon.OpenEverestDefinitionLabel:      "ServiceAccountToken",
	})
}

// caBundle is a tracked Secret of the core definition ca-bundle.
func caBundle(name string) *corev1.Secret {
	return newSecret(name, map[string]string{pkgcommon.OpenEverestDefinitionLabel: pkgcommon.OpenEverestCABundleDefinition})
}

// marked gives secret core's marks: count annotations and, unless no
// finalizer is given, the finalizers.
func marked(secret *corev1.Secret, counts map[string]string, finalizers ...string) *corev1.Secret {
	secret.Annotations = counts
	secret.Finalizers = finalizers
	return secret
}

func deleted[T client.Object](obj T) T {
	if len(obj.GetFinalizers()) == 0 {
		obj.SetFinalizers([]string{testHold})
	}
	obj.SetDeletionTimestamp(&metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	return obj
}

func inNamespace[T client.Object](obj T, namespace string) T {
	obj.SetNamespace(namespace)
	return obj
}

// destination references creds and, unless empty, ca.
func destination(name, creds, ca string) *monitoringv1alpha1.MonitoringDestination {
	dst := &monitoringv1alpha1.MonitoringDestination{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Spec: monitoringv1alpha1.MonitoringDestinationSpec{
			ClassRef:       common.ObjectRef{Name: "pmm"},
			Authentication: common.Authentication{Mode: "ServiceAccountToken"},
		},
	}
	if creds != "" {
		dst.Spec.Authentication.CredentialsSecretRef = &common.SecretRef{Name: creds}
	}
	if ca != "" {
		dst.Spec.TLS = &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: ca, Key: "ca.crt"}}}
	}
	return dst
}

func pod(name, secret string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
			Name:         "creds",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: secret}},
		}}},
	}
}

func counts(destinations string) map[string]string {
	return map[string]string{destinationKey: destinations}
}

func TestReconcile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		secret *corev1.Secret
		cached []client.Object
		// live is what the API server has; nil when it agrees with the cache.
		live []client.Object
		refs []Referrer

		wantAnnotations map[string]string
		wantFinalizers  []string
		wantGone        bool
		wantEvent       string
		// wantUntouched: no patch at all.
		wantUntouched bool
	}{
		{
			name:            "an unused tracked Secret counts 0 and has no finalizer",
			secret:          credentials("pmm-token"),
			cached:          []client.Object{destination("other", "other-token", "")},
			wantAnnotations: counts("0"),
		},
		{
			name:            "one destination counts 1 and holds the finalizer",
			secret:          credentials("pmm-token"),
			cached:          []client.Object{destination("pmm-prod", "pmm-token", "")},
			wantAnnotations: counts("1"),
			wantFinalizers:  []string{finalizer},
		},
		{
			name:   "two destinations count 2",
			secret: credentials("shared-token"),
			cached: []client.Object{
				destination("pmm-prod", "shared-token", ""),
				destination("pmm-staging", "shared-token", ""),
			},
			wantAnnotations: counts("2"),
			wantFinalizers:  []string{finalizer},
		},
		{
			name:   "a CA bundle shared by two destinations counts both, a destination naming it twice once",
			secret: caBundle("corp-ca"),
			cached: []client.Object{
				destination("pmm-prod", "pmm-token", "corp-ca"),
				destination("grafana", "grafana-token", "corp-ca"),
				destination("odd", "corp-ca", "corp-ca"),
			},
			wantAnnotations: counts("3"),
			wantFinalizers:  []string{finalizer},
		},
		{
			name:   "a destination in another namespace does not count",
			secret: credentials("pmm-token"),
			cached: []client.Object{
				inNamespace(destination("pmm-prod", "pmm-token", ""), "team-b"),
			},
			wantAnnotations: counts("0"),
		},
		{
			name:   "a destination being deleted does not count while the Secret is live, but holds the finalizer",
			secret: credentials("pmm-token"),
			cached: []client.Object{
				deleted(destination("pmm-prod", "pmm-token", "")),
				destination("pmm-staging", "other-token", ""),
			},
			wantAnnotations: counts("0"),
			wantFinalizers:  []string{finalizer},
		},
		{
			name:   "a Secret being deleted counts destinations being deleted, read from the API server",
			secret: deleted(marked(credentials("pmm-token"), counts("0"), finalizer)),
			// The cache has not seen the destinations yet.
			live: []client.Object{
				deleted(destination("pmm-prod", "pmm-token", "")),
				destination("pmm-staging", "pmm-token", ""),
			},
			wantAnnotations: counts("2"),
			wantFinalizers:  []string{finalizer},
			wantEvent:       "Warning DeletionBlocked deletion waits for 2 MonitoringDestination(s) to stop using it: pmm-prod, pmm-staging",
		},
		{
			name:   "a Secret being deleted goes once nothing uses it, whatever the cache says",
			secret: deleted(marked(credentials("pmm-token"), counts("1"), finalizer)),
			cached: []client.Object{destination("pmm-prod", "pmm-token", "")},
			live:   []client.Object{},
			// The release and the count are one patch, which lets the
			// deletion finish.
			wantGone: true,
		},
		{
			name:   "a Secret being deleted without the finalizer gets its counts but no finalizer",
			secret: deleted(credentials("pmm-token")),
			cached: []client.Object{destination("pmm-prod", "pmm-token", "")},
			// Finalizers cannot be added to an object being deleted.
			wantAnnotations: counts("1"),
			wantFinalizers:  []string{testHold},
		},
		{
			name:   "the finalizer stays while the API server has a referrer the cache has not seen",
			secret: marked(credentials("pmm-token"), counts("0"), finalizer),
			live:   []client.Object{destination("pmm-prod", "pmm-token", "")},
			// The count follows the API server too.
			wantAnnotations: counts("1"),
			wantFinalizers:  []string{finalizer},
		},
		{
			name:            "the finalizer goes once the API server confirms nothing uses the Secret",
			secret:          marked(credentials("pmm-token"), counts("1"), finalizer),
			live:            []client.Object{destination("pmm-prod", "other-token", "")},
			wantAnnotations: counts("0"),
		},
		{
			name:   "nothing is written when nothing changed",
			secret: marked(credentials("pmm-token"), counts("1"), finalizer),
			cached: []client.Object{destination("pmm-prod", "pmm-token", "")},
			// Someone else's annotations and finalizers are kept as they are.
			wantUntouched: true,
		},
		{
			name: "someone else's annotations and finalizers are kept",
			secret: marked(credentials("pmm-token"),
				map[string]string{"example.com/note": "x", destinationKey: "5"}, "example.com/keep",
			),
			cached:          []client.Object{destination("pmm-prod", "pmm-token", "")},
			wantAnnotations: map[string]string{"example.com/note": "x", destinationKey: "1"},
			wantFinalizers:  []string{"example.com/keep", finalizer},
		},
		{
			name:          "an unlabelled Secret is never written",
			secret:        newSecret("cert-manager-ca", nil),
			cached:        []client.Object{destination("pmm-prod", "pmm-token", "cert-manager-ca")},
			wantUntouched: true,
		},
		{
			name: "a Secret a Provider defines is not tracked yet",
			secret: newSecret("orders-users", map[string]string{
				pkgcommon.OpenEverestProviderLabel:   "percona-server-mongodb",
				pkgcommon.OpenEverestDefinitionLabel: "users",
			}),
			cached:        []client.Object{destination("pmm-prod", "orders-users", "")},
			wantUntouched: true,
		},
		{
			name: "a Secret whose labels were removed while in use loses its counts and keeps the finalizer",
			secret: marked(newSecret("pmm-token", nil),
				map[string]string{destinationKey: "1", "example.com/note": "x"}, finalizer,
			),
			cached: []client.Object{destination("pmm-prod", "pmm-token", "")},
			// Not being deleted, so nothing to report.
			wantAnnotations: map[string]string{"example.com/note": "x"},
			wantFinalizers:  []string{finalizer},
		},
		{
			name:   "a Secret whose labels were removed is released once the API server confirms nothing uses it",
			secret: marked(newSecret("pmm-token", nil), counts("1"), finalizer),
			cached: []client.Object{destination("pmm-prod", "pmm-token", "")},
			live:   []client.Object{},
		},
		{
			name:   "a Secret whose labels were removed while unused is not written",
			secret: marked(newSecret("pmm-token", nil), counts("0")),
			// The left-over 0 refuses nothing; the release subcommand strips it.
			wantUntouched: true,
		},
		{
			name:           "a Secret whose labels were removed and that is being deleted reports what it waits for",
			secret:         deleted(marked(newSecret("pmm-token", nil), counts("1"), finalizer)),
			cached:         []client.Object{destination("pmm-prod", "pmm-token", "")},
			wantFinalizers: []string{finalizer},
			wantEvent:      "Warning DeletionBlocked deletion waits for 1 MonitoringDestination(s) to stop using it: pmm-prod",
		},
		{
			name:   "every kind has its count",
			secret: caBundle("exporter-ca"),
			cached: []client.Object{
				destination("pmm-prod", "pmm-token", "exporter-ca"),
				pod("exporter", "exporter-ca"),
				pod("sidecar", "exporter-ca"),
				pod("other", "other-ca"),
			},
			refs:            []Referrer{destinations(), pods()},
			wantAnnotations: map[string]string{destinationKey: "1", podsCountKey: "2"},
			wantFinalizers:  []string{finalizer},
		},
		{
			name:   "a deletion held by several kinds names each",
			secret: deleted(marked(caBundle("corp-ca"), map[string]string{destinationKey: "1", podsCountKey: "2"}, finalizer)),
			cached: []client.Object{
				destination("pmm-prod", "pmm-token", "corp-ca"),
				pod("exporter", "corp-ca"),
				pod("sidecar", "corp-ca"),
			},
			refs: []Referrer{destinations(), pods()},
			// The counts are right already: only the event.
			wantUntouched: true,
			wantEvent: "Warning DeletionBlocked deletion waits for 3 objects to stop using it: " +
				"1 MonitoringDestination: pmm-prod; 2 Pods: exporter, sidecar",
		},
		{
			name:            "the finalizer is held while only another kind uses the Secret",
			secret:          marked(caBundle("corp-ca"), map[string]string{destinationKey: "1", podsCountKey: "0"}, finalizer),
			cached:          []client.Object{pod("exporter", "corp-ca")},
			refs:            []Referrer{destinations(), pods()},
			wantAnnotations: map[string]string{destinationKey: "0", podsCountKey: "1"},
			wantFinalizers:  []string{finalizer},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := newWorld(t, tt.secret, tt.cached, tt.live, tt.refs...)
			before := w.stored(t, tt.secret.Name)

			w.reconcile(t, tt.secret.Name)

			after := w.stored(t, tt.secret.Name)
			if tt.wantGone {
				assert.Nil(t, after, "the Secret should be gone")
				return
			}
			require.NotNil(t, after)
			assert.Equal(t, before.Data, after.Data, "the data is never written")
			assert.Equal(t, before.Labels, after.Labels, "the labels are never written")
			if tt.wantUntouched {
				assert.Zero(t, w.patches.Load(), "no patch")
				assert.Equal(t, before.ResourceVersion, after.ResourceVersion)
				assert.Equal(t, before.Annotations, after.Annotations)
				assert.Equal(t, before.Finalizers, after.Finalizers)
			} else {
				assert.Equal(t, int32(1), w.patches.Load(), "one patch")
				assertEqualOrEmpty(t, tt.wantAnnotations, after.Annotations)
				assertEqualOrEmpty(t, tt.wantFinalizers, after.Finalizers)
			}
			if tt.wantEvent == "" {
				assert.Empty(t, w.rec.Events)
				return
			}
			require.Len(t, w.rec.Events, 1)
			assert.Equal(t, tt.wantEvent, <-w.rec.Events)
		})
	}
}

// assertEqualOrEmpty treats nil and empty alike.
func assertEqualOrEmpty[T []string | map[string]string](t *testing.T, want, got T) {
	t.Helper()
	if len(want) == 0 {
		assert.Empty(t, got)
		return
	}
	assert.Equal(t, want, got)
}

func TestReconcileIsIdempotent(t *testing.T) {
	t.Parallel()
	w := newWorld(t, credentials("pmm-token"), []client.Object{destination("pmm-prod", "pmm-token", "")}, nil)

	w.reconcile(t, "pmm-token")
	version := w.stored(t, "pmm-token").ResourceVersion
	w.reconcile(t, "pmm-token")

	assert.Equal(t, int32(1), w.patches.Load())
	assert.Equal(t, version, w.stored(t, "pmm-token").ResourceVersion)
}

func TestReconcileReadsASecretTheCacheDoesNotHold(t *testing.T) {
	t.Parallel()
	// A cache filtered by label drops a Secret whose labels were removed.
	secret := marked(newSecret("pmm-token", nil), counts("1"), finalizer)
	hidden := interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*corev1.Secret); ok {
				return apierrors.NewNotFound(corev1.Resource("secrets"), key.Name)
			}
			return c.Get(ctx, key, obj, opts...)
		},
	}
	cache := newClient(t, hidden, secret)
	apiServer := newClient(t, interceptor.Funcs{}, secret)
	r := &Reconciler{Client: cache, APIReader: apiServer, Recorder: events.NewFakeRecorder(1), Referrers: []Referrer{destinations()}}

	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: "pmm-token"}})
	require.NoError(t, err)

	// The cache is written through, as the API server would be.
	list := &corev1.SecretList{}
	require.NoError(t, cache.List(t.Context(), list))
	require.Len(t, list.Items, 1)
	assert.Empty(t, list.Items[0].Finalizers)
	assert.Empty(t, list.Items[0].Annotations)
}

func TestReconcileMissingSecret(t *testing.T) {
	t.Parallel()
	c := newClient(t, interceptor.Funcs{})
	r := &Reconciler{Client: c, APIReader: c, Recorder: events.NewFakeRecorder(1), Referrers: []Referrer{destinations()}}

	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: "gone"}})

	require.NoError(t, err)
}

func TestReconcileConflict(t *testing.T) {
	t.Parallel()
	// Someone wrote the Secret after the cache read it: the patch, locked on
	// the version read, must not overwrite that change.
	conflict := interceptor.Funcs{
		Patch: func(_ context.Context, _ client.WithWatch, obj client.Object, _ client.Patch, _ ...client.PatchOption) error {
			return apierrors.NewConflict(corev1.Resource("secrets"), obj.GetName(), nil)
		},
	}
	c := newClient(t, conflict, credentials("pmm-token"), destination("pmm-prod", "pmm-token", ""))
	r := &Reconciler{Client: c, APIReader: c, Recorder: events.NewFakeRecorder(1), Referrers: []Referrer{destinations()}}

	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: "pmm-token"}})

	assert.True(t, apierrors.IsConflict(err), "a conflict is returned so the Secret is requeued: %v", err)
}

func TestEnqueueSecretsOf(t *testing.T) {
	t.Parallel()
	before := destination("pmm-prod", "pmm-token", "corp-ca")
	repointed := destination("pmm-prod", "pmm-token-v2", "corp-ca")
	deleting := deleted(repointed.DeepCopy())
	released := deleting.DeepCopy()
	released.Finalizers = []string{"test.openeverest.io/other"}
	withStatus := before.DeepCopy()
	withStatus.Status.AttachedInstances = 3
	requests := func(names ...string) []reconcile.Request {
		reqs := make([]reconcile.Request, 0, len(names))
		for _, name := range names {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: name}})
		}
		return reqs
	}

	tests := []struct {
		name string
		send func(handler handler.Funcs, q workqueue.TypedRateLimitingInterface[reconcile.Request])
		want []reconcile.Request
	}{
		{
			name: "create enqueues every Secret named",
			send: func(h handler.Funcs, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Create(t.Context(), event.CreateEvent{Object: before}, q)
			},
			want: requests("pmm-token", "corp-ca"),
		},
		{
			name: "re-pointing enqueues the old and the new Secret",
			send: func(h handler.Funcs, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Update(t.Context(), event.UpdateEvent{ObjectOld: before, ObjectNew: repointed}, q)
			},
			want: requests("pmm-token", "pmm-token-v2"),
		},
		{
			name: "an update that keeps the Secrets enqueues nothing",
			send: func(h handler.Funcs, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Update(t.Context(), event.UpdateEvent{ObjectOld: before, ObjectNew: withStatus}, q)
			},
		},
		{
			name: "the start of the deletion enqueues every Secret named",
			send: func(h handler.Funcs, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Update(t.Context(), event.UpdateEvent{ObjectOld: repointed, ObjectNew: deleting}, q)
			},
			want: requests("pmm-token-v2", "corp-ca"),
		},
		{
			name: "an update of an object already being deleted that keeps the Secrets enqueues nothing",
			send: func(h handler.Funcs, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Update(t.Context(), event.UpdateEvent{ObjectOld: deleting, ObjectNew: released}, q)
			},
		},
		{
			name: "delete enqueues every Secret named",
			send: func(h handler.Funcs, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Delete(t.Context(), event.DeleteEvent{Object: repointed}, q)
			},
			want: requests("pmm-token-v2", "corp-ca"),
		},
		{
			name: "Secrets are enqueued in the object's namespace",
			send: func(h handler.Funcs, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Create(t.Context(), event.CreateEvent{Object: inNamespace(destination("pmm-prod", "pmm-token", ""), "team-b")}, q)
			},
			want: []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: "team-b", Name: "pmm-token"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			q := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
			defer q.ShutDown()

			tt.send(enqueueSecretsOf(destinations()), q)

			var got []reconcile.Request
			for q.Len() > 0 {
				req, _ := q.Get()
				got = append(got, req)
				q.Done(req)
			}
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestSecretPredicate(t *testing.T) {
	t.Parallel()
	p := secretPredicate()
	unlabelled := newSecret("cert-manager-ca", nil)
	held := marked(newSecret("pmm-token", nil), counts("1"), finalizer)
	provider := newSecret("orders-users", map[string]string{
		pkgcommon.OpenEverestProviderLabel:   "percona-server-mongodb",
		pkgcommon.OpenEverestDefinitionLabel: "users",
	})

	assert.True(t, p.Create(event.CreateEvent{Object: credentials("pmm-token")}), "tracked")
	assert.True(t, p.Create(event.CreateEvent{Object: caBundle("corp-ca")}), "tracked CA bundle")
	assert.True(t, p.Create(event.CreateEvent{Object: held}), "untracked but held")
	assert.False(t, p.Create(event.CreateEvent{Object: unlabelled}), "unlabelled")
	assert.False(t, p.Create(event.CreateEvent{Object: provider}), "defined by a Provider")
	assert.True(t, p.Update(event.UpdateEvent{ObjectOld: credentials("pmm-token"), ObjectNew: unlabelled}), "labels removed")
	assert.True(t, p.Update(event.UpdateEvent{ObjectOld: unlabelled, ObjectNew: credentials("pmm-token")}), "labels added")
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: unlabelled, ObjectNew: unlabelled.DeepCopy()}), "unlabelled throughout")
	assert.True(t, p.Delete(event.DeleteEvent{Object: credentials("pmm-token")}), "tracked")
	assert.False(t, p.Delete(event.DeleteEvent{Object: unlabelled}), "unlabelled")
}

func TestContentChanged(t *testing.T) {
	t.Parallel()
	base := credentials("pmm-token")
	change := func(mutate func(*corev1.Secret)) *corev1.Secret {
		s := base.DeepCopy()
		mutate(s)
		return s
	}

	tests := []struct {
		name string
		new  *corev1.Secret
		want bool
	}{
		{name: "counts written by core", new: change(func(s *corev1.Secret) { s.Annotations = counts("2") })},
		{name: "finalizer added by core", new: change(func(s *corev1.Secret) { s.Finalizers = []string{finalizer} })},
		{name: "resync", new: base.DeepCopy()},
		{name: "data", new: change(func(s *corev1.Secret) { s.Data["token"] = []byte("glsa_rotated") }), want: true},
		{name: "a key added", new: change(func(s *corev1.Secret) { s.Data["orgID"] = []byte("1") }), want: true},
		{name: "labels", new: change(func(s *corev1.Secret) { s.Labels = nil }), want: true},
		{name: "type", new: change(func(s *corev1.Secret) { s.Type = corev1.SecretTypeBasicAuth }), want: true},
		{name: "deletion", new: change(func(s *corev1.Secret) { deleted(s) }), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ContentChanged().Update(event.UpdateEvent{ObjectOld: base, ObjectNew: tt.new}))
		})
	}
	assert.True(t, ContentChanged().Create(event.CreateEvent{Object: base}))
	assert.True(t, ContentChanged().Delete(event.DeleteEvent{Object: base}))
}

func TestValidate(t *testing.T) {
	t.Parallel()
	incomplete := destinations()
	incomplete.Names = nil
	sameKey := pods()
	sameKey.CountKey = destinationKey

	tests := []struct {
		name    string
		refs    []Referrer
		wantErr string
	}{
		{name: "one kind", refs: []Referrer{destinations()}},
		{name: "two kinds", refs: []Referrer{destinations(), pods()}},
		{name: "no kind", wantErr: "no referrer kind registered"},
		{name: "an entry misses a field", refs: []Referrer{incomplete}, wantErr: "are required"},
		{name: "a kind twice", refs: []Referrer{destinations(), destinations()}, wantErr: "registered twice"},
		{name: "a count key twice", refs: []Referrer{destinations(), sameKey}, wantErr: "registered twice"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := (&Reconciler{Referrers: tt.refs}).validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
