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
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

const (
	monTestClusterID   = "6f1c2d3e-0000-4000-8000-000000000001"
	monTestInstanceUID = "6f1c2d3e-0000-4000-8000-0000000000aa"
)

// monProvider implements both monitoring interfaces on top of the plain
// test provider.
type monProvider struct {
	readyTestProvider

	sources    *v1alpha1.MonitoringSources
	sourcesErr error
}

func (p *monProvider) MonitoringSources(*controller.Context) (*v1alpha1.MonitoringSources, error) {
	return p.sources.DeepCopy(), p.sourcesErr
}

func (p *monProvider) MonitoringIntegrations() []string { return []string{"pmm"} }

// vmagentProvider renders an integration other than pmm.
type vmagentProvider struct{ monProvider }

func (p *vmagentProvider) MonitoringIntegrations() []string { return []string{"vmagent"} }

// sourcesOnlyProvider implements MonitoringSourcesProvider only.
type sourcesOnlyProvider struct{ readyTestProvider }

func (p *sourcesOnlyProvider) MonitoringSources(*controller.Context) (*v1alpha1.MonitoringSources, error) {
	return &v1alpha1.MonitoringSources{}, nil
}

// requiredSourcesProvider records what RequiredSources answers during Sync.
type requiredSourcesProvider struct {
	monProvider

	got controller.RequiredSources
	err error
}

func (p *requiredSourcesProvider) Sync(c *controller.Context) error {
	p.got, p.err = c.RequiredSources()
	return nil
}

// monTypeConverter makes the fake client's server-side apply follow the
// generated monitoring CRD schemas (conditions merge by type).
func monTypeConverter(t *testing.T) managedfields.TypeConverter { //nolint:ireturn // the fake client takes the interface
	t.Helper()
	crds := os.DirFS(filepath.Join("..", "..", "config", "crd", "bases"))
	models := make(map[string]*spec.Schema)
	for _, plural := range []string{"monitoringclasses", "monitoringdestinations", "monitoringbindings"} {
		raw, err := fs.ReadFile(crds, "monitoring.openeverest.io_"+plural+".yaml")
		require.NoError(t, err)
		crd := &apiextensionsv1.CustomResourceDefinition{}
		require.NoError(t, yaml.Unmarshal(raw, crd))
		for _, v := range crd.Spec.Versions {
			js, err := json.Marshal(v.Schema.OpenAPIV3Schema)
			require.NoError(t, err)
			s := &spec.Schema{}
			require.NoError(t, json.Unmarshal(js, s))
			s.AddExtension("x-kubernetes-group-version-kind", []any{
				map[string]any{"group": crd.Spec.Group, "version": v.Name, "kind": crd.Spec.Names.Kind},
			})
			models[crd.Spec.Group+"."+v.Name+"."+crd.Spec.Names.Kind] = s
		}
	}
	tc, err := managedfields.NewTypeConverter(models, false)
	require.NoError(t, err)
	return tc
}

func newMonReconciler(t *testing.T, p providerAdapter, objs ...client.Object) *ProviderReconciler {
	t.Helper()
	return newMonReconcilerWith(t, p, interceptor.Funcs{}, objs...)
}

// newMonReconcilerWith is newMonReconciler with every client call going
// through funcs.
func newMonReconcilerWith(t *testing.T, p providerAdapter, funcs interceptor.Funcs, objs ...client.Object) *ProviderReconciler {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, v1alpha1.AddToScheme(s))
	require.NoError(t, monitoringv1alpha1.AddToScheme(s))
	kubeSystem := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: types.UID(monTestClusterID)}}
	c := interceptor.NewClient(fake.NewClientBuilder().
		WithScheme(s).
		WithTypeConverters(monTypeConverter(t), managedfields.NewDeducedTypeConverter()).
		WithObjects(append(objs, kubeSystem)...).
		WithStatusSubresource(&v1alpha1.Instance{}, &monitoringv1alpha1.MonitoringBinding{}).
		WithIndex(&monitoringv1alpha1.MonitoringDestination{}, indexDestinationCredentialsSecret, destinationCredentialsSecret).
		Build(), funcs,
	)
	return &ProviderReconciler{provider: p, Client: c, apiReader: c}
}

// credentialSecret returns a Secret of "orders", controlled by the Instance
// when controlled is true and only owned by it otherwise.
func credentialSecret(name string, controlled bool) *corev1.Secret {
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: "team-a",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "core.openeverest.io/v1alpha1", Kind: "Instance", Name: "orders", UID: monTestInstanceUID, Controller: new(controlled),
		}},
	}}
}

func monProviderObject() *v1alpha1.Provider {
	return &v1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: "test-provider"},
		Spec: v1alpha1.ProviderSpec{
			Components: map[string]v1alpha1.Component{
				"engine":      {Type: "mysql"},
				"proxy":       {Type: "haproxy"},
				"router":      {Type: "mysqlrouter"},
				"backupAgent": {Type: "xtrabackup"},
			},
			Monitoring: &v1alpha1.ProviderMonitoring{
				ComponentTypes: map[string]v1alpha1.MonitoringComponentType{
					"mysql":       {Kind: "mysql", Metrics: v1alpha1.MetricsSourceExporter, CredentialProfiles: []string{"metrics"}},
					"haproxy":     {Kind: "haproxy", Metrics: v1alpha1.MetricsSourceNative},
					"mysqlrouter": {Kind: "mysqlrouter", Metrics: v1alpha1.MetricsSourceNone},
				},
			},
		},
	}
}

