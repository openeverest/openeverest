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
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/managedfields"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kube-openapi/pkg/validation/spec"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

const (
	testNamespace = "team-a"
	testInstance  = "orders"
	testUID       = types.UID("orders-uid")
)

func newFakeClient(t *testing.T, objs ...client.Object) client.Client { //nolint:ireturn // the fake is used through the client interface
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))
	require.NoError(t, monitoringv1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithTypeConverters(crdTypeConverter(t), managedfields.NewDeducedTypeConverter()).
		WithObjects(objs...).
		WithStatusSubresource(
			&monitoringv1alpha1.MonitoringClass{},
			&monitoringv1alpha1.MonitoringDestination{},
			&monitoringv1alpha1.MonitoringBinding{},
		).
		WithIndex(&corev1alpha1.Instance{}, instanceMonitoringDestinationField, instanceMonitoringDestinationNames).
		WithIndex(&monitoringv1alpha1.MonitoringDestination{}, destinationCredentialsSecretField, destinationCredentialsSecretName).
		WithInterceptorFuncs(generations()).
		Build()
}

// generations sets metadata.generation the way the API server does, which
// the fake client does not: 1 on create, +1 on every binding spec change.
func generations() interceptor.Funcs {
	return interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			obj.SetGeneration(1)
			return c.Create(ctx, obj, opts...)
		},
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			if b, ok := obj.(*monitoringv1alpha1.MonitoringBinding); ok {
				stored := &monitoringv1alpha1.MonitoringBinding{}
				if err := c.Get(ctx, client.ObjectKeyFromObject(b), stored); err != nil {
					return err
				}
				b.Generation = stored.Generation
				if !equality.Semantic.DeepEqual(stored.Spec, b.Spec) {
					b.Generation++
				}
			}
			return c.Update(ctx, obj, opts...)
		},
	}
}

// crdTypeConverter makes the fake client's server-side apply follow the
// generated CRD schemas (listType=map conditions and providers) instead of
// treating every list as atomic.
func crdTypeConverter(t *testing.T) managedfields.TypeConverter { //nolint:ireturn // the fake client takes the interface
	t.Helper()
	crds := os.DirFS(filepath.Join("..", "..", "..", "config", "crd", "bases"))
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

func claimed(class *monitoringv1alpha1.MonitoringClass) *monitoringv1alpha1.MonitoringClass {
	class.Status.Conditions = []metav1.Condition{{
		Type:               monitoringv1alpha1.MonitoringClassConditionAccepted,
		Status:             metav1.ConditionTrue,
		Reason:             monitoringv1alpha1.ReasonAccepted,
		LastTransitionTime: metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
	}}
	return class
}

func objectSchema(properties map[string]apiextensionsv1.JSONSchemaProps, required ...string) *common.ParametersSchema {
	return &common.ParametersSchema{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
		Type:       "object",
		Required:   required,
		Properties: properties,
	}}
}

func stringMap() apiextensionsv1.JSONSchemaProps {
	return apiextensionsv1.JSONSchemaProps{
		Type:                 "object",
		AdditionalProperties: &apiextensionsv1.JSONSchemaPropsOrBool{Allows: true, Schema: &apiextensionsv1.JSONSchemaProps{Type: "string"}},
	}
}

// pmmClass is a claimed ProviderManaged-only class, as the PMM extension
// ships it.
func pmmClass() *monitoringv1alpha1.MonitoringClass {
	return claimed(&monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pmm"},
		Spec: monitoringv1alpha1.MonitoringClassSpec{
			ControllerName: "openeverest.io/monitoring-pmm",
			ProviderManaged: &monitoringv1alpha1.ProviderManagedFulfilment{
				Integration:  "pmm",
				AgentImage:   "percona/pmm-client:3.8.0",
				AgentVersion: "3.8.0",
				Features:     []string{"queryAnalytics"},
			},
			DestinationParametersSchema: objectSchema(map[string]apiextensionsv1.JSONSchemaProps{
				"url":       {Type: "string"},
				"verifyTLS": {Type: "boolean"},
			}, "url",
			),
			InstanceParametersSchema: objectSchema(map[string]apiextensionsv1.JSONSchemaProps{
				"resources": {Type: "object", Properties: map[string]apiextensionsv1.JSONSchemaProps{
					"limits":   stringMap(),
					"requests": stringMap(),
				}},
			}),
			CredentialsSchema: &monitoringv1alpha1.CredentialsSchema{Required: []string{"apiKey"}},
		},
	})
}

