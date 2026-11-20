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
	"slices"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	pkgcommon "github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/events"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestK8s_MonitoringDestinationWrites(t *testing.T) {
	t.Parallel()

	ownerRefs := []metav1.OwnerReference{{APIVersion: "v1", Kind: "ConfigMap", Name: "owner", UID: "1234"}}
	stored := func() *monitoringv1alpha1.MonitoringDestination {
		return &monitoringv1alpha1.MonitoringDestination{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pmm-prod", Namespace: "ns1",
				Labels:          map[string]string{"team": "a"},
				Annotations:     map[string]string{events.AnnotationActorType: "user", events.AnnotationActorID: "carol"},
				OwnerReferences: ownerRefs,
				Finalizers:      []string{"example.com/keep"},
			},
			Spec: monitoringv1alpha1.MonitoringDestinationSpec{
				ClassRef:   common.ObjectRef{Name: "pmm"},
				Parameters: &runtime.RawExtension{Raw: []byte(`{"url":"https://pmm.example.com"}`)},
			},
			Status: monitoringv1alpha1.MonitoringDestinationStatus{ServerVersion: "3.8.0"},
		}
	}
	// What a client sends: everything it may not author set to something else.
	forged := func() *monitoringv1alpha1.MonitoringDestination {
		return &monitoringv1alpha1.MonitoringDestination{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pmm-prod", Namespace: "other-ns",
				Labels: map[string]string{"team": "b"},
				Annotations: map[string]string{
					"team.example.com/owner":   "platform",
					events.AnnotationActorType: "user",
					events.AnnotationActorID:   "mallory",
				},
				OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "Secret", Name: "victim", UID: "5678"}},
				Finalizers:      []string{"example.com/stuck"},
			},
			Spec: monitoringv1alpha1.MonitoringDestinationSpec{
				ClassRef:       common.ObjectRef{Name: "pmm"},
				Authentication: common.Authentication{Mode: "ServiceAccountToken", CredentialsSecretRef: &common.SecretRef{Name: "pmm-prod-credentials"}},
				Parameters:     &runtime.RawExtension{Raw: []byte(`{"url":"https://pmm2.example.com"}`)},
			},
			Status: monitoringv1alpha1.MonitoringDestinationStatus{
				ServerVersion: "9.9.9",
				Conditions: []metav1.Condition{{
					Type: "Ready", Status: metav1.ConditionTrue, Reason: "Ready", LastTransitionTime: metav1.Now(),
				}},
			},
		}
	}

	newHandler := func(objs ...ctrlclient.Object) *k8sHandler {
		fakeClient := fake.NewClientBuilder().
			WithScheme(kubernetes.CreateScheme()).
			WithObjects(objs...).
			Build()
		return &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}
	}
	bob := context.WithValue(context.Background(), pkgcommon.UserCtxKey, &jwt.Token{ //nolint:staticcheck
		Claims: jwt.MapClaims{"sub": "bob", "iss": "everest"},
	})

	t.Run("create keeps only what a client may author, in the path namespace", func(t *testing.T) {
		t.Parallel()

		handler := newHandler()
		_, err := handler.CreateMonitoringDestination(context.Background(), "prod", "ns1", forged())
		require.NoError(t, err)

		created, err := handler.kubeConnector.GetMonitoringDestination(context.Background(), types.NamespacedName{Namespace: "ns1", Name: "pmm-prod"})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"team": "b"}, created.Labels)
		assert.Equal(t, map[string]string{"team.example.com/owner": "platform"}, created.Annotations, "forged actor annotations must not survive when there is no caller to stamp")
		assert.Empty(t, created.OwnerReferences)
		assert.Empty(t, created.Finalizers)
		assert.Equal(t, monitoringv1alpha1.MonitoringDestinationStatus{}, created.Status)
		assert.Equal(t, "pmm-prod-credentials", created.Spec.Authentication.CredentialsSecretRef.Name)
		assert.JSONEq(t, `{"url":"https://pmm2.example.com"}`, string(created.Spec.Parameters.Raw))

		_, err = handler.kubeConnector.GetMonitoringDestination(context.Background(), types.NamespacedName{Namespace: "other-ns", Name: "pmm-prod"})
		assert.True(t, k8serrors.IsNotFound(err), "the namespace in the body must be ignored")
	})

	t.Run("create stamps the caller over forged actor annotations", func(t *testing.T) {
		t.Parallel()

		created, err := newHandler().CreateMonitoringDestination(bob, "prod", "ns1", forged())
		require.NoError(t, err)
		assert.Equal(t, "user", created.Annotations[events.AnnotationActorType])
		assert.Equal(t, "bob", created.Annotations[events.AnnotationActorID])
		assert.Equal(t, "platform", created.Annotations["team.example.com/owner"])
	})

	t.Run("update replaces labels, annotations and spec and keeps the rest", func(t *testing.T) {
		t.Parallel()

		handler := newHandler(stored())
		current, err := handler.kubeConnector.GetMonitoringDestination(context.Background(), types.NamespacedName{Namespace: "ns1", Name: "pmm-prod"})
		require.NoError(t, err)

		body := forged()
		body.Name = "renamed"
		body.ResourceVersion = current.ResourceVersion
		_, err = handler.UpdateMonitoringDestination(context.Background(), "prod", "ns1", "pmm-prod", body)
		require.NoError(t, err)

		updated, err := handler.kubeConnector.GetMonitoringDestination(context.Background(), types.NamespacedName{Namespace: "ns1", Name: "pmm-prod"})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"team": "b"}, updated.Labels)
		wantAnnotations := map[string]string{
			"team.example.com/owner":   "platform",
			events.AnnotationActorType: "user",
			events.AnnotationActorID:   "carol",
		}
		assert.Equal(t, wantAnnotations, updated.Annotations, "the stored actor annotations win over forged ones")
		assert.Equal(t, ownerRefs, updated.OwnerReferences)
		assert.Equal(t, []string{"example.com/keep"}, updated.Finalizers)
		assert.Equal(t, "3.8.0", updated.Status.ServerVersion)
		assert.Empty(t, updated.Status.Conditions)
		assert.JSONEq(t, `{"url":"https://pmm2.example.com"}`, string(updated.Spec.Parameters.Raw))
		assert.Equal(t, "pmm-prod-credentials", updated.Spec.Authentication.CredentialsSecretRef.Name)

		_, err = handler.kubeConnector.GetMonitoringDestination(context.Background(), types.NamespacedName{Namespace: "ns1", Name: "renamed"})
		assert.True(t, k8serrors.IsNotFound(err), "the name in the body must be ignored")
	})

	t.Run("update stamps the caller", func(t *testing.T) {
		t.Parallel()

		handler := newHandler(stored())
		current, err := handler.kubeConnector.GetMonitoringDestination(bob, types.NamespacedName{Namespace: "ns1", Name: "pmm-prod"})
		require.NoError(t, err)

		body := forged()
		body.ResourceVersion = current.ResourceVersion
		updated, err := handler.UpdateMonitoringDestination(bob, "prod", "ns1", "pmm-prod", body)
		require.NoError(t, err)
		assert.Equal(t, "bob", updated.Annotations[events.AnnotationActorID])
	})

	t.Run("update with a stale resourceVersion conflicts and changes nothing", func(t *testing.T) {
		t.Parallel()

		handler := newHandler(stored())
		body := forged()
		body.ResourceVersion = "1"
		_, err := handler.UpdateMonitoringDestination(context.Background(), "prod", "ns1", "pmm-prod", body)
		require.True(t, k8serrors.IsConflict(err), "want a conflict, got %v", err)

		unchanged, err := handler.kubeConnector.GetMonitoringDestination(context.Background(), types.NamespacedName{Namespace: "ns1", Name: "pmm-prod"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"url":"https://pmm.example.com"}`, string(unchanged.Spec.Parameters.Raw))
	})

	t.Run("update without a resourceVersion applies to the stored version and keeps the rest", func(t *testing.T) {
		t.Parallel()

		handler := newHandler(stored())
		_, err := handler.UpdateMonitoringDestination(context.Background(), "prod", "ns1", "pmm-prod", forged())
		require.NoError(t, err)

		updated, err := handler.kubeConnector.GetMonitoringDestination(context.Background(), types.NamespacedName{Namespace: "ns1", Name: "pmm-prod"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"url":"https://pmm2.example.com"}`, string(updated.Spec.Parameters.Raw))
		assert.Equal(t, ownerRefs, updated.OwnerReferences)
		assert.Equal(t, []string{"example.com/keep"}, updated.Finalizers)
	})

	t.Run("update of a missing destination is not found", func(t *testing.T) {
		t.Parallel()

		_, err := newHandler().UpdateMonitoringDestination(context.Background(), "prod", "ns1", "pmm-prod", forged())
		assert.True(t, k8serrors.IsNotFound(err), "want not found, got %v", err)
	})

	t.Run("delete removes the destination named by the path", func(t *testing.T) {
		t.Parallel()

		withoutFinalizers := stored()
		withoutFinalizers.Finalizers = nil
		handler := newHandler(withoutFinalizers)
		require.NoError(t, handler.DeleteMonitoringDestination(context.Background(), "prod", "ns1", "pmm-prod"))

		_, err := handler.kubeConnector.GetMonitoringDestination(context.Background(), types.NamespacedName{Namespace: "ns1", Name: "pmm-prod"})
		assert.True(t, k8serrors.IsNotFound(err), "want not found, got %v", err)
	})
}