// monInstance returns the Instance "orders" with one destination entry per
// name, each pointing at the destination "<name>-dest".
func monInstance(entries ...string) *v1alpha1.Instance {
	in := readyTestInstance(func(in *v1alpha1.Instance) {
		in.UID = monTestInstanceUID
		in.Spec.Components = map[string]v1alpha1.ComponentSpec{"engine": {Type: "mysql"}, "proxy": {}}
	})
	if len(entries) > 0 {
		in.Spec.Monitoring = &v1alpha1.InstanceMonitoringSpec{}
	}
	for _, e := range entries {
		in.Spec.Monitoring.Destinations = append(in.Spec.Monitoring.Destinations, v1alpha1.InstanceMonitoringDestination{
			Name:           e,
			DestinationRef: common.MonitoringDestinationRef{Kind: "MonitoringDestination", Name: e + "-dest"},
		})
	}
	return in
}

// monBinding returns the binding core creates for entry of "orders", bound
// to the class named like the entry.
func monBinding(entry string, mode v1alpha1.MonitoringExecutionMode, conditions ...metav1.Condition) *monitoringv1alpha1.MonitoringBinding {
	return &monitoringv1alpha1.MonitoringBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       monitoring.BindingName("orders", entry),
			Namespace:  "team-a",
			Generation: 2,
			Labels:     monitoring.BindingLabels("orders", entry, entry+"-dest", entry),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "core.openeverest.io/v1alpha1", Kind: "Instance", Name: "orders", UID: monTestInstanceUID, Controller: new(true),
			}},
		},
		Spec: monitoringv1alpha1.MonitoringBindingSpec{
			InstanceRef:    common.ObjectRef{Name: "orders"},
			DestinationRef: common.MonitoringDestinationRef{Kind: "MonitoringDestination", Name: entry + "-dest"},
			ClassRef:       &common.ObjectRef{Name: entry},
			ExecutionMode:  mode,
		},
		Status: monitoringv1alpha1.MonitoringBindingStatus{Conditions: conditions},
	}
}

func pmmDestination() *monitoringv1alpha1.MonitoringDestination {
	return &monitoringv1alpha1.MonitoringDestination{
		ObjectMeta: metav1.ObjectMeta{Name: "pmm-dest", Namespace: "team-a"},
		Spec:       monitoringv1alpha1.MonitoringDestinationSpec{ClassRef: common.ObjectRef{Name: "pmm"}},
	}
}

func pmmClass() *monitoringv1alpha1.MonitoringClass {
	return &monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pmm"},
		Spec: monitoringv1alpha1.MonitoringClassSpec{
			ControllerName:  "openeverest.io/monitoring-pmm",
			ProviderManaged: &monitoringv1alpha1.ProviderManagedFulfilment{Integration: "pmm"},
		},
	}
}

func monCondition(condType string, status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type: condType, Status: status, Reason: reason, Message: message,
		ObservedGeneration: 2, LastTransitionTime: metav1.Now(),
	}
}

func accepted() metav1.Condition {
	return monCondition(monitoringv1alpha1.MonitoringBindingConditionAccepted, metav1.ConditionTrue, monitoringv1alpha1.ReasonAccepted, "")
}

// reconcileMonitoringOf runs reconcileMonitoring as Reconcile does after a
// successful Sync and returns the Instance status it produced.
func reconcileMonitoringOf(t *testing.T, r *ProviderReconciler, in *v1alpha1.Instance, stage func(*controller.Context)) *v1alpha1.Instance {
	t.Helper()
	syncCtx := controller.NewContext(t.Context(), r.Client, in.DeepCopy(), r.provider.Name())
	if stage != nil {
		stage(syncCtx)
	}
	r.reconcileMonitoring(t.Context(), syncCtx, in)
	return in
}

func storedConfigured(t *testing.T, r *ProviderReconciler, entry string) *metav1.Condition {
	t.Helper()
	b := &monitoringv1alpha1.MonitoringBinding{}
	require.NoError(t, r.Get(t.Context(), client.ObjectKey{Namespace: "team-a", Name: monitoring.BindingName("orders", entry)}, b))
	return meta.FindStatusCondition(b.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionConfigured)
}

func TestReconcileMonitoring_ProviderWithoutContract(t *testing.T) {
	t.Parallel()

	t.Run("every entry is unsupported", func(t *testing.T) {
		t.Parallel()
		in := monInstance("pmm", "prometheus")
		r := newMonReconciler(t, &readyTestProvider{}, monProviderObject(), in)

		got := reconcileMonitoringOf(t, r, in, nil)

		require.NotNil(t, got.Status.Monitoring)
		assert.Nil(t, got.Status.Monitoring.Sources)
		msg := "provider test-provider does not implement the monitoring contract"
		assert.Equal(t, []v1alpha1.InstanceMonitoringDestinationStatus{
			{Name: "pmm", Configured: metav1.ConditionFalse, Reason: "IntegrationUnsupported", Message: msg},
			{Name: "prometheus", Configured: metav1.ConditionFalse, Reason: "IntegrationUnsupported", Message: msg},
		}, got.Status.Monitoring.Destinations,
		)
		c := meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionMonitoringConfigured)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionFalse, c.Status)
		assert.Equal(t, v1alpha1.ReasonMonitoringIntegrationUnsupported, c.Reason)
		assert.Equal(t, "pmm: IntegrationUnsupported; prometheus: IntegrationUnsupported", c.Message)
	})

	t.Run("no entries clears the status", func(t *testing.T) {
		t.Parallel()
		in := monInstance()
		in.Status.Monitoring = &v1alpha1.InstanceMonitoringStatus{Destinations: []v1alpha1.InstanceMonitoringDestinationStatus{{Name: "pmm"}}}
		in.Status.Conditions = []metav1.Condition{{Type: v1alpha1.ConditionMonitoringConfigured, Status: metav1.ConditionFalse}}
		r := newMonReconciler(t, &readyTestProvider{}, monProviderObject(), in)

		got := reconcileMonitoringOf(t, r, in, nil)

		assert.Nil(t, got.Status.Monitoring)
		assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionMonitoringConfigured))
	})
}

