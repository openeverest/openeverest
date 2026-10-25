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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

func TestBindingReconcilerMaterialisesOneBindingPerEntry(t *testing.T) {
	t.Parallel()
	pmm := entry("pmm", "pmm-prod")
	pmm.Parameters = raw(`{"resources":{"limits":{"memory":"256Mi"}}}`)
	c := newFakeClient(t,
		pxcProvider(), pmmClass(), prometheusClass(),
		pmmDestination("pmm-prod"), newDestination("prom", "prometheus"),
		newInstance(pxcProvider(), pmm, entry("metrics", "prom")),
	)

	reconcileInstance(t, c)

	list := &monitoringv1alpha1.MonitoringBindingList{}
	require.NoError(t, c.List(t.Context(), list, client.InNamespace(testNamespace)))
	assert.Len(t, list.Items, 2)

	tests := []struct {
		entry       string
		destination string
		class       string
		mode        corev1alpha1.MonitoringExecutionMode
		parameters  string
	}{
		{entry: "pmm", destination: "pmm-prod", class: "pmm", mode: corev1alpha1.MonitoringExecutionModeProviderManaged, parameters: `{"resources":{"limits":{"memory":"256Mi"}}}`},
		{entry: "metrics", destination: "prom", class: "prometheus", mode: corev1alpha1.MonitoringExecutionModeExtensionManaged},
	}
	for _, tt := range tests {
		b := getBinding(t, c, tt.entry)
		assert.Equal(t, rtmonitoring.BindingLabels(testInstance, tt.entry, tt.destination, tt.class), b.Labels)
		assert.Equal(t, []metav1.OwnerReference{{
			APIVersion: "core.openeverest.io/v1alpha1", Kind: "Instance", Name: testInstance, UID: testUID,
			Controller: new(true), BlockOwnerDeletion: new(true),
		}}, b.OwnerReferences)
		assert.Equal(t, common.ObjectRef{Name: testInstance}, b.Spec.InstanceRef)
		assert.Equal(t, common.MonitoringDestinationRef{Kind: "MonitoringDestination", Name: tt.destination}, b.Spec.DestinationRef)
		assert.Equal(t, &common.ObjectRef{Name: tt.class}, b.Spec.ClassRef)
		assert.Equal(t, tt.mode, b.Spec.ExecutionMode)
		if tt.parameters == "" {
			assert.Nil(t, b.Spec.Parameters)
		} else {
			require.NotNil(t, b.Spec.Parameters)
			assert.JSONEq(t, tt.parameters, string(b.Spec.Parameters.Raw))
		}
		accepted := findAccepted(t, b.Status.Conditions)
		assert.Equal(t, metav1.ConditionTrue, accepted.Status)
		assert.Equal(t, monitoringv1alpha1.ReasonAccepted, accepted.Reason)
		assert.Equal(t, "resolved to "+string(tt.mode), accepted.Message)
	}
}