// hybridPMMClass is the pmm class once it also declares an ExtensionManaged
// fallback.
func hybridPMMClass() *monitoringv1alpha1.MonitoringClass {
	class := pmmClass()
	class.Spec.ExtensionManaged = &monitoringv1alpha1.ExtensionManagedFulfilment{
		Requires: monitoringv1alpha1.SourceRequirements{Metrics: true},
	}
	return class
}

// scrapeClass is a claimed ExtensionManaged class requiring metrics, as the
// Prometheus and VictoriaMetrics extensions ship it.
func scrapeClass(name, labelsParameter string) *monitoringv1alpha1.MonitoringClass {
	return claimed(&monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: monitoringv1alpha1.MonitoringClassSpec{
			ControllerName: "openeverest.io/monitoring-" + name,
			ExtensionManaged: &monitoringv1alpha1.ExtensionManagedFulfilment{
				Requires: monitoringv1alpha1.SourceRequirements{Metrics: true},
			},
			DestinationParametersSchema: objectSchema(map[string]apiextensionsv1.JSONSchemaProps{
				labelsParameter: stringMap(),
				"interval":      {Type: "string"},
			}),
			InstanceParametersSchema: objectSchema(make(map[string]apiextensionsv1.JSONSchemaProps)),
		},
	})
}

func prometheusClass() *monitoringv1alpha1.MonitoringClass {
	return scrapeClass("prometheus", "podMonitorLabels")
}

func victoriaMetricsClass() *monitoringv1alpha1.MonitoringClass {
	return scrapeClass("victoriametrics", "vmPodScrapeLabels")
}

func pmmIntegration() map[string]corev1alpha1.MonitoringIntegration {
	return map[string]corev1alpha1.MonitoringIntegration{"pmm": {AgentVersions: ">=3.0.0 <4.0.0"}}
}

func newProvider(name string, components map[string]string, monitoring *corev1alpha1.ProviderMonitoring) *corev1alpha1.Provider {
	p := &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1alpha1.ProviderSpec{
			Components: make(map[string]corev1alpha1.Component),
			Monitoring: monitoring,
		},
	}
	for component, componentType := range components {
		p.Spec.Components[component] = corev1alpha1.Component{Type: componentType}
	}
	return p
}

func pxcProvider() *corev1alpha1.Provider {
	return newProvider("percona-xtradb-cluster",
		map[string]string{"engine": "pxc", "proxy": "haproxy"},
		&corev1alpha1.ProviderMonitoring{
			Integrations: pmmIntegration(),
			ComponentTypes: map[string]corev1alpha1.MonitoringComponentType{
				"pxc": {
					Kind: "mysql", Metrics: corev1alpha1.MetricsSourceExporter,
					CredentialProfiles: []string{"metrics", "queryAnalytics"}, Features: []string{"queryAnalytics"},
				},
				"haproxy": {Kind: "haproxy", Metrics: corev1alpha1.MetricsSourceNative},
			},
		})
}

func psmdbProvider() *corev1alpha1.Provider {
	return newProvider("percona-server-mongodb",
		map[string]string{"engine": "mongod", "configServer": "mongod", "router": "mongos"},
		&corev1alpha1.ProviderMonitoring{
			Integrations: pmmIntegration(),
			ComponentTypes: map[string]corev1alpha1.MonitoringComponentType{
				"mongod": {Kind: "mongodb", Metrics: corev1alpha1.MetricsSourceExporter},
				"mongos": {Kind: "mongodb", Metrics: corev1alpha1.MetricsSourceExporter},
			},
		})
}