func TestReconcileMonitoring_NoEntries(t *testing.T) {
	t.Parallel()

	engine := v1alpha1.MetricsEndpoint{
		Component: "engine", Kind: "mysql", Port: intstr.FromInt32(9104),
		PodSelector: map[string]string{controller.InstanceLabel: "orders"},
	}
	tests := []struct {
		name    string
		sources *v1alpha1.MonitoringSources
		want    *v1alpha1.InstanceMonitoringStatus
	}{
		{
			name: "no sources",
		},
		{
			name:    "sources only",
			sources: &v1alpha1.MonitoringSources{Metrics: []v1alpha1.MetricsEndpoint{engine}},
			want: &v1alpha1.InstanceMonitoringStatus{Sources: &v1alpha1.MonitoringSources{
				Identity: v1alpha1.MonitoringIdentity{ClusterID: monTestClusterID, Labels: map[string]string{
					"openeverest_instance": "orders", "openeverest_namespace": "team-a",
					"openeverest_provider": "test-provider", "k8s_cluster_id": monTestClusterID,
				}},
				Metrics: []v1alpha1.MetricsEndpoint{engine},
			}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := monInstance()
			in.Status.Conditions = []metav1.Condition{{Type: v1alpha1.ConditionMonitoringConfigured, Status: metav1.ConditionTrue}}
			r := newMonReconciler(t, &monProvider{sources: tc.sources}, monProviderObject(), in)

			got := reconcileMonitoringOf(t, r, in, nil)

			assert.Equal(t, tc.want, got.Status.Monitoring)
			assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionMonitoringConfigured))
		})
	}
}

func TestReconcileMonitoring_SourcesNotes(t *testing.T) {
	t.Parallel()

	engine := v1alpha1.MetricsEndpoint{
		Component: "engine", Kind: "mysql", Port: intstr.FromInt32(9104),
		PodSelector: map[string]string{controller.InstanceLabel: "orders"},
	}
	unlabelled := *engine.DeepCopy()
	unlabelled.PodSelector = map[string]string{controller.ComponentLabel: "engine"}
	published := &v1alpha1.MonitoringSources{
		Identity: v1alpha1.MonitoringIdentity{ClusterID: monTestClusterID},
		Metrics:  []v1alpha1.MetricsEndpoint{engine},
	}
	forbidden := apierrors.NewForbidden(schema.GroupResource{Resource: "namespaces"}, "kube-system", errors.New("RBAC: access denied"))
	tests := []struct {
		name        string
		provider    *monProvider
		funcs       interceptor.Funcs
		wantMetrics []v1alpha1.MetricsEndpoint
		wantMessage string
	}{
		{
			name:        "MonitoringSources fails",
			provider:    &monProvider{sourcesErr: errors.New("engine CR not readable")},
			wantMetrics: published.Metrics,
			wantMessage: "All monitoring destinations are configured; monitoring sources not updated: engine CR not readable",
		},
		{
			name:     "the cluster ID cannot be read",
			provider: &monProvider{sources: &v1alpha1.MonitoringSources{Metrics: []v1alpha1.MetricsEndpoint{engine, engine}}},
			funcs: interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if _, ok := obj.(*corev1.Namespace); ok {
						return forbidden
					}
					return c.Get(ctx, key, obj, opts...)
				},
			},
			wantMetrics: published.Metrics,
			wantMessage: "All monitoring destinations are configured; monitoring sources not updated: get namespace kube-system: " + forbidden.Error(),
		},
		{
			name:        "endpoints dropped by validation",
			provider:    &monProvider{sources: &v1alpha1.MonitoringSources{Metrics: []v1alpha1.MetricsEndpoint{engine, unlabelled}}},
			wantMetrics: []v1alpha1.MetricsEndpoint{engine},
			wantMessage: "All monitoring destinations are configured; 1 of 2 metrics endpoints dropped",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := monInstance("pmm")
			in.Status.Monitoring = &v1alpha1.InstanceMonitoringStatus{Sources: published.DeepCopy()}
			r := newMonReconcilerWith(t, tc.provider, tc.funcs, monProviderObject(), in, pmmDestination(), pmmClass(),
				monBinding("pmm", v1alpha1.MonitoringExecutionModeProviderManaged, accepted()),
			)

			got := reconcileMonitoringOf(t, r, in, nil)

			require.NotNil(t, got.Status.Monitoring)
			require.NotNil(t, got.Status.Monitoring.Sources)
			assert.Equal(t, tc.wantMetrics, got.Status.Monitoring.Sources.Metrics)
			c := meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionMonitoringConfigured)
			require.NotNil(t, c)
			assert.Equal(t, metav1.ConditionTrue, c.Status)
			assert.Equal(t, tc.wantMessage, c.Message)
		})
	}
}

func TestReconcileMonitoring_BindingsUnreadable(t *testing.T) {
	t.Parallel()
	in := monInstance("pmm")
	r := newMonReconcilerWith(t, &monProvider{}, interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if _, ok := list.(*monitoringv1alpha1.MonitoringBindingList); ok {
				return errors.New("injected")
			}
			return c.List(ctx, list, opts...)
		},
	}, monProviderObject(), in,
	)

	got := reconcileMonitoringOf(t, r, in, nil)

	c := meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionMonitoringConfigured)
	require.NotNil(t, c)
	assert.Equal(t, metav1.ConditionUnknown, c.Status)
	assert.Equal(t, v1alpha1.ReasonMonitoringPending, c.Reason)
	assert.Equal(t, `list monitoring bindings for instance "orders": injected`, c.Message)
}