func TestBindingReconcilerRejectsEntry(t *testing.T) {
	t.Parallel()

	otel := newDestination("otel", "otel")
	unclaimed := &monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{Name: "otel"},
		Spec: monitoringv1alpha1.MonitoringClassSpec{
			ControllerName:   "openeverest.io/monitoring-otel",
			ExtensionManaged: &monitoringv1alpha1.ExtensionManagedFulfilment{},
		},
	}
	badParameters := entry("pmm", "pmm-prod")
	badParameters.Parameters = raw(`{"replicas":3}`)

	tests := []struct {
		name        string
		provider    *corev1alpha1.Provider
		noProvider  bool
		objects     []client.Object
		entries     []corev1alpha1.InstanceMonitoringDestination
		check       string
		wantReason  string
		wantMessage string
		wantClass   string
	}{
		{
			name:        "destination not found",
			provider:    pxcProvider(),
			entries:     []corev1alpha1.InstanceMonitoringDestination{entry("pmm", "missing")},
			check:       "pmm",
			wantReason:  monitoringv1alpha1.ReasonDestinationNotFound,
			wantMessage: `MonitoringDestination "missing" not found`,
		},
		{
			name:        "class not found",
			provider:    pxcProvider(),
			objects:     []client.Object{newDestination("orphan", "gone")},
			entries:     []corev1alpha1.InstanceMonitoringDestination{entry("metrics", "orphan")},
			check:       "metrics",
			wantReason:  monitoringv1alpha1.ReasonClassNotFound,
			wantMessage: `MonitoringClass "gone" not found`,
		},
		{
			name:        "class not claimed",
			provider:    pxcProvider(),
			objects:     []client.Object{otel, unclaimed},
			entries:     []corev1alpha1.InstanceMonitoringDestination{entry("traces", "otel")},
			check:       "traces",
			wantReason:  monitoringv1alpha1.ReasonClassNotAccepted,
			wantMessage: `no controller has claimed MonitoringClass "otel" (controllerName "openeverest.io/monitoring-otel")`,
			wantClass:   "otel",
		},
		{
			name:        "provider not found",
			provider:    pxcProvider(),
			noProvider:  true,
			entries:     []corev1alpha1.InstanceMonitoringDestination{entry("pmm", "pmm-prod")},
			check:       "pmm",
			wantReason:  monitoringv1alpha1.ReasonProviderNotFound,
			wantMessage: `Provider "percona-xtradb-cluster" not found`,
			wantClass:   "pmm",
		},
		{
			name:        "parameters rejected by the instance schema",
			provider:    pxcProvider(),
			entries:     []corev1alpha1.InstanceMonitoringDestination{badParameters},
			check:       "pmm",
			wantReason:  monitoringv1alpha1.ReasonInvalidParameters,
			wantMessage: `parameters rejected by the instanceParametersSchema of MonitoringClass "pmm": `,
			wantClass:   "pmm",
		},
		{
			name:        "provider without the integration",
			provider:    kserveProvider(),
			entries:     []corev1alpha1.InstanceMonitoringDestination{entry("pmm", "pmm-prod")},
			check:       "pmm",
			wantReason:  monitoringv1alpha1.ReasonIntegrationUnsupported,
			wantMessage: `Provider "provider-kserve" does not declare integration "pmm"`,
			wantClass:   "pmm",
		},
		{
			name:        "integration already used by another entry",
			provider:    pxcProvider(),
			objects:     []client.Object{pmmDestination("pmm-staging")},
			entries:     []corev1alpha1.InstanceMonitoringDestination{entry("pmm", "pmm-prod"), entry("pmm-staging", "pmm-staging")},
			check:       "pmm-staging",
			wantReason:  monitoringv1alpha1.ReasonIntegrationInUse,
			wantMessage: `integration "pmm" is already fulfilled by entry "pmm"`,
			wantClass:   "pmm",
		},
		{
			name:        "no component exposes metrics",
			provider:    pgProvider(),
			entries:     []corev1alpha1.InstanceMonitoringDestination{entry("metrics", "prom")},
			check:       "metrics",
			wantReason:  monitoringv1alpha1.ReasonIncompatible,
			wantMessage: `no component satisfies the requirements of MonitoringClass "prometheus"`,
			wantClass:   "prometheus",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			objs := append([]client.Object{
				pmmClass(), prometheusClass(), pmmDestination("pmm-prod"), newDestination("prom", "prometheus"),
				newInstance(tt.provider, tt.entries...),
			}, tt.objects...)
			if !tt.noProvider {
				objs = append(objs, tt.provider)
			}
			c := newFakeClient(t, objs...)

			reconcileInstance(t, c)

			b := getBinding(t, c, tt.check)
			accepted := findAccepted(t, b.Status.Conditions)
			assert.Equal(t, metav1.ConditionFalse, accepted.Status)
			assert.Equal(t, tt.wantReason, accepted.Reason)
			assert.Contains(t, accepted.Message, tt.wantMessage)
			assert.Equal(t, b.Generation, accepted.ObservedGeneration)
			assert.Empty(t, b.Spec.ExecutionMode)
			if tt.wantClass == "" {
				assert.Nil(t, b.Spec.ClassRef)
				assert.NotContains(t, b.Labels, monitoringv1alpha1.BindingClassLabel)
			} else {
				assert.Equal(t, &common.ObjectRef{Name: tt.wantClass}, b.Spec.ClassRef)
				assert.Equal(t, tt.wantClass, b.Labels[monitoringv1alpha1.BindingClassLabel])
			}
		})
	}
}

