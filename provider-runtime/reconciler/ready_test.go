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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestSetReadyCondition(t *testing.T) {
	t.Parallel()

	backupFailed := metav1.Condition{
		Type: v1alpha1.ConditionBackupConfigured, Status: metav1.ConditionFalse,
		Reason: "LimitsExceeded", Message: "too many storages",
	}
	tests := []struct {
		name        string
		status      v1alpha1.InstanceStatus
		blocker     *blocker
		wantStatus  metav1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{
			name:        "blocker wins over a Ready phase",
			status:      v1alpha1.InstanceStatus{Phase: v1alpha1.InstancePhaseReady},
			blocker:     &blocker{reason: v1alpha1.ReasonValidationFailed, message: "replicas must be odd"},
			wantStatus:  metav1.ConditionFalse,
			wantReason:  v1alpha1.ReasonValidationFailed,
			wantMessage: "replicas must be odd",
		},
		{
			name: "a False reference condition keeps a Ready phase from being Ready",
			status: v1alpha1.InstanceStatus{
				Phase:      v1alpha1.InstancePhaseReady,
				Conditions: []metav1.Condition{backupFailed},
			},
			wantStatus:  metav1.ConditionFalse,
			wantReason:  "LimitsExceeded",
			wantMessage: "too many storages",
		},
		{
			name: "Ready phase with healthy references is Ready",
			status: v1alpha1.InstanceStatus{
				Phase: v1alpha1.InstancePhaseReady,
				Conditions: []metav1.Condition{{
					Type: v1alpha1.ConditionBackupConfigured, Status: metav1.ConditionTrue, Reason: "Configured",
				}},
			},
			wantStatus:  metav1.ConditionTrue,
			wantReason:  v1alpha1.ReasonReady,
			wantMessage: "The instance is ready",
		},
		{
			name:        "any other phase is the reason, with the status message",
			status:      v1alpha1.InstanceStatus{Phase: v1alpha1.InstancePhaseProvisioning, Message: "creating pods"},
			wantStatus:  metav1.ConditionFalse,
			wantReason:  "Provisioning",
			wantMessage: "creating pods",
		},
		{
			name:        "Failed without a message still explains itself",
			status:      v1alpha1.InstanceStatus{Phase: v1alpha1.InstancePhaseFailed},
			wantStatus:  metav1.ConditionFalse,
			wantReason:  "Failed",
			wantMessage: "The instance is in phase Failed",
		},
		{
			name:        "no phase yet reads as Pending",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  "Pending",
			wantMessage: "The instance is in phase Pending",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := &v1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Generation: 4}, Status: tc.status}
			setReadyCondition(in, tc.blocker, metav1.Now())

			c := meta.FindStatusCondition(in.Status.Conditions, v1alpha1.ConditionReady)
			require.NotNil(t, c)
			assert.Equal(t, tc.wantStatus, c.Status)
			assert.Equal(t, tc.wantReason, c.Reason)
			assert.Equal(t, tc.wantMessage, c.Message)
			assert.Equal(t, int64(4), c.ObservedGeneration)
		})
	}
}

type readyTestProvider struct {
	validateErrs field.ErrorList
	status       controller.Status
	calls        int
}

func (p *readyTestProvider) Name() string                       { return "test-provider" }
func (p *readyTestProvider) Types() func(*runtime.Scheme) error { return nil }
func (p *readyTestProvider) Sync(*controller.Context) error     { return nil }
func (p *readyTestProvider) Cleanup(*controller.Context) error  { return nil }

func (p *readyTestProvider) Validate(controller.ValidationInput) controller.ValidationResult {
	p.calls++
	return controller.ValidationResult{Errors: p.validateErrs}
}

func (p *readyTestProvider) Status(*controller.Context) (controller.Status, error) {
	return p.status, nil
}

func readyTestInstance(mutate func(*v1alpha1.Instance)) *v1alpha1.Instance {
	in := &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "orders",
			Namespace:  "team-a",
			Generation: 2,
			Labels:     map[string]string{controller.ProviderLabel: "test-provider"},
			Finalizers: []string{finalizerName},
		},
		Spec: v1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: "test-provider"}},
	}
	if mutate != nil {
		mutate(in)
	}
	return in
}

func newReadyTestReconciler(p providerAdapter, objs ...client.Object) *ProviderReconciler {
	scheme := newTestScheme()
	_ = corev1.AddToScheme(scheme)
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&v1alpha1.Instance{}).
		Build()
	return &ProviderReconciler{provider: p, Client: c}
}

func reconcileAndGet(t *testing.T, r *ProviderReconciler) (*v1alpha1.Instance, error) {
	t.Helper()
	key := client.ObjectKey{Namespace: "team-a", Name: "orders"}
	_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
	got := &v1alpha1.Instance{}
	require.NoError(t, r.Get(t.Context(), key, got))
	return got, err
}

