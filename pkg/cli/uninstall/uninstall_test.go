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

package uninstall

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/cli/namespaces"
	"github.com/openeverest/openeverest/v2/pkg/cli/steps"
	inuseuninstall "github.com/openeverest/openeverest/v2/pkg/inuse/uninstall"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

const coreNamespace = "everest-system"

func newTestUninstall(newReader func() (ctrlclient.Reader, error)) *Uninstall {
	l := zap.NewNop().Sugar()
	return &Uninstall{
		kubeConnector:  kubernetes.NewEmpty(l, coreNamespace),
		l:              l,
		newCheckReader: newReader,
		checkInterval:  time.Millisecond,
		checkTimeout:   50 * time.Millisecond,
	}
}

func descs(s []steps.Step) []string {
	out := make([]string, 0, len(s))
	for _, step := range s {
		out = append(out, step.Desc)
	}
	return out
}

func destination(finalizers ...string) *monitoringv1alpha1.MonitoringDestination {
	return &monitoringv1alpha1.MonitoringDestination{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "prometheus", Finalizers: finalizers},
	}
}

func deleting(d *monitoringv1alpha1.MonitoringDestination) *monitoringv1alpha1.MonitoringDestination {
	d.DeletionTimestamp = &metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return d
}

func newReader(t *testing.T, funcs interceptor.Funcs, objs ...ctrlclient.Object) ctrlclient.WithWatch { //nolint:ireturn
	t.Helper()
	return fake.NewClientBuilder().WithScheme(kubernetes.CreateScheme()).WithObjects(objs...).WithInterceptorFuncs(funcs).Build()
}

// countDestinationLists counts the checks, and runs then on the n-th.
func countDestinationLists(n int, then func(ctx context.Context, c ctrlclient.WithWatch), lists *int) interceptor.Funcs {
	return interceptor.Funcs{
		List: func(ctx context.Context, c ctrlclient.WithWatch, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error {
			if list.GetObjectKind().GroupVersionKind().Kind == "MonitoringDestinationList" {
				*lists++
				if *lists == n && then != nil {
					then(ctx, c)
				}
			}
			return c.List(ctx, list, opts...)
		},
	}
}

func TestNewUninstallSteps(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		namespaces *corev1.NamespaceList
	}{
		{name: "no database namespaces"},
		{
			name: "database namespaces",
			namespaces: &corev1.NamespaceList{Items: []corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "team-a"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "team-b"}},
			}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			u := newTestUninstall(nil)

			got := descs(u.newUninstallSteps(tc.namespaces))

			// The wait follows the removal of the database namespaces,
			// which it waits for, and comes right before the chart goes.
			i := slices.Index(got, "Waiting for OpenEverest to release in-use objects")
			require.NotEqual(t, -1, i, got)
			databaseSteps := 0
			if tc.namespaces != nil {
				for _, ns := range tc.namespaces.Items {
					databaseSteps += len(namespaces.NewRemoveNamespaceSteps(ns.Name, false, u.kubeConnector))
				}
			}
			assert.Equal(t, databaseSteps, i, got)
			want := []string{
				"Waiting for OpenEverest to release in-use objects",
				"Uninstalling Helm chart",
				"Deleting namespace 'everest-monitoring'",
				"Deleting namespace 'everest-system'",
				"Uninstalling CRDs",
			}
			assert.Equal(t, want, got[i:])
		})
	}
}

func TestWaitForInUseRelease(t *testing.T) {
	t.Parallel()

	t.Run("passes when nothing is held", func(t *testing.T) {
		t.Parallel()
		lists := 0
		r := newReader(t, countDestinationLists(0, nil, &lists), destination())
		u := newTestUninstall(nil)

		require.NoError(t, u.waitForInUseRelease(t.Context(), r))
		assert.Equal(t, 1, lists)
	})

	t.Run("polls until core releases what is being deleted", func(t *testing.T) {
		t.Parallel()
		lists := 0
		release := func(ctx context.Context, c ctrlclient.WithWatch) {
			// Core's controller releases the destination; it goes.
			d := destination()
			require.NoError(t, c.Get(ctx, ctrlclient.ObjectKeyFromObject(d), d))
			d.Finalizers = nil
			require.NoError(t, c.Update(ctx, d))
		}
		r := newReader(t, countDestinationLists(3, release, &lists),
			deleting(destination(monitoringv1alpha1.InUseProtectionFinalizer)),
		)
		u := newTestUninstall(nil)
		u.checkTimeout = time.Hour

		require.NoError(t, u.waitForInUseRelease(t.Context(), r))
		assert.Equal(t, 3, lists)
	})

	t.Run("times out with the objects held", func(t *testing.T) {
		t.Parallel()
		r := newReader(t, interceptor.Funcs{}, deleting(destination(monitoringv1alpha1.InUseProtectionFinalizer)))
		u := newTestUninstall(nil)

		err := u.waitForInUseRelease(t.Context(), r)

		_, held := errors.AsType[*inuseuninstall.HeldError](err)
		require.True(t, held, "want the objects held, got %v", err)
		assert.Contains(t, err.Error(), "1 MonitoringDestination: team-a/prometheus (being deleted)")
		assert.Contains(t, err.Error(), "helm uninstall --no-hooks")
	})

	t.Run("fails at once when nothing held is being deleted", func(t *testing.T) {
		t.Parallel()
		lists := 0
		r := newReader(t, countDestinationLists(0, nil, &lists), destination(monitoringv1alpha1.InUseProtectionFinalizer))
		u := newTestUninstall(nil)
		u.checkTimeout = time.Hour

		err := u.waitForInUseRelease(t.Context(), r)

		_, held := errors.AsType[*inuseuninstall.HeldError](err)
		require.True(t, held, "want the objects held, got %v", err)
		assert.Contains(t, err.Error(), "1 MonitoringDestination: team-a/prometheus\n")
		assert.Equal(t, 1, lists)
	})

	t.Run("fails when the check cannot run", func(t *testing.T) {
		t.Parallel()
		funcs := interceptor.Funcs{
			List: func(context.Context, ctrlclient.WithWatch, ctrlclient.ObjectList, ...ctrlclient.ListOption) error {
				return apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, "", errors.New("no"))
			},
		}
		r := newReader(t, funcs)
		u := newTestUninstall(nil)
		u.checkTimeout = time.Hour

		require.True(t, apierrors.IsForbidden(u.waitForInUseRelease(t.Context(), r)))
	})
}

func TestStepWaitForInUseRelease(t *testing.T) {
	t.Parallel()

	t.Run("checks with the reader it creates", func(t *testing.T) {
		t.Parallel()
		r := newReader(t, interceptor.Funcs{}, destination(monitoringv1alpha1.InUseProtectionFinalizer))
		u := newTestUninstall(func() (ctrlclient.Reader, error) { return r, nil })

		err := u.newStepWaitForInUseRelease().F(t.Context())

		assert.ErrorContains(t, err, "team-a/prometheus")
	})

	t.Run("fails without a client", func(t *testing.T) {
		t.Parallel()
		u := newTestUninstall(func() (ctrlclient.Reader, error) { return nil, errors.New("no kubeconfig") })

		require.ErrorContains(t, u.newStepWaitForInUseRelease().F(t.Context()), "no kubeconfig")
	})
}