func TestBindingReconcilerSelectsMode(t *testing.T) {
	t.Parallel()

	pinned := func(e corev1alpha1.InstanceMonitoringDestination, mode corev1alpha1.MonitoringExecutionMode) corev1alpha1.InstanceMonitoringDestination {
		e.ExecutionMode = mode
		return e
	}
	oldAgent := pmmClass()
	oldAgent.Spec.ProviderManaged.AgentVersion = "2.44.0"

	tests := []struct {
		name        string
		provider    *corev1alpha1.Provider
		class       *monitoringv1alpha1.MonitoringClass
		destination *monitoringv1alpha1.MonitoringDestination
		entry       corev1alpha1.InstanceMonitoringDestination
		wantMode    corev1alpha1.MonitoringExecutionMode
		wantReason  string
		wantMessage string
	}{
		{
			name:        "ProviderManaged is preferred",
			provider:    pxcProvider(),
			class:       hybridPMMClass(),
			destination: pmmDestination("pmm-prod"),
			entry:       entry("pmm", "pmm-prod"),
			wantMode:    corev1alpha1.MonitoringExecutionModeProviderManaged,
		},
		{
			name:        "an ExtensionManaged pin wins over ProviderManaged",
			provider:    pxcProvider(),
			class:       hybridPMMClass(),
			destination: pmmDestination("pmm-prod"),
			entry:       pinned(entry("pmm", "pmm-prod"), corev1alpha1.MonitoringExecutionModeExtensionManaged),
			wantMode:    corev1alpha1.MonitoringExecutionModeExtensionManaged,
		},
		{
			name:        "ExtensionManaged when the provider lacks the integration",
			provider:    kserveProvider(),
			class:       hybridPMMClass(),
			destination: pmmDestination("pmm-prod"),
			entry:       entry("pmm", "pmm-prod"),
			wantMode:    corev1alpha1.MonitoringExecutionModeExtensionManaged,
		},
		{
			name:        "a ProviderManaged pin on a class without the block",
			provider:    pxcProvider(),
			class:       prometheusClass(),
			destination: newDestination("prom", "prometheus"),
			entry:       pinned(entry("metrics", "prom"), corev1alpha1.MonitoringExecutionModeProviderManaged),
			wantReason:  monitoringv1alpha1.ReasonIncompatible,
			wantMessage: `MonitoringClass "prometheus" declares no providerManaged block`,
		},
		{
			name:        "agent version outside the provider's range",
			provider:    pxcProvider(),
			class:       oldAgent,
			destination: pmmDestination("pmm-prod"),
			entry:       entry("pmm", "pmm-prod"),
			wantReason:  monitoringv1alpha1.ReasonIncompatible,
			wantMessage: `MonitoringClass "pmm" agentVersion 2.44.0 is outside the range ">=3.0.0 <4.0.0" Provider "percona-xtradb-cluster" supports`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newFakeClient(t, tt.provider, tt.class, tt.destination, newInstance(tt.provider, tt.entry))

			reconcileInstance(t, c)

			b := getBinding(t, c, tt.entry.Name)
			assert.Equal(t, tt.wantMode, b.Spec.ExecutionMode)
			accepted := findAccepted(t, b.Status.Conditions)
			if tt.wantReason == "" {
				assert.Equal(t, metav1.ConditionTrue, accepted.Status)
				return
			}
			assert.Equal(t, metav1.ConditionFalse, accepted.Status)
			assert.Equal(t, tt.wantReason, accepted.Reason)
			assert.Equal(t, tt.wantMessage, accepted.Message)
		})
	}
}

