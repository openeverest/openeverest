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

package monitoring

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// newDestinationReconciler reads Instances through c both cached and live.
func newDestinationReconciler(c client.Client) (*DestinationReconciler, *events.FakeRecorder) {
	rec := events.NewFakeRecorder(10)
	return &DestinationReconciler{Client: c, APIReader: c, Recorder: rec}, rec
}

func reconcileDestination(t *testing.T, c client.Client, name string) *monitoringv1alpha1.MonitoringDestination {
	t.Helper()
	r, _ := newDestinationReconciler(c)
	return runDestinationReconcile(t, r, name)
}

func runDestinationReconcile(t *testing.T, r *DestinationReconciler, name string) *monitoringv1alpha1.MonitoringDestination {
	t.Helper()
	key := types.NamespacedName{Namespace: testNamespace, Name: name}
	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	dst := &monitoringv1alpha1.MonitoringDestination{}
	require.NoError(t, r.Get(t.Context(), key, dst))
	return dst
}

// namedInstance is newInstance under another name and namespace.
func namedInstance(name, namespace string, entries ...corev1alpha1.InstanceMonitoringDestination) *corev1alpha1.Instance {
	in := newInstance(pxcProvider(), entries...)
	in.Name, in.Namespace, in.UID = name, namespace, types.UID(name+"-uid")
	return in
}

func TestDestinationReconcilerAccepted(t *testing.T) {
	t.Parallel()

	token := newSecret("pmm-prod-credentials", map[string]string{"token": "glsa_token"})
	noURL := pmmDestination("pmm-prod")
	noURL.Spec.Parameters = nil
	unknownParameter := newDestination("prom", "prometheus")
	unknownParameter.Spec.Parameters = raw(`{"url":"https://prometheus.example.com"}`)
	basicAuth := newDestination("prom", "prometheus")
	basicAuth.Spec.Authentication = common.Authentication{Mode: "BasicAuth", CredentialsSecretRef: &common.SecretRef{Name: "prom-basic-auth"}}
	withCA := pmmDestination("pmm-prod")
	withCA.Spec.TLS = &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: "corp-ca", Key: "ca.crt"}}}

	tests := []struct {
		name        string
		objects     []client.Object
		destination string
		wantStatus  metav1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{
			name:        "class requirements met",
			objects:     []client.Object{pmmDestination("pmm-prod"), token},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionTrue,
			wantReason:  monitoringv1alpha1.ReasonAccepted,
			wantMessage: `valid for MonitoringClass "pmm"`,
		},
		{
			name:        "anonymous mode needs no Secret",
			objects:     []client.Object{newDestination("prom", "prometheus")},
			destination: "prom",
			wantStatus:  metav1.ConditionTrue,
			wantReason:  monitoringv1alpha1.ReasonAccepted,
			wantMessage: `valid for MonitoringClass "prometheus"`,
		},
		{
			name:        "class not found",
			objects:     []client.Object{newDestination("datadog", "datadog")},
			destination: "datadog",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonClassNotFound,
			wantMessage: `MonitoringClass "datadog" not found`,
		},
		{
			name:        "no parameters although the destination schema requires one",
			objects:     []client.Object{noURL, token},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonInvalidParameters,
			wantMessage: "(root): url is required",
		},
		{
			name:        "parameters rejected by the destination schema",
			objects:     []client.Object{unknownParameter},
			destination: "prom",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonInvalidParameters,
			wantMessage: `parameters rejected by the destinationParametersSchema of MonitoringClass "prometheus": `,
		},
		{
			name:        "mode the class does not define",
			objects:     []client.Object{basicAuth, newSecret("prom-basic-auth", map[string]string{"username": "u", "password": "p"})},
			destination: "prom",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonAuthModeUnsupported,
			wantMessage: `MonitoringClass "prometheus" has no authentication mode "BasicAuth"; it supports Anonymous`,
		},
		{
			name:        "credentials Secret missing",
			objects:     []client.Object{pmmDestination("pmm-prod")},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonCredentialsNotFound,
			wantMessage: `credentials Secret "pmm-prod-credentials" not found`,
		},
		{
			name:        "credentials Secret without a key the mode requires",
			objects:     []client.Object{pmmDestination("pmm-prod"), newSecret("pmm-prod-credentials", map[string]string{"apiKey": "glsa_token"})},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonCredentialsNotFound,
			wantMessage: `credentials Secret "pmm-prod-credentials": missing key "token" required by authentication mode "ServiceAccountToken"`,
		},
		{
			name:        "CA bundle in a Secret",
			objects:     []client.Object{withCA, token, newSecret("corp-ca", map[string]string{"ca.crt": testCAPEM(t)})},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionTrue,
			wantReason:  monitoringv1alpha1.ReasonAccepted,
			wantMessage: `valid for MonitoringClass "pmm"`,
		},
		{
			name:        "CA Secret missing",
			objects:     []client.Object{withCA, token},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonInvalidCABundle,
			wantMessage: `CA Secret "corp-ca" not found`,
		},
		{
			name:        "CA key is not a PEM bundle",
			objects:     []client.Object{withCA, token, newSecret("corp-ca", map[string]string{"ca.crt": "not a certificate"})},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonInvalidCABundle,
			wantMessage: `CA Secret "corp-ca" key "ca.crt": no PEM certificate found`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newFakeClient(t, append([]client.Object{pmmClass(), prometheusClass()}, tt.objects...)...)

			dst := reconcileDestination(t, c, tt.destination)

			accepted := findAccepted(t, dst.Status.Conditions)
			assert.Equal(t, tt.wantStatus, accepted.Status)
			assert.Equal(t, tt.wantReason, accepted.Reason)
			assert.Contains(t, accepted.Message, tt.wantMessage)
			assert.Equal(t, int64(2), accepted.ObservedGeneration)
			assert.Equal(t, []string{monitoringv1alpha1.InUseProtectionFinalizer}, dst.Finalizers)
		})
	}
}