func TestReconcileMonitoring_ConfiguredWriteFails(t *testing.T) {
	t.Parallel()
	in := monInstance("pmm")
	r := newMonReconcilerWith(t, &monProvider{}, interceptor.Funcs{
		SubResourceApply: func(context.Context, client.Client, string, runtime.ApplyConfiguration, ...client.SubResourceApplyOption) error {
			return errors.New("injected")
		},
	}, monProviderObject(), in, pmmDestination(), pmmClass(),
		monBinding("pmm", v1alpha1.MonitoringExecutionModeProviderManaged, accepted()),
	)

	got := reconcileMonitoringOf(t, r, in, nil)

	require.NotNil(t, got.Status.Monitoring)
	assert.Equal(t, []v1alpha1.InstanceMonitoringDestinationStatus{{
		Name: "pmm", Mode: v1alpha1.MonitoringExecutionModeProviderManaged, Configured: metav1.ConditionUnknown, Reason: "Pending",
		Message: "failed to write the Configured condition: apply status of MonitoringBinding team-a/" +
			monitoring.BindingName("orders", "pmm") + ": injected",
	}}, got.Status.Monitoring.Destinations,
	)
	c := meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionMonitoringConfigured)
	require.NotNil(t, c)
	assert.Equal(t, metav1.ConditionUnknown, c.Status)
	assert.Equal(t, "pmm: Pending", c.Message)
}

func TestReconcileMonitoring_RollUp(t *testing.T) {
	t.Parallel()

	const (
		pm = v1alpha1.MonitoringExecutionModeProviderManaged
		em = v1alpha1.MonitoringExecutionModeExtensionManaged
	)
	configured := func(status metav1.ConditionStatus, reason string) metav1.Condition {
		return monCondition(monitoringv1alpha1.MonitoringBindingConditionConfigured, status, reason, "")
	}
	tests := []struct {
		name        string
		entries     []string
		bindings    []client.Object
		wantStatus  metav1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{
			name:    "every destination configured",
			entries: []string{"pmm", "prometheus"},
			bindings: []client.Object{
				monBinding("pmm", pm, accepted()),
				monBinding("prometheus", em, accepted(), configured(metav1.ConditionTrue, "Configured")),
			},
			wantStatus:  metav1.ConditionTrue,
			wantReason:  v1alpha1.ReasonMonitoringConfigured,
			wantMessage: "All monitoring destinations are configured",
		},
		{
			name:    "a class controller has not reported yet",
			entries: []string{"pmm", "prometheus"},
			bindings: []client.Object{
				monBinding("pmm", pm, accepted()),
				monBinding("prometheus", em, accepted(), configured(metav1.ConditionUnknown, "Pending")),
			},
			wantStatus:  metav1.ConditionUnknown,
			wantReason:  v1alpha1.ReasonMonitoringPending,
			wantMessage: "prometheus: Pending",
		},
		{
			name:        "a binding not created yet",
			entries:     []string{"pmm", "prometheus"},
			bindings:    []client.Object{monBinding("pmm", pm, accepted())},
			wantStatus:  metav1.ConditionUnknown,
			wantReason:  v1alpha1.ReasonMonitoringPending,
			wantMessage: "prometheus: Pending",
		},
		{
			name:    "a failure outweighs a pending entry",
			entries: []string{"pmm", "prometheus"},
			bindings: []client.Object{monBinding("prometheus", em, monCondition(
				monitoringv1alpha1.MonitoringBindingConditionAccepted, metav1.ConditionFalse, monitoringv1alpha1.ReasonClassNotAccepted, "class prometheus is not claimed",
			))},
			wantStatus:  metav1.ConditionFalse,
			wantReason:  v1alpha1.ReasonMonitoringNotConfigured,
			wantMessage: "pmm: Pending; prometheus: ClassNotAccepted",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := monInstance(tc.entries...)
			r := newMonReconciler(t, &monProvider{}, append(tc.bindings, monProviderObject(), in, pmmDestination(), pmmClass())...)

			got := reconcileMonitoringOf(t, r, in, nil)

			c := meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionMonitoringConfigured)
			require.NotNil(t, c)
			assert.Equal(t, tc.wantStatus, c.Status)
			assert.Equal(t, tc.wantReason, c.Reason)
			assert.Equal(t, tc.wantMessage, c.Message)
			require.NotNil(t, got.Status.Monitoring)
			assert.Len(t, got.Status.Monitoring.Destinations, len(tc.entries))
		})
	}
}

