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

package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/internal/instanceprep"
)

const (
	probeNamespace = "conformance"
	probeName      = "probe"
)

// observation is everything the harness saw during one render.
type observation struct {
	applied  []string
	requests []string
}

// contains reports whether the token surfaced, and where.
func (o observation) contains(token string) (outcome, bool) {
	for _, obj := range o.applied {
		if strings.Contains(obj, token) {
			return reconciled, true
		}
	}
	for _, req := range o.requests {
		if strings.Contains(req, token) {
			return read, true
		}
	}
	return ignored, false
}

// changedFrom reports whether the provider behaved differently, and how. Some
// fields never surface as a literal — a shard count changes how many objects
// exist rather than appearing in one, and a toggle may only change what the
// provider reads — so a change is evidence in itself.
func (o observation) changedFrom(baseline observation) (outcome, bool) {
	if strings.Join(o.applied, "\n") != strings.Join(baseline.applied, "\n") {
		return reconciled, true
	}
	if strings.Join(o.requests, "\n") != strings.Join(baseline.requests, "\n") {
		return read, true
	}
	return ignored, false
}

// probeBool decides a boolean by rendering it both ways rather than searching
// for its value: "true" is a substring of every boolean the provider sets for
// its own reasons, so a token would match objects that have nothing to do with
// this field. Two values exhaust a bool, so if neither render moves the
// provider off its baseline, nothing consumes the field.
func probeBool(cfg Config, spec *corev1alpha1.ProviderSpec, topology, path string) result {
	unverifiable := func(detail string, requests []string) result {
		return result{topology: topology, path: path, outcome: unverified, detail: detail, requests: requests}
	}

	baseline, err := render(cfg, spec, topology, "", nil)
	if err != nil {
		return unverifiable(fmt.Sprintf("baseline render failed: %v", err), baseline.requests)
	}

	for _, value := range []bool{true, false} {
		probed, err := render(cfg, spec, topology, path, value)
		if err != nil {
			return unverifiable(err.Error(), probed.requests)
		}
		if got, changed := probed.changedFrom(baseline); changed {
			return result{topology: topology, path: path, outcome: got, requests: probed.requests}
		}
	}
	return result{topology: topology, path: path, outcome: ignored}
}

func probeTopology(
	t *testing.T,
	cfg Config,
	spec *corev1alpha1.ProviderSpec,
	topology string,
	paths []string,
) []result {
	t.Helper()

	results := make([]result, 0, len(paths))
	for _, path := range paths {
		results = append(results, probePath(cfg, spec, topology, path))
	}
	return results
}

func probePath(cfg Config, spec *corev1alpha1.ProviderSpec, topology, path string) result {
	failed := func(detail string, requests []string) result {
		return result{topology: topology, path: path, outcome: unverified, detail: detail, requests: requests}
	}

	kind, err := resolveLeaf(spec, topology, path)
	if err != nil {
		return failed(err.Error(), nil)
	}
	if kind == leafBool {
		return probeBool(cfg, spec, topology, path)
	}
	value, token, err := sentinel(spec, path, kind)
	if err != nil {
		return failed(err.Error(), nil)
	}

	baseline, err := render(cfg, spec, topology, "", nil)
	if err != nil {
		return failed(fmt.Sprintf("baseline render failed: %v", err), baseline.requests)
	}
	if _, found := baseline.contains(token); found {
		return failed("the probe value already appears without being set", nil)
	}

	probed, err := render(cfg, spec, topology, path, value)
	// A failed Sync does not discard what was already observed: if the value
	// reached a request on the way, the field was read.
	if got, found := probed.contains(token); found {
		return result{topology: topology, path: path, outcome: got, requests: probed.requests}
	}
	if err != nil {
		return failed(err.Error(), probed.requests)
	}

	got, _ := probed.changedFrom(baseline)
	return result{topology: topology, path: path, outcome: got, requests: probed.requests}
}

// render runs Sync once against an Instance carrying value at path, and reports
// every object the provider applied and every request it made.
func render(
	cfg Config,
	spec *corev1alpha1.ProviderSpec,
	topology, path string,
	value any,
) (observation, error) {
	var obs observation

	scheme, err := buildScheme(cfg.Provider)
	if err != nil {
		return obs, err
	}

	instance, err := buildInstance(spec, cfg.Provider.Name(), topology, path, value)
	if err != nil {
		return obs, err
	}

	// Hand the provider the Instance the reconciler would, not the one a user
	// wrote: anything the runtime resolves before Sync happens here too.
	instance, _, err = instanceprep.PrepareForSync(spec, instance)
	if err != nil {
		return obs, err
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(instance, providerObject(cfg.Provider.Name(), spec)).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				obs.requests = append(obs.requests, describe(scheme, obj)+"/"+key.Name)
				return c.Get(ctx, key, obj, opts...)
			},
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				obs.applied = append(obs.applied, serialize(obj))
				return c.Create(ctx, obj, opts...)
			},
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				obs.applied = append(obs.applied, serialize(obj))
				return c.Update(ctx, obj, opts...)
			},
		}).
		Build()

	ctx := controller.NewContext(context.Background(), cl, instance, cfg.Provider.Name())
	if err := cfg.Provider.Sync(ctx); err != nil {
		return obs, fmt.Errorf("sync: %w", err)
	}
	return obs, nil
}

func buildScheme(provider ProviderUnderTest) (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1alpha1.AddToScheme, corev1.AddToScheme} {
		if err := add(scheme); err != nil {
			return nil, err
		}
	}
	if types := provider.Types(); types != nil {
		if err := types(scheme); err != nil {
			return nil, fmt.Errorf("registering provider types: %w", err)
		}
	}
	return scheme, nil
}

// buildInstance assembles the smallest Instance the topology allows: every
// required component, plus whichever optional component the path addresses.
func buildInstance(
	spec *corev1alpha1.ProviderSpec,
	providerName, topology, path string,
	value any,
) (*corev1alpha1.Instance, error) {
	components := map[string]any{}
	for name, topologyComponent := range spec.Topologies[topology].Components {
		if topologyComponent.Optional && !addressesComponent(path, name) {
			continue
		}
		components[name] = map[string]any{
			"name": name,
			"type": spec.Components[name].Type,
		}
	}

	object := map[string]any{
		"apiVersion": corev1alpha1.GroupVersion.String(),
		"kind":       "Instance",
		"metadata": map[string]any{
			"name":      probeName,
			"namespace": probeNamespace,
		},
		"spec": map[string]any{
			"providerRef":     map[string]any{"name": providerName},
			"topology":        map[string]any{"type": topology},
			componentsSegment: components,
		},
	}

	if value != nil {
		if err := unstructured.SetNestedField(object, value, strings.Split(path, ".")...); err != nil {
			return nil, fmt.Errorf("setting %s: %w", path, err)
		}
	}

	instance := &corev1alpha1.Instance{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object, instance); err != nil {
		return nil, fmt.Errorf("building an Instance with %s set: %w", path, err)
	}
	return instance, nil
}

func addressesComponent(path, component string) bool {
	return strings.HasPrefix(path, "spec."+componentsSegment+"."+component+".")
}

func serialize(obj client.Object) string {
	data, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(data)
}

func describe(scheme *runtime.Scheme, obj client.Object) string {
	kinds, _, err := scheme.ObjectKinds(obj)
	if err != nil || len(kinds) == 0 {
		return fmt.Sprintf("%T", obj)
	}
	return kinds[0].Kind
}