// instanceBeingDeleted is an Instance listing the destination prom whose
// deletion a test finalizer holds.
func instanceBeingDeleted(name string) *corev1alpha1.Instance {
	in := namedInstance(name, testNamespace, entry("metrics", "prom"))
	in.Finalizers = []string{"test.openeverest.io/hold"}
	in.DeletionTimestamp = &metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return in
}

func TestDestinationReconcilerCountsAttachedInstances(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		instances []client.Object
		want      int32
	}{
		{name: "no Instance", want: 0},
		{
			name:      "one Instance",
			instances: []client.Object{namedInstance("orders", testNamespace, entry("metrics", "prom"))},
			want:      1,
		},
		{
			name: "an Instance being deleted does not count",
			instances: []client.Object{
				namedInstance("orders", testNamespace, entry("metrics", "prom")),
				instanceBeingDeleted("payments"),
			},
			want: 1,
		},
		{
			name: "Instances listing another destination or in another namespace do not count",
			instances: []client.Object{
				namedInstance("orders", testNamespace, entry("metrics", "pmm-prod")),
				namedInstance("orders", "team-b", entry("metrics", "prom")),
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newFakeClient(t, append([]client.Object{prometheusClass(), newDestination("prom", "prometheus")}, tt.instances...)...)

			dst := reconcileDestination(t, c, "prom")

			assert.Equal(t, tt.want, dst.Status.AttachedInstances)
			assert.Equal(t, monitoringv1alpha1.ReasonAccepted, findAccepted(t, dst.Status.Conditions).Reason)
		})
	}
}

func TestDestinationReconcilerCountFollowsInstances(t *testing.T) {
	t.Parallel()
	c := newFakeClient(t, prometheusClass(), newDestination("prom", "prometheus"), newInstance(pxcProvider(), entry("metrics", "prom")))

	require.Equal(t, int32(1), reconcileDestination(t, c, "prom").Status.AttachedInstances)

	updateInstance(t, c, func(in *corev1alpha1.Instance) { in.Spec.Monitoring = nil })
	dst := reconcileDestination(t, c, "prom")

	assert.Equal(t, int32(0), dst.Status.AttachedInstances)
	// The count and Accepted go in one apply, so neither prunes the other.
	assert.Equal(t, monitoringv1alpha1.ReasonAccepted, findAccepted(t, dst.Status.Conditions).Reason)
}

func TestDestinationDeletionWaitsForInstances(t *testing.T) {
	t.Parallel()
	c := newFakeClient(t, pxcProvider(), prometheusClass(), newDestination("prom", "prometheus"),
		newInstance(pxcProvider(), entry("metrics", "prom")),
	)
	r, rec := newDestinationReconciler(c)
	key := types.NamespacedName{Namespace: testNamespace, Name: "prom"}

	dst := runDestinationReconcile(t, r, "prom")
	require.NoError(t, c.Delete(t.Context(), dst))

	dst = runDestinationReconcile(t, r, "prom")
	assert.False(t, dst.DeletionTimestamp.IsZero())
	assert.Equal(t, []string{monitoringv1alpha1.InUseProtectionFinalizer}, dst.Finalizers)
	assert.Equal(t, int32(1), dst.Status.AttachedInstances)
	require.Len(t, rec.Events, 1)
	assert.Equal(t, "Warning DeletionBlocked deletion waits for 1 Instance(s) to stop using it: orders", <-rec.Events)

	reconcileInstance(t, c)
	accepted := findAccepted(t, getBinding(t, c, "metrics").Status.Conditions)
	assert.Equal(t, metav1.ConditionFalse, accepted.Status)
	assert.Equal(t, monitoringv1alpha1.ReasonDestinationTerminating, accepted.Reason)
	assert.Equal(t, `MonitoringDestination "prom" is being deleted; remove this entry to let the deletion finish`, accepted.Message)

	updateInstance(t, c, func(in *corev1alpha1.Instance) { in.Spec.Monitoring = nil })
	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
	require.NoError(t, err)

	err = c.Get(t.Context(), key, &monitoringv1alpha1.MonitoringDestination{})
	require.True(t, apierrors.IsNotFound(err), "destination still present: %v", err)
	assert.Empty(t, rec.Events)
}