func TestReconcileMonitoring_DestinationStatus(t *testing.T) {
	t.Parallel()

	const (
		pm = v1alpha1.MonitoringExecutionModeProviderManaged
		em = v1alpha1.MonitoringExecutionModeExtensionManaged
	)
	notAccepted := monCondition(monitoringv1alpha1.MonitoringBindingConditionAccepted, metav1.ConditionFalse,
		monitoringv1alpha1.ReasonDestinationNotFound, "MonitoringDestination pmm-dest not found",
	)
	staleAccepted := accepted()
	staleAccepted.ObservedGeneration = 1
	configured := func(status metav1.ConditionStatus, reason, message string) metav1.Condition {
		return monCondition(monitoringv1alpha1.MonitoringBindingConditionConfigured, status, reason, message)
	}
	reportedForGeneration2 := func() *monitoringv1alpha1.MonitoringBinding {
		b := monBinding("pmm", em, configured(metav1.ConditionTrue, "Configured", "PodMonitor orders-pmm created"))
		b.Generation = 3
		current := accepted()
		current.ObservedGeneration = 3
		b.Status.Conditions = append(b.Status.Conditions, current)
		return b
	}
	type want struct {
		status  metav1.ConditionStatus
		reason  string
		message string
	}
	tests := []struct {
		name               string
		binding            *monitoringv1alpha1.MonitoringBinding
		provider           providerAdapter
		destinationDeleted bool
		classDeleted       bool
		stage              func(*controller.Context)
		// entry is the Instance's view; configured is the binding's
		// Configured condition afterwards, nil when it must be absent.
		entry      want
		configured *want
	}{
		{
			name:    "not accepted yet",
			binding: monBinding("pmm", pm),
			entry:   want{metav1.ConditionUnknown, "Pending", "waiting for the binding to be accepted"},
		},
		{
			name:    "accepted for an older generation",
			binding: monBinding("pmm", pm, staleAccepted),
			entry:   want{metav1.ConditionUnknown, "Pending", "waiting for the binding to be accepted"},
		},
		{
			name:       "ProviderManaged not accepted, nothing rendered before",
			binding:    monBinding("pmm", pm, notAccepted),
			entry:      want{metav1.ConditionFalse, "DestinationNotFound", "MonitoringDestination pmm-dest not found"},
			configured: &want{metav1.ConditionFalse, "NotAccepted", "Binding is not Accepted (DestinationNotFound); nothing is rendered"},
		},
		{
			name:       "ProviderManaged not accepted after being configured",
			binding:    monBinding("pmm", pm, notAccepted, configured(metav1.ConditionTrue, "Configured", "")),
			entry:      want{metav1.ConditionFalse, "DestinationNotFound", "MonitoringDestination pmm-dest not found"},
			configured: &want{metav1.ConditionUnknown, "Retained", "Binding is no longer Accepted (DestinationNotFound); keeping the live rendering"},
		},
		{
			name:       "ProviderManaged still not accepted while retained",
			binding:    monBinding("pmm", pm, notAccepted, configured(metav1.ConditionUnknown, "Retained", "")),
			entry:      want{metav1.ConditionFalse, "DestinationNotFound", "MonitoringDestination pmm-dest not found"},
			configured: &want{metav1.ConditionUnknown, "Retained", "Binding is no longer Accepted (DestinationNotFound); keeping the live rendering"},
		},
		{
			name:       "ProviderManaged rendered by Sync",
			binding:    monBinding("pmm", pm, accepted()),
			entry:      want{metav1.ConditionTrue, "Configured", "Rendered by provider test-provider"},
			configured: &want{metav1.ConditionTrue, "Configured", "Rendered by provider test-provider"},
		},
		{
			name:    "ProviderManaged with a staged failure",
			binding: monBinding("pmm", pm, accepted()),
			stage: func(c *controller.Context) {
				c.SetMonitoringBindingResult(monitoring.BindingName("orders", "pmm"), monitoringv1alpha1.ReasonRenderFailed, "destination has no url")
			},
			entry:      want{metav1.ConditionFalse, "RenderFailed", "destination has no url"},
			configured: &want{metav1.ConditionFalse, "RenderFailed", "destination has no url"},
		},
		{
			name:               "ProviderManaged still accepted after its destination was deleted",
			binding:            monBinding("pmm", pm, accepted(), configured(metav1.ConditionTrue, "Configured", "Rendered by provider test-provider")),
			destinationDeleted: true,
			entry:              want{metav1.ConditionUnknown, "Pending", "waiting for the destination and class to be resolved"},
			configured:         &want{metav1.ConditionTrue, "Configured", "Rendered by provider test-provider"},
		},
		{
			name:         "ProviderManaged still accepted after its class was deleted",
			binding:      monBinding("pmm", pm, accepted(), configured(metav1.ConditionTrue, "Configured", "Rendered by provider test-provider")),
			classDeleted: true,
			entry:        want{metav1.ConditionUnknown, "Pending", "waiting for the destination and class to be resolved"},
			configured:   &want{metav1.ConditionTrue, "Configured", "Rendered by provider test-provider"},
		},
		{
			name:       "ProviderManaged integration the provider does not render",
			binding:    monBinding("pmm", pm, accepted()),
			provider:   &vmagentProvider{},
			entry:      want{metav1.ConditionFalse, "RenderFailed", "provider test-provider does not render integration pmm"},
			configured: &want{metav1.ConditionFalse, "RenderFailed", "provider test-provider does not render integration pmm"},
		},
		{
			name:     "ProviderManaged with a provider that renders no integration",
			binding:  monBinding("pmm", pm, accepted()),
			provider: &sourcesOnlyProvider{},
			stage: func(c *controller.Context) {
				c.SetMonitoringBindingResult(monitoring.BindingName("orders", "pmm"), monitoringv1alpha1.ReasonConfigured, "rendered")
			},
			entry:      want{metav1.ConditionFalse, "RenderFailed", "provider test-provider does not render integration pmm"},
			configured: &want{metav1.ConditionFalse, "RenderFailed", "provider test-provider does not render integration pmm"},
		},
		{
			name:       "ExtensionManaged configured by its class controller",
			binding:    monBinding("pmm", em, accepted(), configured(metav1.ConditionTrue, "Configured", "PodMonitor orders-pmm created")),
			entry:      want{metav1.ConditionTrue, "Configured", "PodMonitor orders-pmm created"},
			configured: &want{metav1.ConditionTrue, "Configured", "PodMonitor orders-pmm created"},
		},
		{
			name:       "ExtensionManaged reported for an older generation",
			binding:    reportedForGeneration2(),
			entry:      want{metav1.ConditionUnknown, "Pending", "waiting for the class controller"},
			configured: &want{metav1.ConditionTrue, "Configured", "PodMonitor orders-pmm created"},
		},
		{
			name:    "ExtensionManaged without a report from its class controller",
			binding: monBinding("pmm", em, accepted()),
			entry:   want{metav1.ConditionUnknown, "Pending", "waiting for the class controller"},
		},
		{
			name:    "ExtensionManaged not accepted",
			binding: monBinding("pmm", em, notAccepted),
			entry:   want{metav1.ConditionFalse, "DestinationNotFound", "MonitoringDestination pmm-dest not found"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := monInstance("pmm")
			objs := []client.Object{tc.binding, monProviderObject(), in}
			if !tc.destinationDeleted {
				objs = append(objs, pmmDestination())
			}
			if !tc.classDeleted {
				objs = append(objs, pmmClass())
			}
			p := tc.provider
			if p == nil {
				p = &monProvider{}
			}
			r := newMonReconciler(t, p, objs...)

			got := reconcileMonitoringOf(t, r, in, tc.stage)

			require.NotNil(t, got.Status.Monitoring)
			assert.Equal(t, []v1alpha1.InstanceMonitoringDestinationStatus{{
				Name: "pmm", Mode: tc.binding.Spec.ExecutionMode,
				Configured: tc.entry.status, Reason: tc.entry.reason, Message: tc.entry.message,
			}}, got.Status.Monitoring.Destinations,
			)

			c := storedConfigured(t, r, "pmm")
			if tc.configured == nil {
				assert.Nil(t, c)
				return
			}
			require.NotNil(t, c)
			gotConfigured := want{c.Status, c.Reason, c.Message}
			assert.Equal(t, *tc.configured, gotConfigured)
			assert.Equal(t, int64(2), c.ObservedGeneration)
			stored := &monitoringv1alpha1.MonitoringBinding{}
			require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(tc.binding), stored))
			before := meta.FindStatusCondition(tc.binding.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted)
			after := meta.FindStatusCondition(stored.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted)
			require.NotNil(t, after, "core's Accepted condition is left alone")
			assert.Equal(t, want{before.Status, before.Reason, before.Message}, want{after.Status, after.Reason, after.Message})
		})
	}
}