func TestBindingReconcilerNeverChangesModeOnItsOwn(t *testing.T) {
	t.Parallel()

	t.Run("a better mode becoming available", func(t *testing.T) {
		t.Parallel()
		provider := pxcProvider()
		provider.Spec.Monitoring.Integrations = nil
		c := newFakeClient(t, provider, hybridPMMClass(), pmmDestination("pmm-prod"), newInstance(provider, entry("pmm", "pmm-prod")))
		reconcileInstance(t, c)
		require.Equal(t, corev1alpha1.MonitoringExecutionModeExtensionManaged, getBinding(t, c, "pmm").Spec.ExecutionMode)

		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(provider), provider))
		provider.Spec.Monitoring.Integrations = pmmIntegration()
		require.NoError(t, c.Update(t.Context(), provider))
		reconcileInstance(t, c)

		b := getBinding(t, c, "pmm")
		assert.Equal(t, corev1alpha1.MonitoringExecutionModeExtensionManaged, b.Spec.ExecutionMode)
		assert.Equal(t, metav1.ConditionTrue, findAccepted(t, b.Status.Conditions).Status)
	})

	t.Run("destination recreated with another class", func(t *testing.T) {
		t.Parallel()
		c := newFakeClient(t, pxcProvider(), prometheusClass(), victoriaMetricsClass(),
			newDestination("scrape", "prometheus"), newInstance(pxcProvider(), entry("metrics", "scrape")))
		reconcileInstance(t, c)

		require.NoError(t, c.Delete(t.Context(), newDestination("scrape", "prometheus")))
		require.NoError(t, c.Create(t.Context(), newDestination("scrape", "victoriametrics")))
		reconcileInstance(t, c)

		b := getBinding(t, c, "metrics")
		assert.Equal(t, corev1alpha1.MonitoringExecutionModeExtensionManaged, b.Spec.ExecutionMode)
		assert.Equal(t, &common.ObjectRef{Name: "prometheus"}, b.Spec.ClassRef)
		assert.Equal(t, "prometheus", b.Labels[monitoringv1alpha1.BindingClassLabel])
		accepted := findAccepted(t, b.Status.Conditions)
		assert.Equal(t, metav1.ConditionFalse, accepted.Status)
		assert.Equal(t, monitoringv1alpha1.ReasonIncompatible, accepted.Reason)
		assert.Equal(t, `binding was created for MonitoringClass "prometheus" but MonitoringDestination "scrape" now references "victoriametrics"`, accepted.Message)
	})
}

func TestBindingReconcilerRecreatesBindingForChangedEntry(t *testing.T) {
	t.Parallel()

	t.Run("entry pointed at another destination", func(t *testing.T) {
		t.Parallel()
		c := newFakeClient(t, pxcProvider(), prometheusClass(), victoriaMetricsClass(),
			newDestination("prom", "prometheus"), newDestination("vm", "victoriametrics"),
			newInstance(pxcProvider(), entry("metrics", "prom")))
		reconcileInstance(t, c)

		updateInstance(t, c, func(in *corev1alpha1.Instance) {
			in.Spec.Monitoring.Destinations[0].DestinationRef.Name = "vm"
		})
		reconcileInstance(t, c)

		b := getBinding(t, c, "metrics")
		assert.Equal(t, common.MonitoringDestinationRef{Kind: "MonitoringDestination", Name: "vm"}, b.Spec.DestinationRef)
		assert.Equal(t, &common.ObjectRef{Name: "victoriametrics"}, b.Spec.ClassRef)
		assert.Equal(t, rtmonitoring.BindingLabels(testInstance, "metrics", "vm", "victoriametrics"), b.Labels)
		assert.Equal(t, metav1.ConditionTrue, findAccepted(t, b.Status.Conditions).Status)
	})

	t.Run("entry pinned to another mode", func(t *testing.T) {
		t.Parallel()
		c := newFakeClient(t, pxcProvider(), hybridPMMClass(), pmmDestination("pmm-prod"),
			newInstance(pxcProvider(), entry("pmm", "pmm-prod")))
		reconcileInstance(t, c)
		require.Equal(t, corev1alpha1.MonitoringExecutionModeProviderManaged, getBinding(t, c, "pmm").Spec.ExecutionMode)

		updateInstance(t, c, func(in *corev1alpha1.Instance) {
			in.Spec.Monitoring.Destinations[0].ExecutionMode = corev1alpha1.MonitoringExecutionModeExtensionManaged
		})
		reconcileInstance(t, c)

		b := getBinding(t, c, "pmm")
		assert.Equal(t, corev1alpha1.MonitoringExecutionModeExtensionManaged, b.Spec.ExecutionMode)
		assert.Equal(t, metav1.ConditionTrue, findAccepted(t, b.Status.Conditions).Status)
	})
}