func TestDestinationDeletionCountsInstancesBeingDeleted(t *testing.T) {
	t.Parallel()
	c := newFakeClient(t, prometheusClass(), newDestination("prom", "prometheus"),
		namedInstance("orders", testNamespace, entry("metrics", "prom")), instanceBeingDeleted("payments"),
	)
	r, rec := newDestinationReconciler(c)
	key := types.NamespacedName{Namespace: testNamespace, Name: "prom"}

	dst := runDestinationReconcile(t, r, "prom")
	require.Equal(t, int32(1), dst.Status.AttachedInstances, "an Instance being deleted counted while the destination is not")

	updateInstance(t, c, func(in *corev1alpha1.Instance) { in.Spec.Monitoring = nil })
	dst = runDestinationReconcile(t, r, "prom")
	require.Equal(t, int32(0), dst.Status.AttachedInstances)
	// Admitted by the in-use policy now; the Instance being deleted holds it.
	require.NoError(t, c.Delete(t.Context(), dst))

	dst = runDestinationReconcile(t, r, "prom")
	assert.Equal(t, []string{monitoringv1alpha1.InUseProtectionFinalizer}, dst.Finalizers)
	assert.Equal(t, int32(1), dst.Status.AttachedInstances, "the count states what the deletion waits for")
	require.Len(t, rec.Events, 1)
	assert.Equal(t, "Warning DeletionBlocked deletion waits for 1 Instance(s) to stop using it: payments", <-rec.Events)

	payments := &corev1alpha1.Instance{}
	require.NoError(t, c.Get(t.Context(), types.NamespacedName{Namespace: testNamespace, Name: "payments"}, payments))
	payments.Finalizers = nil
	require.NoError(t, c.Update(t.Context(), payments))
	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
	require.NoError(t, err)

	err = c.Get(t.Context(), key, &monitoringv1alpha1.MonitoringDestination{})
	require.True(t, apierrors.IsNotFound(err), "destination still present: %v", err)
	assert.Empty(t, rec.Events)
}

func TestDestinationDeletionReadsInstancesFromTheAPIServer(t *testing.T) {
	t.Parallel()
	// The cache has not seen the Instance yet; the API server has.
	cached := newFakeClient(t, prometheusClass(), newDestination("prom", "prometheus"))
	live := newFakeClient(t, newInstance(pxcProvider(), entry("metrics", "prom")))
	rec := events.NewFakeRecorder(10)
	r := &DestinationReconciler{Client: cached, APIReader: live, Recorder: rec}

	dst := runDestinationReconcile(t, r, "prom")
	require.Equal(t, int32(0), dst.Status.AttachedInstances)
	require.NoError(t, cached.Delete(t.Context(), dst))

	dst = runDestinationReconcile(t, r, "prom")

	assert.Equal(t, []string{monitoringv1alpha1.InUseProtectionFinalizer}, dst.Finalizers)
	assert.Equal(t, int32(1), dst.Status.AttachedInstances)
	assert.Len(t, rec.Events, 1)
}