func TestValidateSources(t *testing.T) {
	t.Parallel()

	selector := map[string]string{controller.InstanceLabel: "orders", controller.ComponentLabel: "engine"}
	valid := v1alpha1.MetricsEndpoint{
		Component:   "engine",
		Kind:        "mysql",
		PodSelector: selector,
		Port:        intstr.FromString("metrics"),
		Path:        "/probe",
		Scheme:      "https",
		TLS: &v1alpha1.MetricsEndpointTLS{
			CA:         &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "orders-ca"}, Key: "ca.crt"},
			ServerName: "orders-engine",
		},
		Auth:   &v1alpha1.MetricsEndpointAuth{CredentialProfile: "metrics"},
		Params: map[string][]string{"target": {"mysql://127.0.0.1:3306"}},
	}
	proxy := v1alpha1.MetricsEndpoint{
		Component: "proxy", Kind: "haproxy", Port: intstr.FromInt32(8404),
		PodSelector: map[string]string{controller.InstanceLabel: "orders"},
	}
	credentials := []v1alpha1.MonitoringCredential{{Profile: "metrics", SecretRef: common.SecretRef{Name: "orders-monitoring-metrics"}}}
	credentialIn := func(secret string) []v1alpha1.MonitoringCredential {
		return []v1alpha1.MonitoringCredential{{Profile: "metrics", SecretRef: common.SecretRef{Name: secret}}}
	}
	with := func(mutate func(*v1alpha1.MetricsEndpoint)) v1alpha1.MetricsEndpoint {
		ep := *valid.DeepCopy()
		mutate(&ep)
		return ep
	}
	tests := []struct {
		name            string
		credentials     []v1alpha1.MonitoringCredential
		metrics         []v1alpha1.MetricsEndpoint
		want            []v1alpha1.MetricsEndpoint
		wantCredentials []v1alpha1.MonitoringCredential
	}{
		{
			name:            "valid endpoints are kept unchanged",
			credentials:     credentials,
			metrics:         []v1alpha1.MetricsEndpoint{valid, proxy},
			want:            []v1alpha1.MetricsEndpoint{valid, proxy},
			wantCredentials: credentials,
		},
		{
			name:            "kind comes from the declared component type",
			credentials:     credentials,
			metrics:         []v1alpha1.MetricsEndpoint{with(func(ep *v1alpha1.MetricsEndpoint) { ep.Kind = "postgresql" })},
			want:            []v1alpha1.MetricsEndpoint{valid},
			wantCredentials: credentials,
		},
		{
			name:        "undeclared component type",
			credentials: credentials,
			metrics: []v1alpha1.MetricsEndpoint{valid, with(func(ep *v1alpha1.MetricsEndpoint) {
				ep.Component, ep.Kind = "backupAgent", "xtrabackup"
			})},
			want:            []v1alpha1.MetricsEndpoint{valid},
			wantCredentials: credentials,
		},
		{
			name:        "component type declared without metrics",
			credentials: credentials,
			metrics: []v1alpha1.MetricsEndpoint{valid, with(func(ep *v1alpha1.MetricsEndpoint) {
				ep.Component, ep.Kind, ep.Auth = "router", "mysqlrouter", nil
			})},
			want:            []v1alpha1.MetricsEndpoint{valid},
			wantCredentials: credentials,
		},
		{
			name:            "component unknown to Instance and Provider",
			credentials:     credentials,
			metrics:         []v1alpha1.MetricsEndpoint{with(func(ep *v1alpha1.MetricsEndpoint) { ep.Component = "sidecar" })},
			wantCredentials: credentials,
		},
		{
			name:        "pod selector without the Instance label",
			credentials: credentials,
			metrics: []v1alpha1.MetricsEndpoint{valid, with(func(ep *v1alpha1.MetricsEndpoint) {
				ep.PodSelector = map[string]string{controller.ComponentLabel: "engine"}
			})},
			want:            []v1alpha1.MetricsEndpoint{valid},
			wantCredentials: credentials,
		},
		{
			name:        "pod selector naming another Instance",
			credentials: credentials,
			metrics: []v1alpha1.MetricsEndpoint{with(func(ep *v1alpha1.MetricsEndpoint) {
				ep.PodSelector = map[string]string{controller.InstanceLabel: "payments"}
			})},
			wantCredentials: credentials,
		},
		{
			name:            "scheme other than http or https",
			credentials:     credentials,
			metrics:         []v1alpha1.MetricsEndpoint{valid, with(func(ep *v1alpha1.MetricsEndpoint) { ep.Scheme = "ftp" })},
			want:            []v1alpha1.MetricsEndpoint{valid},
			wantCredentials: credentials,
		},
		{
			name:    "credential profile not published",
			metrics: []v1alpha1.MetricsEndpoint{valid, proxy},
			want:    []v1alpha1.MetricsEndpoint{proxy},
		},
		{
			name:            "duplicate credential profile",
			credentials:     append(credentialIn("orders-monitoring-metrics"), credentialIn("orders-monitoring-other")...),
			metrics:         []v1alpha1.MetricsEndpoint{valid},
			want:            []v1alpha1.MetricsEndpoint{valid},
			wantCredentials: credentials,
		},
		{
			name:        "credential Secret only owned, not controlled, by the Instance",
			credentials: credentialIn("orders-shared"),
			metrics:     []v1alpha1.MetricsEndpoint{valid, proxy},
			want:        []v1alpha1.MetricsEndpoint{proxy},
		},
		{
			name:        "credential Secret missing",
			credentials: credentialIn("orders-missing"),
			metrics:     []v1alpha1.MetricsEndpoint{valid, proxy},
			want:        []v1alpha1.MetricsEndpoint{proxy},
		},
		{
			name:    "more endpoints than the status holds",
			metrics: slices.Repeat([]v1alpha1.MetricsEndpoint{proxy}, 33),
			want:    slices.Repeat([]v1alpha1.MetricsEndpoint{proxy}, 32),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := monInstance()
			r := newMonReconciler(t, &monProvider{}, monProviderObject(), in,
				credentialSecret("orders-monitoring-metrics", true),
				credentialSecret("orders-monitoring-other", true),
				credentialSecret("orders-shared", false),
			)
			syncCtx := controller.NewContext(t.Context(), r.Client, in, r.provider.Name())
			computed := &v1alpha1.MonitoringSources{
				Identity: v1alpha1.MonitoringIdentity{
					ClusterID: "provider-supplied",
					Labels:    map[string]string{"openeverest_instance": "spoofed", "team": "dba"},
				},
				Credentials: tc.credentials,
				Metrics:     tc.metrics,
			}

			got, err := r.validateSources(t.Context(), syncCtx, computed, logr.Discard())
			require.NoError(t, err)

			assert.Equal(t, tc.want, got.Metrics)
			assert.Equal(t, tc.wantCredentials, got.Credentials)
			assert.Equal(t, v1alpha1.MonitoringIdentity{
				ClusterID: monTestClusterID,
				Labels: map[string]string{
					"openeverest_instance":  "orders",
					"openeverest_namespace": "team-a",
					"openeverest_provider":  "test-provider",
					"k8s_cluster_id":        monTestClusterID,
				},
			}, got.Identity,
			)
		})
	}
}