func TestBindingReconcilerKeepsIntegrationWithItsExistingBinding(t *testing.T) {
	t.Parallel()
	c := newFakeClient(t, pxcProvider(), pmmClass(), pmmDestination("pmm-prod"), pmmDestination("pmm-staging"),
		newInstance(pxcProvider(), entry("pmm", "pmm-prod")))
	reconcileInstance(t, c)

	updateInstance(t, c, func(in *corev1alpha1.Instance) {
		in.Spec.Monitoring.Destinations = append([]corev1alpha1.InstanceMonitoringDestination{entry("staging", "pmm-staging")},
			in.Spec.Monitoring.Destinations...)
	})
	reconcileInstance(t, c)

	kept := getBinding(t, c, "pmm")
	assert.Equal(t, corev1alpha1.MonitoringExecutionModeProviderManaged, kept.Spec.ExecutionMode)
	assert.Equal(t, metav1.ConditionTrue, findAccepted(t, kept.Status.Conditions).Status)
	added := findAccepted(t, getBinding(t, c, "staging").Status.Conditions)
	assert.Equal(t, monitoringv1alpha1.ReasonIntegrationInUse, added.Reason)
	assert.Equal(t, `integration "pmm" is already fulfilled by entry "pmm"`, added.Message)
}

func TestBindingReconcilerFollowsEntryParameters(t *testing.T) {
	t.Parallel()
	pmm := entry("pmm", "pmm-prod")
	pmm.Parameters = raw(`{"resources":{"limits":{"memory":"512Mi"}}}`)
	existing := &monitoringv1alpha1.MonitoringBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       rtmonitoring.BindingName(testInstance, "pmm"),
			Namespace:  testNamespace,
			Generation: 3,
			Labels:     rtmonitoring.BindingLabels(testInstance, "pmm", "pmm-prod", "pmm"),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "core.openeverest.io/v1alpha1", Kind: "Instance", Name: testInstance, UID: testUID, Controller: new(true),
			}},
		},
		Spec: monitoringv1alpha1.MonitoringBindingSpec{
			InstanceRef:    common.ObjectRef{Name: testInstance},
			DestinationRef: common.MonitoringDestinationRef{Kind: "MonitoringDestination", Name: "pmm-prod"},
			ClassRef:       &common.ObjectRef{Name: "pmm"},
			ExecutionMode:  corev1alpha1.MonitoringExecutionModeProviderManaged,
			Parameters:     raw(`{"resources":{"limits":{"memory":"256Mi"}}}`),
		},
	}
	c := newFakeClient(t, pxcProvider(), pmmClass(), pmmDestination("pmm-prod"), newInstance(pxcProvider(), pmm), existing)

	reconcileInstance(t, c)

	b := getBinding(t, c, "pmm")
	require.NotNil(t, b.Spec.Parameters)
	assert.JSONEq(t, `{"resources":{"limits":{"memory":"512Mi"}}}`, string(b.Spec.Parameters.Raw))
	accepted := findAccepted(t, b.Status.Conditions)
	assert.Equal(t, metav1.ConditionTrue, accepted.Status)
	assert.Equal(t, int64(3), accepted.ObservedGeneration)
}

func TestBindingReconcilerDeletesStaleBindings(t *testing.T) {
	t.Parallel()
	foreign := &monitoringv1alpha1.MonitoringBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orders-handmade",
			Namespace: testNamespace,
			Labels:    rtmonitoring.BindingLabels(testInstance, "handmade", "prom", "prometheus"),
		},
	}
	c := newFakeClient(t, pxcProvider(), pmmClass(), prometheusClass(), pmmDestination("pmm-prod"), newDestination("prom", "prometheus"),
		newInstance(pxcProvider(), entry("pmm", "pmm-prod"), entry("metrics", "prom")), foreign)
	reconcileInstance(t, c)

	updateInstance(t, c, func(in *corev1alpha1.Instance) {
		in.Spec.Monitoring.Destinations = in.Spec.Monitoring.Destinations[:1]
	})
	reconcileInstance(t, c)

	list := &monitoringv1alpha1.MonitoringBindingList{}
	require.NoError(t, c.List(t.Context(), list, client.InNamespace(testNamespace)))
	names := make([]string, 0, len(list.Items))
	for _, b := range list.Items {
		names = append(names, b.Name)
	}
	assert.ElementsMatch(t, []string{rtmonitoring.BindingName(testInstance, "pmm"), "orders-handmade"}, names)
}
