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

package k8s

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/AlekSi/pointer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/events"
)

// ListMonitoringDestinations returns the monitoring destinations in a
// namespace, only those of params.Class and referencing params.Secret when
// they are set. Both filters are field selectors answered by the Kubernetes
// API server, as for kubectl get --field-selector. A Secret is referenced
// through either of two fields, so it takes one list per field, merged
// with each destination once.
func (h *k8sHandler) ListMonitoringDestinations(ctx context.Context, _, namespace string, params *api.ListMonitoringDestinationsParams) (*monitoringv1alpha1.MonitoringDestinationList, error) {
	if params == nil {
		params = &api.ListMonitoringDestinationsParams{}
	}
	fields := ctrlclient.MatchingFields{}
	if class := pointer.Get(params.Class); class != "" {
		fields[monitoringv1alpha1.DestinationClassField] = class
	}
	secret := pointer.Get(params.Secret)
	if secret == "" {
		return h.listMonitoringDestinations(ctx, namespace, fields)
	}

	withSecret := func(field string) ctrlclient.MatchingFields {
		selected := maps.Clone(fields)
		selected[field] = secret
		return selected
	}
	merged, err := h.listMonitoringDestinations(ctx, namespace, withSecret(monitoringv1alpha1.DestinationCredentialsSecretField))
	if err != nil {
		return nil, err
	}
	byCA, err := h.listMonitoringDestinations(ctx, namespace, withSecret(monitoringv1alpha1.DestinationCASecretField))
	if err != nil {
		return nil, err
	}
	for _, destination := range byCA.Items {
		if !slices.ContainsFunc(merged.Items, func(d monitoringv1alpha1.MonitoringDestination) bool {
			return d.GetName() == destination.GetName()
		}) {
			merged.Items = append(merged.Items, destination)
		}
	}
	// Two reads have no single resourceVersion to report.
	merged.ResourceVersion = ""
	slices.SortFunc(merged.Items, func(a, b monitoringv1alpha1.MonitoringDestination) int {
		return strings.Compare(a.GetName(), b.GetName())
	})
	return merged, nil
}

func (h *k8sHandler) listMonitoringDestinations(ctx context.Context, namespace string, fields ctrlclient.MatchingFields) (*monitoringv1alpha1.MonitoringDestinationList, error) {
	opts := []ctrlclient.ListOption{ctrlclient.InNamespace(namespace)}
	if len(fields) > 0 {
		opts = append(opts, fields)
	}
	return h.kubeConnector.ListMonitoringDestinations(ctx, opts...)
}

// GetMonitoringDestination returns a monitoring destination by name and namespace.
func (h *k8sHandler) GetMonitoringDestination(ctx context.Context, _, namespace, name string) (*monitoringv1alpha1.MonitoringDestination, error) {
	return h.kubeConnector.GetMonitoringDestination(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// CreateMonitoringDestination creates a monitoring destination in the namespace
// from the path, built only from the members a client may author.
func (h *k8sHandler) CreateMonitoringDestination(ctx context.Context, _, namespace string, destination *monitoringv1alpha1.MonitoringDestination) (*monitoringv1alpha1.MonitoringDestination, error) {
	authored := &monitoringv1alpha1.MonitoringDestination{
		ObjectMeta: metav1.ObjectMeta{
			Name:         destination.GetName(),
			GenerateName: destination.GetGenerateName(),
			Namespace:    namespace,
			Labels:       destination.GetLabels(),
			Annotations:  withManagedAnnotations(destination.GetAnnotations(), nil),
		},
		Spec: destination.Spec,
	}
	stampActor(ctx, authored)
	return h.kubeConnector.CreateMonitoringDestination(ctx, authored)
}

// UpdateMonitoringDestination replaces the labels, annotations and spec of the
// monitoring destination named by the path. Everything else, the server-stamped
// annotations included, is kept from the stored object.
func (h *k8sHandler) UpdateMonitoringDestination(ctx context.Context, _, namespace, name string, destination *monitoringv1alpha1.MonitoringDestination) (*monitoringv1alpha1.MonitoringDestination, error) {
	stored, err := h.kubeConnector.GetMonitoringDestination(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return nil, err
	}
	// The client's version, when it sends one, so a stale document is refused.
	if rv := destination.GetResourceVersion(); rv != "" {
		stored.ResourceVersion = rv
	}
	stored.Labels = destination.GetLabels()
	stored.Annotations = withManagedAnnotations(destination.GetAnnotations(), stored.GetAnnotations())
	stored.Spec = destination.Spec
	stampActor(ctx, stored)
	return h.kubeConnector.UpdateMonitoringDestination(ctx, stored)
}

// DeleteMonitoringDestination deletes a monitoring destination.
func (h *k8sHandler) DeleteMonitoringDestination(ctx context.Context, _, namespace, name string) error {
	return h.kubeConnector.DeleteMonitoringDestination(ctx, &monitoringv1alpha1.MonitoringDestination{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	})
}

// withManagedAnnotations returns the client's annotations with the
// server-stamped ones replaced by those of the stored object.
func withManagedAnnotations(authored, stored map[string]string) map[string]string {
	result := maps.Clone(authored)
	for _, key := range []string{events.AnnotationActorType, events.AnnotationActorID} {
		delete(result, key)
		if value, found := stored[key]; found {
			if result == nil {
				result = make(map[string]string)
			}
			result[key] = value
		}
	}
	return result
}