func TestReconcile_PublishesMonitoringStatus(t *testing.T) {
	t.Parallel()
	engine := v1alpha1.MetricsEndpoint{
		Component: "engine", Kind: "mysql", Port: intstr.FromInt32(9104),
		PodSelector: map[string]string{controller.InstanceLabel: "orders"},
	}
	p := &monProvider{
		readyTestProvider: readyTestProvider{status: controller.Ready()},
		sources:           &v1alpha1.MonitoringSources{Metrics: []v1alpha1.MetricsEndpoint{engine}},
	}
	in := monInstance("pmm")
	r := newMonReconciler(t, p, monProviderObject(), in, pmmDestination(), pmmClass(),
		monBinding("pmm", v1alpha1.MonitoringExecutionModeProviderManaged, accepted()),
	)

	got, err := reconcileAndGet(t, r)
	require.NoError(t, err)

	require.NotNil(t, got.Status.Monitoring)
	require.NotNil(t, got.Status.Monitoring.Sources)
	assert.Equal(t, []v1alpha1.MetricsEndpoint{engine}, got.Status.Monitoring.Sources.Metrics)
	assert.Equal(t, monTestClusterID, got.Status.Monitoring.Sources.Identity.ClusterID)
	assert.Equal(t, []v1alpha1.InstanceMonitoringDestinationStatus{{
		Name: "pmm", Mode: v1alpha1.MonitoringExecutionModeProviderManaged, Configured: metav1.ConditionTrue,
		Reason: "Configured", Message: "Rendered by provider test-provider",
	}}, got.Status.Monitoring.Destinations,
	)
	assert.True(t, meta.IsStatusConditionTrue(got.Status.Conditions, v1alpha1.ConditionMonitoringConfigured))
	c := storedConfigured(t, r, "pmm")
	require.NotNil(t, c)
	assert.Equal(t, metav1.ConditionTrue, c.Status)
}

func TestReconcile_RequiredSourcesIncludeRenderedIntegrations(t *testing.T) {
	t.Parallel()
	p := &requiredSourcesProvider{monProvider: monProvider{readyTestProvider: readyTestProvider{status: controller.Ready()}}}
	in := monInstance("pmm")
	r := newMonReconciler(t, p, monProviderObject(), in, pmmDestination(), pmmClass(),
		monBinding("pmm", v1alpha1.MonitoringExecutionModeProviderManaged, accepted()),
	)

	_, err := reconcileAndGet(t, r)
	require.NoError(t, err)

	require.NoError(t, p.err)
	assert.Equal(t, controller.RequiredSources{
		Metrics:      controller.NotRequired,
		Integrations: map[string]controller.Requirement{"pmm": controller.Required},
	}, p.got, "pmm comes from MonitoringIntegrations: the Provider object declares no integration",
	)
}

