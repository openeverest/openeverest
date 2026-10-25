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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

func reconcileDestination(t *testing.T, c client.Client, name string) *monitoringv1alpha1.MonitoringDestination {
	t.Helper()
	key := types.NamespacedName{Namespace: testNamespace, Name: name}
	_, err := (&DestinationReconciler{Client: c}).Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	dst := &monitoringv1alpha1.MonitoringDestination{}
	require.NoError(t, c.Get(t.Context(), key, dst))
	return dst
}

func TestDestinationReconcilerAccepted(t *testing.T) {
	t.Parallel()

	noCredentials := pmmDestination("pmm-prod")
	noCredentials.Spec.CredentialsSecretRef = nil
	noURL := pmmDestination("pmm-prod")
	noURL.Spec.Parameters = raw(`{"verifyTLS":false}`)
	noParameters := pmmDestination("pmm-prod")
	noParameters.Spec.Parameters = nil
	basicAuth := newDestination("prom", "prometheus")
	basicAuth.Spec.CredentialsSecretRef = &common.SecretRef{Name: "prom-basic-auth"}

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
			objects:     []client.Object{pmmDestination("pmm-prod"), newSecret("pmm-prod-credentials", map[string]string{"apiKey": "glsa_token"})},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionTrue,
			wantReason:  monitoringv1alpha1.ReasonAccepted,
			wantMessage: `valid for MonitoringClass "pmm"`,
		},
		{
			name:        "class requiring no credentials",
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
			name:        "parameters rejected by the destination schema",
			objects:     []client.Object{noURL, newSecret("pmm-prod-credentials", map[string]string{"apiKey": "glsa_token"})},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonInvalidParameters,
			wantMessage: `parameters rejected by the destinationParametersSchema of MonitoringClass "pmm": `,
		},
		{
			name:        "no parameters although the destination schema requires one",
			objects:     []client.Object{noParameters, newSecret("pmm-prod-credentials", map[string]string{"apiKey": "glsa_token"})},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonInvalidParameters,
			wantMessage: "(root): url is required",
		},
		{
			name:        "no credentials Secret referenced",
			objects:     []client.Object{noCredentials},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonCredentialsNotFound,
			wantMessage: `MonitoringClass "pmm" requires credentialsSecretRef with keys apiKey`,
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
			name:        "credentials Secret without a required key",
			objects:     []client.Object{pmmDestination("pmm-prod"), newSecret("pmm-prod-credentials", map[string]string{"token": "glsa_token"})},
			destination: "pmm-prod",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonCredentialsNotFound,
			wantMessage: `credentials Secret "pmm-prod-credentials" is missing key "apiKey"`,
		},
		{
			name:        "referenced Secret missing although the class requires no keys",
			objects:     []client.Object{basicAuth},
			destination: "prom",
			wantStatus:  metav1.ConditionFalse,
			wantReason:  monitoringv1alpha1.ReasonCredentialsNotFound,
			wantMessage: `credentials Secret "prom-basic-auth" not found`,
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
			assert.Empty(t, dst.Finalizers)
		})
	}
}

func TestDestinationInUseIsDeletedImmediately(t *testing.T) {
	t.Parallel()
	c := newFakeClient(t, pxcProvider(), prometheusClass(), newDestination("prom", "prometheus"),
		newInstance(pxcProvider(), entry("metrics", "prom")))
	reconcileInstance(t, c)
	dst := reconcileDestination(t, c, "prom")
	require.Empty(t, dst.Finalizers)

	require.NoError(t, c.Delete(t.Context(), dst))
	err := c.Get(t.Context(), client.ObjectKeyFromObject(dst), &monitoringv1alpha1.MonitoringDestination{})
	require.True(t, apierrors.IsNotFound(err), "destination still present: %v", err)

	requests := (&BindingReconciler{Client: c}).enqueueInstancesForDestination(t.Context(), dst)
	assert.Equal(t, []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testInstance}}}, requests)

	reconcileInstance(t, c)
	b := getBinding(t, c, "metrics")
	accepted := findAccepted(t, b.Status.Conditions)
	assert.Equal(t, metav1.ConditionFalse, accepted.Status)
	assert.Equal(t, monitoringv1alpha1.ReasonDestinationNotFound, accepted.Reason)
	assert.Equal(t, corev1alpha1.MonitoringExecutionModeExtensionManaged, b.Spec.ExecutionMode)
	assert.Equal(t, &common.ObjectRef{Name: "prometheus"}, b.Spec.ClassRef)
}

func TestDestinationReconcilerEnqueuesDestinationsUsingSecret(t *testing.T) {
	t.Parallel()
	withSecret := func(dst *monitoringv1alpha1.MonitoringDestination, namespace, secret string) *monitoringv1alpha1.MonitoringDestination {
		dst.Namespace = namespace
		dst.Spec.CredentialsSecretRef = &common.SecretRef{Name: secret}
		return dst
	}
	c := newFakeClient(t,
		withSecret(newDestination("pmm-prod", "pmm"), testNamespace, "pmm-token"),
		withSecret(newDestination("pmm-staging", "pmm"), testNamespace, "pmm-staging-token"),
		withSecret(newDestination("pmm-prod", "pmm"), "team-b", "pmm-token"),
		newDestination("prom", "prometheus"),
	)

	requests := (&DestinationReconciler{Client: c}).enqueueDestinationsForSecret(t.Context(), newSecret("pmm-token", nil))

	assert.Equal(t, []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: "pmm-prod"}}}, requests)
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
	}, requests)
}