func requireReady(t *testing.T, in *v1alpha1.Instance, status metav1.ConditionStatus, reason string) *metav1.Condition {
	t.Helper()
	c := meta.FindStatusCondition(in.Status.Conditions, v1alpha1.ConditionReady)
	require.NotNil(t, c, "Ready condition missing")
	assert.Equal(t, status, c.Status)
	assert.Equal(t, reason, c.Reason)
	assert.Equal(t, in.Generation, in.Status.ObservedGeneration)
	return c
}

func TestReconcile_ProviderNotFound(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{}
	r := newReadyTestReconciler(p, readyTestInstance(nil))

	got, err := reconcileAndGet(t, r)
	require.NoError(t, err)

	c := requireReady(t, got, metav1.ConditionFalse, v1alpha1.ReasonProviderNotFound)
	assert.Equal(t, "provider test-provider not found", c.Message)
	assert.Equal(t, v1alpha1.InstancePhasePending, got.Status.Phase)
	assert.Zero(t, p.calls, "the provider must not be called")
}

func TestReconcile_UserSecretNotFound(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{status: controller.Ready()}
	provider := &v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}
	in := readyTestInstance(func(in *v1alpha1.Instance) {
		in.Spec.UserSecretRef = &common.SecretRef{Name: "orders-users"}
	})
	r := newReadyTestReconciler(p, provider, in)

	got, err := reconcileAndGet(t, r)
	require.NoError(t, err)
	c := requireReady(t, got, metav1.ConditionFalse, v1alpha1.ReasonUserSecretNotFound)
	assert.Equal(t, "secret orders-users named by spec.userSecretRef not found in team-a", c.Message)
	assert.Equal(t, v1alpha1.InstancePhasePending, got.Status.Phase)
	assert.Zero(t, p.calls, "the provider must not be called")

	require.NoError(t, r.Create(t.Context(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "orders-users", Namespace: "team-a"},
	}))
	got, err = reconcileAndGet(t, r)
	require.NoError(t, err)
	requireReady(t, got, metav1.ConditionTrue, v1alpha1.ReasonReady)
	assert.Equal(t, 1, p.calls)
}

func TestReconcile_ValidationFailedKeepsPhase(t *testing.T) {
	t.Parallel()
	replicas := field.NewPath("spec", "components").Key("engine").Child("replicas")
	p := &readyTestProvider{validateErrs: field.ErrorList{field.Invalid(replicas, 2, "must be odd")}}
	provider := &v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}
	in := readyTestInstance(func(in *v1alpha1.Instance) {
		in.Status.Phase = v1alpha1.InstancePhaseReady
	})
	r := newReadyTestReconciler(p, provider, in)

	got, err := reconcileAndGet(t, r)
	require.NoError(t, err, "a rejected spec is not retried with backoff")
	c := requireReady(t, got, metav1.ConditionFalse, v1alpha1.ReasonValidationFailed)
	assert.Equal(t, "spec.components[engine].replicas: Invalid value: 2: must be odd", c.Message)
	assert.Equal(t, v1alpha1.InstancePhaseReady, got.Status.Phase, "a rejected spec must not move the phase to Failed")
}

func TestReconcile_ReadyFollowsProviderStatus(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{status: controller.Provisioning("starting pods")}
	provider := &v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}
	r := newReadyTestReconciler(p, provider, readyTestInstance(nil))

	got, err := reconcileAndGet(t, r)
	require.NoError(t, err)
	c := requireReady(t, got, metav1.ConditionFalse, string(got.Status.Phase))
	assert.Equal(t, "starting pods", c.Message)

	p.status = controller.Ready()
	got, err = reconcileAndGet(t, r)
	require.NoError(t, err)
	requireReady(t, got, metav1.ConditionTrue, v1alpha1.ReasonReady)
}

func TestInstancesForSecret(t *testing.T) {
	t.Parallel()
	withSecret := func(ns, provider, secret string) func(*v1alpha1.Instance) {
		return func(in *v1alpha1.Instance) {
			in.Namespace = ns
			in.Spec.ProviderRef.Name = provider
			if secret != "" {
				in.Spec.UserSecretRef = &common.SecretRef{Name: secret}
			}
		}
	}
	named := func(name string, mutate func(*v1alpha1.Instance)) *v1alpha1.Instance {
		in := readyTestInstance(mutate)
		in.Name = name
		return in
	}
	r := newReadyTestReconciler(&readyTestProvider{},
		named("match", withSecret("team-a", "test-provider", "users")),
		named("other-secret", withSecret("team-a", "test-provider", "admins")),
		named("no-secret", withSecret("team-a", "test-provider", "")),
		named("other-provider", withSecret("team-a", "someone-else", "users")),
		named("other-namespace", withSecret("team-b", "test-provider", "users")),
	)

	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "users", Namespace: "team-a"}}
	reqs := r.instancesForSecret(t.Context(), secret)
	require.Len(t, reqs, 1)
	assert.Equal(t, "match", reqs[0].Name)
}