func TestEnqueueDestinationsOfInstance(t *testing.T) {
	t.Parallel()
	before := newInstance(pxcProvider(), entry("pmm", "pmm-prod"), entry("prom", "prom"))
	after := newInstance(pxcProvider(), entry("prom", "prom"), entry("vm", "victoria"))
	afterDeleting := after.DeepCopy()
	afterDeleting.Finalizers = []string{"test.openeverest.io/hold", "test.openeverest.io/other"}
	afterDeleting.DeletionTimestamp = &metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	afterReleased := afterDeleting.DeepCopy()
	afterReleased.Finalizers = []string{"test.openeverest.io/other"}
	requests := func(names ...string) []reconcile.Request {
		reqs := make([]reconcile.Request, 0, len(names))
		for _, name := range names {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: name}})
		}
		return reqs
	}

	tests := []struct {
		name string
		send func(handler.EventHandler, workqueue.TypedRateLimitingInterface[reconcile.Request])
		want []reconcile.Request
	}{
		{
			name: "create enqueues every destination listed",
			send: func(h handler.EventHandler, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Create(t.Context(), event.CreateEvent{Object: before}, q)
			},
			want: requests("pmm-prod", "prom"),
		},
		{
			name: "update enqueues the destinations added and removed",
			send: func(h handler.EventHandler, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Update(t.Context(), event.UpdateEvent{ObjectOld: before, ObjectNew: after}, q)
			},
			want: requests("pmm-prod", "victoria"),
		},
		{
			name: "update that keeps the destinations enqueues nothing",
			send: func(h handler.EventHandler, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Update(t.Context(), event.UpdateEvent{ObjectOld: after, ObjectNew: after.DeepCopy()}, q)
			},
		},
		{
			name: "update that starts the deletion enqueues every destination listed",
			send: func(h handler.EventHandler, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Update(t.Context(), event.UpdateEvent{ObjectOld: after, ObjectNew: afterDeleting}, q)
			},
			want: requests("prom", "victoria"),
		},
		{
			name: "update of an Instance already being deleted that keeps the destinations enqueues nothing",
			send: func(h handler.EventHandler, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Update(t.Context(), event.UpdateEvent{ObjectOld: afterDeleting, ObjectNew: afterReleased}, q)
			},
		},
		{
			name: "delete enqueues every destination listed",
			send: func(h handler.EventHandler, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				h.Delete(t.Context(), event.DeleteEvent{Object: after}, q)
			},
			want: requests("prom", "victoria"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			q := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
			defer q.ShutDown()

			tt.send(enqueueDestinationsOfInstance(), q)

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

func TestDestinationReconcilerEnqueuesDestinationsUsingSecret(t *testing.T) {
	t.Parallel()
	withSecret := func(dst *monitoringv1alpha1.MonitoringDestination, namespace, secret string) *monitoringv1alpha1.MonitoringDestination {
		dst.Namespace = namespace
		dst.Spec.Authentication.CredentialsSecretRef = &common.SecretRef{Name: secret}
		return dst
	}
	withCA := newDestination("pmm-ca", "pmm")
	withCA.Spec.Authentication.CredentialsSecretRef = &common.SecretRef{Name: "pmm-ca-token"}
	withCA.Spec.TLS = &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: "pmm-token", Key: "ca.crt"}}}
	c := newFakeClient(t,
		withSecret(newDestination("pmm-prod", "pmm"), testNamespace, "pmm-token"),
		withSecret(newDestination("pmm-staging", "pmm"), testNamespace, "pmm-staging-token"),
		withSecret(newDestination("pmm-prod", "pmm"), "team-b", "pmm-token"),
		withCA,
		newDestination("prom", "prometheus"),
	)

	requests := (&DestinationReconciler{Client: c}).enqueueDestinationsForSecret(t.Context(), newSecret("pmm-token", nil))

	assert.ElementsMatch(t, []reconcile.Request{
		{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: "pmm-prod"}},
		{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: "pmm-ca"}},
	}, requests,
	)
}

func TestSecretReferrer(t *testing.T) {
	t.Parallel()
	dst := newDestination("pmm-prod", "pmm")
	dst.Spec.Authentication.CredentialsSecretRef = &common.SecretRef{Name: "pmm-token"}
	dst.Spec.TLS = &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: "corp-ca", Key: "ca.crt"}}}
	ref := SecretReferrer()

	assert.Equal(t, "MonitoringDestination", ref.Kind)
	assert.Equal(t, "monitoring.openeverest.io/attached-destinations", ref.CountKey)
	assert.Equal(t, []string{"pmm-token", "corp-ca"}, ref.Names(dst))
	assert.Empty(t, ref.Names(newSecret("pmm-token", nil)))
	assert.IsType(t, &monitoringv1alpha1.MonitoringDestinationList{}, ref.NewList())

	// The entry lists destinations by the index SetupIndexes registers.
	c := newFakeClient(t, dst)
	list := ref.NewList()
	require.NoError(t, c.List(t.Context(), list, client.InNamespace(testNamespace), client.MatchingFields{ref.IndexField: "corp-ca"}))
	assert.Len(t, list.(*monitoringv1alpha1.MonitoringDestinationList).Items, 1) //nolint:forcetypeassert
}

func TestDestinationReconcilerEnqueuesDestinationsOfClass(t *testing.T) {
	t.Parallel()
	teamB := newDestination("prom", "prometheus")
	teamB.Namespace = "team-b"
	c := newFakeClient(t, newDestination("prom", "prometheus"), teamB, pmmDestination("pmm-prod"))

	requests := (&DestinationReconciler{Client: c}).enqueueDestinationsForClass(t.Context(), prometheusClass())

	assert.ElementsMatch(t, []reconcile.Request{
		{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: "prom"}},
		{NamespacedName: types.NamespacedName{Namespace: "team-b", Name: "prom"}},
	}, requests,
	)
}

func testCAPEM(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}
