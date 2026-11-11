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

package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	monitoringcontroller "github.com/openeverest/openeverest/v2/internal/controller/monitoring"
	"github.com/openeverest/openeverest/v2/internal/controller/secretusage"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// recordingManager records what is added to the manager instead of running
// it.
type recordingManager struct {
	manager.Manager

	added []manager.Runnable
}

func (m *recordingManager) Add(r manager.Runnable) error {
	m.added = append(m.added, r)
	return nil
}

func TestSetupSecretProtection(t *testing.T) {
	t.Parallel()

	for _, enabled := range []bool{true, false} {
		mgr, err := ctrl.NewManager(&rest.Config{Host: "https://127.0.0.1:1"}, ctrl.Options{
			Scheme:                 scheme,
			Metrics:                metricsserver.Options{BindAddress: "0"},
			HealthProbeBindAddress: "0",
			Controller:             config.Controller{SkipNameValidation: new(true)},
		})
		require.NoError(t, err)
		rec := &recordingManager{Manager: mgr}

		require.NoError(t, setupSecretProtection(rec, enabled, []secretusage.Referrer{monitoringcontroller.SecretReferrer()}))

		require.Len(t, rec.added, 1)
		_, isController := rec.added[0].(controller.Controller)
		_, isRelease := rec.added[0].(manager.RunnableFunc)
		if enabled {
			assert.True(t, isController, "protection enabled: the Secret usage reconciler")
		} else {
			assert.False(t, isController, "protection disabled: no Secret usage reconciler")
			assert.True(t, isRelease, "protection disabled: the release at start")
		}
	}
}

func TestReleaseSecrets(t *testing.T) {
	t.Parallel()

	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name: "pmm-token", Namespace: "team-a",
		Labels:      map[string]string{common.OpenEverestDefinitionLabel: common.OpenEverestCABundleDefinition},
		Annotations: map[string]string{common.AttachedDestinationsAnnotation: "1", "example.com/note": "x"},
		Finalizers:  []string{common.OpenEverestInUseProtectionFinalizer, "example.com/keep"},
	}}
	dst := &monitoringv1alpha1.MonitoringDestination{ObjectMeta: metav1.ObjectMeta{
		Name: "pmm-prod", Namespace: "team-a", Finalizers: []string{monitoringv1alpha1.InUseProtectionFinalizer},
	}}
	// The first patch fails, as when the API server is briefly unavailable.
	var patches atomic.Int32
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret, dst).WithInterceptorFuncs(interceptor.Funcs{
		Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			if patches.Add(1) == 1 {
				return errors.New("connection refused")
			}
			return c.Patch(ctx, obj, patch, opts...)
		},
	}).Build()

	require.NoError(t, releaseSecrets(c, []secretusage.Referrer{monitoringcontroller.SecretReferrer()}, releaseRetry{first: time.Millisecond, most: time.Millisecond})(t.Context()))

	stored := &corev1.Secret{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(secret), stored))
	assert.Equal(t, []string{"example.com/keep"}, stored.Finalizers)
	assert.Equal(t, map[string]string{"example.com/note": "x"}, stored.Annotations)
	assert.Equal(t, int32(2), patches.Load(), "retried once")

	storedDst := &monitoringv1alpha1.MonitoringDestination{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(dst), storedDst))
	assert.Equal(t, []string{monitoringv1alpha1.InUseProtectionFinalizer}, storedDst.Finalizers, "only Secrets are released")
}

func TestReleaseSecretsStopsWithTheManager(t *testing.T) {
	t.Parallel()

	c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{
		List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
			return errors.New("connection refused")
		},
	}).Build()
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	// A failure never stops the manager: it retries until the manager stops.
	require.NoError(t, releaseSecrets(c, []secretusage.Referrer{monitoringcontroller.SecretReferrer()}, releaseRetry{first: time.Millisecond, most: 5 * time.Millisecond})(ctx))
}