func TestMonitoringWatchMapping(t *testing.T) {
	t.Parallel()

	orders := monInstance("pmm")
	payments := monInstance("pmm")
	payments.Name, payments.Spec.ProviderRef.Name = "payments", "other-provider"
	billing := monInstance()
	billing.Name = "billing"
	ordersBinding := monBinding("pmm", v1alpha1.MonitoringExecutionModeProviderManaged)
	paymentsBinding := monBinding("pmm", v1alpha1.MonitoringExecutionModeProviderManaged)
	paymentsBinding.Name, paymentsBinding.Labels[monitoringv1alpha1.BindingInstanceLabel] = "payments-pmm-00000000", "payments"
	paymentsBinding.Spec.InstanceRef.Name = "payments"
	r := newMonReconciler(t, &monProvider{}, orders, payments, billing, ordersBinding, paymentsBinding)

	ordersOnly := []reconcile.Request{{NamespacedName: client.ObjectKeyFromObject(orders)}}
	assert.Equal(t, ordersOnly, r.instancesForBinding(t.Context(), ordersBinding))
	assert.Empty(t, r.instancesForBinding(t.Context(), paymentsBinding), "another provider's Instance")
	assert.Equal(t, ordersOnly, r.instancesForDestination(t.Context(), &monitoringv1alpha1.MonitoringDestination{
		ObjectMeta: metav1.ObjectMeta{Name: "pmm-dest", Namespace: "team-a"},
	}))
	assert.Equal(t, ordersOnly, r.instancesForClass(t.Context(), &monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pmm"},
	}))
	assert.Empty(t, r.instancesForClass(t.Context(), &monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{Name: "prometheus"},
	}))
}

func TestMonitoringCredentialsSecretWatch(t *testing.T) {
	t.Parallel()

	orders := monInstance("pmm")
	billing := monInstance("vm")
	billing.Name = "billing"
	payments := monInstance("pmm")
	payments.Name, payments.Spec.ProviderRef.Name = "payments", "other-provider"
	bindingOf := func(instance, entry string) *monitoringv1alpha1.MonitoringBinding {
		b := monBinding(entry, v1alpha1.MonitoringExecutionModeProviderManaged)
		b.Name, b.Labels = monitoring.BindingName(instance, entry), monitoring.BindingLabels(instance, entry, entry+"-dest", entry)
		b.Spec.InstanceRef.Name = instance
		return b
	}
	destination := func(name, secret string) *monitoringv1alpha1.MonitoringDestination {
		return &monitoringv1alpha1.MonitoringDestination{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "team-a"},
			Spec: monitoringv1alpha1.MonitoringDestinationSpec{
				ClassRef:             common.ObjectRef{Name: "pmm"},
				CredentialsSecretRef: &common.SecretRef{Name: secret},
			},
		}
	}
	r := newMonReconciler(t, &monProvider{}, orders, billing, payments,
		bindingOf("orders", "pmm"), bindingOf("billing", "vm"), bindingOf("payments", "pmm"),
		destination("pmm-dest", "pmm-token"), destination("vm-dest", "vm-token"), destination("idle-dest", "idle-token"),
	)

	tests := []struct {
		name      string
		namespace string
		secret    string
		want      []string
	}{
		{name: "credentials of a destination bound by this provider and another", namespace: "team-a", secret: "pmm-token", want: []string{"orders"}},
		{name: "credentials of another bound destination", namespace: "team-a", secret: "vm-token", want: []string{"billing"}},
		{name: "credentials of a destination nothing is bound to", namespace: "team-a", secret: "idle-token"},
		{name: "Secret no destination names", namespace: "team-a", secret: "orders-users"},
		{name: "same name in another namespace", namespace: "team-b", secret: "pmm-token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			secret := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{
				Name: tt.secret, Namespace: tt.namespace, ResourceVersion: "7",
			}}
			secret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))

			var got []string
			for _, req := range r.instancesForCredentialsSecret(t.Context(), secret) {
				got = append(got, req.Name)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMonitoringDestinationWatch(t *testing.T) {
	t.Parallel()

	old := &monitoringv1alpha1.MonitoringDestination{
		ObjectMeta: metav1.ObjectMeta{Name: "pmm-dest", Namespace: "team-a", Generation: 1, ResourceVersion: "10"},
		Spec:       monitoringv1alpha1.MonitoringDestinationSpec{ClassRef: common.ObjectRef{Name: "pmm"}},
	}
	statusChanged := old.DeepCopy()
	statusChanged.ResourceVersion = "11"
	statusChanged.Status.Conditions = []metav1.Condition{{
		Type: monitoringv1alpha1.MonitoringDestinationConditionReady, Status: metav1.ConditionTrue, Reason: "Ready",
	}}
	specChanged := old.DeepCopy()
	specChanged.ResourceVersion, specChanged.Generation = "11", 2
	statusUpdate := event.UpdateEvent{ObjectOld: old, ObjectNew: statusChanged}
	specUpdate := event.UpdateEvent{ObjectOld: old, ObjectNew: specChanged}

	tests := []struct {
		name         string
		provider     providerAdapter
		wantOnStatus bool
	}{
		{name: "provider rendering integrations", provider: &monProvider{}, wantOnStatus: true},
		{name: "provider publishing sources only", provider: &sourcesOnlyProvider{}, wantOnStatus: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			unbound := monInstance("pmm")
			unbound.Name = "unbound"
			r := newMonReconciler(t, tt.provider, monInstance("pmm"), unbound,
				monBinding("pmm", v1alpha1.MonitoringExecutionModeProviderManaged),
			)

			assert.Equal(t, tt.wantOnStatus, r.destinationChanged().Update(statusUpdate), "status-only update")
			assert.True(t, r.destinationChanged().Update(specUpdate), "spec update")
			assert.Equal(t, []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "orders"}}},
				r.instancesForDestination(t.Context(), statusChanged), "only Instances with a binding to the destination",
			)
		})
	}
}