func TestK8s_ListMonitoringDestinations(t *testing.T) {
	t.Parallel()

	destination := func(namespace, name, class, credentials, ca string) ctrlclient.Object {
		d := &monitoringv1alpha1.MonitoringDestination{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
			Spec:       monitoringv1alpha1.MonitoringDestinationSpec{ClassRef: common.ObjectRef{Name: class}},
		}
		if credentials != "" {
			d.Spec.Authentication = common.Authentication{Mode: "BasicAuth", CredentialsSecretRef: &common.SecretRef{Name: credentials}}
		}
		if ca != "" {
			d.Spec.TLS = &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: ca, Key: "ca.crt"}}}
		}
		return d
	}
	// The values the API server reads for the CRD's selectableFields.
	index := func(value func(spec monitoringv1alpha1.MonitoringDestinationSpec) string) ctrlclient.IndexerFunc {
		return func(o ctrlclient.Object) []string {
			d, _ := o.(*monitoringv1alpha1.MonitoringDestination)
			if v := value(d.Spec); v != "" {
				return []string{v}
			}
			return nil
		}
	}
	// newHandler also returns the field selectors of every list call, in order.
	newHandler := func() (*k8sHandler, *[]fields.Set) {
		var selectors []fields.Set
		d := &monitoringv1alpha1.MonitoringDestination{}
		fakeClient := fake.NewClientBuilder().
			WithScheme(kubernetes.CreateScheme()).
			WithIndex(d, monitoringv1alpha1.DestinationClassField, index(func(s monitoringv1alpha1.MonitoringDestinationSpec) string {
				return s.ClassRef.Name
			})).
			WithIndex(d, monitoringv1alpha1.DestinationCredentialsSecretField, index(func(s monitoringv1alpha1.MonitoringDestinationSpec) string {
				if s.Authentication.CredentialsSecretRef == nil {
					return ""
				}
				return s.Authentication.CredentialsSecretRef.Name
			})).
			WithIndex(d, monitoringv1alpha1.DestinationCASecretField, index(func(s monitoringv1alpha1.MonitoringDestinationSpec) string {
				if s.TLS == nil || s.TLS.CA == nil || s.TLS.CA.SecretKeyRef == nil {
					return ""
				}
				return s.TLS.CA.SecretKeyRef.Name
			})).
			WithObjects(
				destination("ns1", "pmm-prod", "pmm", "pmm-creds", "corp-ca"),
				destination("ns1", "pmm-staging", "pmm", "pmm-staging-creds", "shared"),
				destination("ns1", "prom", "prometheus", "", "corp-ca"),
				destination("ns1", "prom-shared", "prometheus", "shared", "shared"),
				destination("ns1", "anonymous", "prometheus", "", ""),
				destination("ns2", "pmm-other", "pmm", "pmm-creds", "corp-ca"),
			).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(ctx context.Context, c ctrlclient.WithWatch, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error {
					o := (&ctrlclient.ListOptions{}).ApplyOptions(opts)
					selected := fields.Set{}
					if o.FieldSelector != nil {
						for _, r := range o.FieldSelector.Requirements() {
							selected[r.Field] = r.Value
						}
					}
					selectors = append(selectors, selected)
					return c.List(ctx, list, opts...)
				},
			}).
			Build()
		return &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}, &selectors
	}

	const (
		classField       = monitoringv1alpha1.DestinationClassField
		credentialsField = monitoringv1alpha1.DestinationCredentialsSecretField
		caField          = monitoringv1alpha1.DestinationCASecretField
	)
	everything := []string{"anonymous", "pmm-prod", "pmm-staging", "prom", "prom-shared"}
	tests := map[string]struct {
		params        *api.ListMonitoringDestinationsParams
		want          []string
		wantSelectors []fields.Set
	}{
		"nil params return the namespace": {
			params:        nil,
			want:          everything,
			wantSelectors: []fields.Set{{}},
		},
		"empty filters are ignored": {
			params:        &api.ListMonitoringDestinationsParams{Class: new(""), Secret: new("")},
			want:          everything,
			wantSelectors: []fields.Set{{}},
		},
		"class with two destinations": {
			params:        &api.ListMonitoringDestinationsParams{Class: new("pmm")},
			want:          []string{"pmm-prod", "pmm-staging"},
			wantSelectors: []fields.Set{{classField: "pmm"}},
		},
		"class with destinations with and without Secrets": {
			params:        &api.ListMonitoringDestinationsParams{Class: new("prometheus")},
			want:          []string{"anonymous", "prom", "prom-shared"},
			wantSelectors: []fields.Set{{classField: "prometheus"}},
		},
		"class with no destination matches": {
			params:        &api.ListMonitoringDestinationsParams{Class: new("victoriametrics")},
			want:          []string{},
			wantSelectors: []fields.Set{{classField: "victoriametrics"}},
		},
		"secret used as credentials only": {
			params:        &api.ListMonitoringDestinationsParams{Secret: new("pmm-creds")},
			want:          []string{"pmm-prod"},
			wantSelectors: []fields.Set{{credentialsField: "pmm-creds"}, {caField: "pmm-creds"}},
		},
		"secret used as CA only": {
			params:        &api.ListMonitoringDestinationsParams{Secret: new("corp-ca")},
			want:          []string{"pmm-prod", "prom"},
			wantSelectors: []fields.Set{{credentialsField: "corp-ca"}, {caField: "corp-ca"}},
		},
		"secret used as credentials and CA, by the same destination and another, lists each once": {
			params:        &api.ListMonitoringDestinationsParams{Secret: new("shared")},
			want:          []string{"pmm-staging", "prom-shared"},
			wantSelectors: []fields.Set{{credentialsField: "shared"}, {caField: "shared"}},
		},
		"secret and class must both match": {
			params:        &api.ListMonitoringDestinationsParams{Class: new("prometheus"), Secret: new("corp-ca")},
			want:          []string{"prom"},
			wantSelectors: []fields.Set{{classField: "prometheus", credentialsField: "corp-ca"}, {classField: "prometheus", caField: "corp-ca"}},
		},
		"secret no destination references": {
			params:        &api.ListMonitoringDestinationsParams{Secret: new("unused")},
			want:          []string{},
			wantSelectors: []fields.Set{{credentialsField: "unused"}, {caField: "unused"}},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h, selectors := newHandler()
			list, err := h.ListMonitoringDestinations(context.Background(), "prod", "ns1", tc.params)
			require.NoError(t, err)
			got := make([]string, 0, len(list.Items))
			for _, d := range list.Items {
				got = append(got, d.Name)
			}
			assert.ElementsMatch(t, tc.want, got)
			assert.Equal(t, tc.wantSelectors, *selectors)
			if tc.params != nil && pointer.Get(tc.params.Secret) != "" {
				assert.True(t, slices.IsSorted(got), "merged list sorted by name: %v", got)
				assert.Empty(t, list.ResourceVersion)
			}
		})
	}
}