func kserveProvider() *corev1alpha1.Provider {
	return newProvider("provider-kserve",
		map[string]string{"model": "llm"},
		&corev1alpha1.ProviderMonitoring{
			ComponentTypes: map[string]corev1alpha1.MonitoringComponentType{
				"llm": {Kind: "vllm", Metrics: corev1alpha1.MetricsSourceNative},
			},
		})
}

func pgProvider() *corev1alpha1.Provider {
	return newProvider("percona-postgresql",
		map[string]string{"engine": "postgresql", "proxy": "pgbouncer"},
		&corev1alpha1.ProviderMonitoring{
			Integrations: pmmIntegration(),
			ComponentTypes: map[string]corev1alpha1.MonitoringComponentType{
				"postgresql": {Kind: "postgresql", Metrics: corev1alpha1.MetricsSourceNone},
				"pgbouncer":  {Kind: "pgbouncer", Metrics: corev1alpha1.MetricsSourceNone},
			},
		})
}

// newInstance is an Instance of provider with one component per provider
// component, sending to entries.
func newInstance(provider *corev1alpha1.Provider, entries ...corev1alpha1.InstanceMonitoringDestination) *corev1alpha1.Instance {
	components := make(map[string]corev1alpha1.ComponentSpec)
	for name, comp := range provider.Spec.Components {
		components[name] = corev1alpha1.ComponentSpec{Type: comp.Type}
	}
	return &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: testInstance, Namespace: testNamespace, UID: testUID},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef: common.ObjectRef{Name: provider.Name},
			Components:  components,
			Monitoring:  &corev1alpha1.InstanceMonitoringSpec{Destinations: entries},
		},
	}
}

func entry(name, destination string) corev1alpha1.InstanceMonitoringDestination {
	return corev1alpha1.InstanceMonitoringDestination{
		Name:           name,
		DestinationRef: common.MonitoringDestinationRef{Name: destination},
	}
}

func raw(js string) *runtime.RawExtension {
	return &runtime.RawExtension{Raw: []byte(js)}
}

func newDestination(name, class string) *monitoringv1alpha1.MonitoringDestination {
	return &monitoringv1alpha1.MonitoringDestination{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace, Generation: 2},
		Spec:       monitoringv1alpha1.MonitoringDestinationSpec{ClassRef: common.ObjectRef{Name: class}},
	}
}

func pmmDestination(name string) *monitoringv1alpha1.MonitoringDestination {
	dst := newDestination(name, "pmm")
	dst.Spec.CredentialsSecretRef = &common.SecretRef{Name: name + "-credentials"}
	dst.Spec.Parameters = raw(`{"url":"https://pmm.example.com"}`)
	return dst
}

func newSecret(name string, data map[string]string) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Data:       make(map[string][]byte),
	}
	for k, v := range data {
		s.Data[k] = []byte(v)
	}
	return s
}

func reconcileInstance(t *testing.T, c client.Client) {
	t.Helper()
	r := &BindingReconciler{Client: c, Scheme: c.Scheme()}
	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testInstance}})
	require.NoError(t, err)
}

func updateInstance(t *testing.T, c client.Client, mutate func(*corev1alpha1.Instance)) {
	t.Helper()
	in := &corev1alpha1.Instance{}
	require.NoError(t, c.Get(t.Context(), types.NamespacedName{Namespace: testNamespace, Name: testInstance}, in))
	mutate(in)
	require.NoError(t, c.Update(t.Context(), in))
}

func getBinding(t *testing.T, c client.Client, entryName string) *monitoringv1alpha1.MonitoringBinding {
	t.Helper()
	b := &monitoringv1alpha1.MonitoringBinding{}
	require.NoError(t, c.Get(t.Context(),
		types.NamespacedName{Namespace: testNamespace, Name: rtmonitoring.BindingName(testInstance, entryName)}, b,
	))
	return b
}

func findAccepted(t *testing.T, conditions []metav1.Condition) metav1.Condition {
	t.Helper()
	cond := meta.FindStatusCondition(conditions, "Accepted")
	require.NotNil(t, cond, "no Accepted condition in %v", conditions)
	return *cond
}
