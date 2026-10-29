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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

func reconcileClass(t *testing.T, c client.Client, name string) *monitoringv1alpha1.MonitoringClass {
	t.Helper()
	key := types.NamespacedName{Name: name}
	_, err := (&ClassCompatibilityReconciler{Client: c}).Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	class := &monitoringv1alpha1.MonitoringClass{}
	require.NoError(t, c.Get(t.Context(), key, class))
	return class
}

func TestClassCompatibilityMatrix(t *testing.T) {
	t.Parallel()

	oldAgent := pmmClass()
	oldAgent.Spec.ProviderManaged.AgentVersion = "2.44.0"
	outOfRange := func(provider string) monitoringv1alpha1.ProviderCompatibility {
		return monitoringv1alpha1.ProviderCompatibility{
			Name:    provider,
			Reason:  monitoringv1alpha1.ReasonIncompatible,
			Message: `MonitoringClass "pmm" agentVersion 2.44.0 is outside the range ">=3.0.0 <4.0.0" Provider "` + provider + `" supports`,
		}
	}

	basicAuthPMM := pmmClass()
	basicAuthPMM.Spec.AuthModes = []common.AuthMode{{Name: "BasicAuth", Credentials: &common.CredentialKeys{Required: []string{"username", "password"}}}}
	noMode := func(provider string) monitoringv1alpha1.ProviderCompatibility {
		return monitoringv1alpha1.ProviderCompatibility{
			Name:    provider,
			Reason:  monitoringv1alpha1.ReasonAuthModeUnsupported,
			Message: `Provider "` + provider + `" renders none of the authentication modes of MonitoringClass "pmm" for integration "pmm"`,
		}
	}

	tests := []struct {
		name  string
		class *monitoringv1alpha1.MonitoringClass
		want  []monitoringv1alpha1.ProviderCompatibility
	}{
		{
			name:  "ProviderManaged class",
			class: pmmClass(),
			want: []monitoringv1alpha1.ProviderCompatibility{
				{
					Name:    "mariadb",
					Reason:  monitoringv1alpha1.ReasonIntegrationUnsupported,
					Message: `Provider "mariadb" declares no monitoring contract`,
				},
				{Name: "percona-postgresql", Mode: corev1alpha1.MonitoringExecutionModeProviderManaged, Disruption: monitoringv1alpha1.DisruptionRollingRestart},
				{Name: "percona-server-mongodb", Mode: corev1alpha1.MonitoringExecutionModeProviderManaged, Disruption: monitoringv1alpha1.DisruptionRollingRestart},
				{Name: "percona-xtradb-cluster", Mode: corev1alpha1.MonitoringExecutionModeProviderManaged, Disruption: monitoringv1alpha1.DisruptionRollingRestart},
				{
					Name:    "provider-kserve",
					Reason:  monitoringv1alpha1.ReasonIntegrationUnsupported,
					Message: `Provider "provider-kserve" does not declare integration "pmm"`,
				},
			},
		},
		{
			name:  "ProviderManaged class with an agent outside the providers' range",
			class: oldAgent,
			want: []monitoringv1alpha1.ProviderCompatibility{
				{
					Name:    "mariadb",
					Reason:  monitoringv1alpha1.ReasonIntegrationUnsupported,
					Message: `Provider "mariadb" declares no monitoring contract`,
				},
				outOfRange("percona-postgresql"),
				outOfRange("percona-server-mongodb"),
				outOfRange("percona-xtradb-cluster"),
				{
					Name:    "provider-kserve",
					Reason:  monitoringv1alpha1.ReasonIntegrationUnsupported,
					Message: `Provider "provider-kserve" does not declare integration "pmm"`,
				},
			},
		},
		{
			name:  "ProviderManaged class whose modes no provider renders",
			class: basicAuthPMM,
			want: []monitoringv1alpha1.ProviderCompatibility{
				{
					Name:    "mariadb",
					Reason:  monitoringv1alpha1.ReasonIntegrationUnsupported,
					Message: `Provider "mariadb" declares no monitoring contract`,
				},
				noMode("percona-postgresql"),
				noMode("percona-server-mongodb"),
				noMode("percona-xtradb-cluster"),
				{
					Name:    "provider-kserve",
					Reason:  monitoringv1alpha1.ReasonIntegrationUnsupported,
					Message: `Provider "provider-kserve" does not declare integration "pmm"`,
				},
			},
		},
		{
			name:  "ExtensionManaged class",
			class: prometheusClass(),
			want: []monitoringv1alpha1.ProviderCompatibility{
				{
					Name:    "mariadb",
					Reason:  monitoringv1alpha1.ReasonIncompatible,
					Message: `Provider "mariadb" declares no monitoring contract`,
				},
				{
					Name:    "percona-postgresql",
					Reason:  monitoringv1alpha1.ReasonIncompatible,
					Message: `no component satisfies the requirements of MonitoringClass "prometheus"`,
				},
				// mongod and mongos both run an exporter sidecar.
				{Name: "percona-server-mongodb", Mode: corev1alpha1.MonitoringExecutionModeExtensionManaged, Disruption: monitoringv1alpha1.DisruptionRollingRestart},
				// haproxy is native, pxc needs an exporter: the worst case wins.
				{Name: "percona-xtradb-cluster", Mode: corev1alpha1.MonitoringExecutionModeExtensionManaged, Disruption: monitoringv1alpha1.DisruptionRollingRestart},
				{Name: "provider-kserve", Mode: corev1alpha1.MonitoringExecutionModeExtensionManaged, Disruption: monitoringv1alpha1.DisruptionNone},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newFakeClient(t, tt.class, pxcProvider(), psmdbProvider(), kserveProvider(), pgProvider(),
				newProvider("mariadb", map[string]string{"engine": "mariadb"}, nil),
			)

			class := reconcileClass(t, c, tt.class.Name)

			assert.Equal(t, tt.want, class.Status.Providers)
		})
	}
}

func TestClassCompatibilityKeepsClassControllerCondition(t *testing.T) {
	t.Parallel()
	unclaimed := pmmClass()
	unclaimed.Status = monitoringv1alpha1.MonitoringClassStatus{}
	c := newFakeClient(t, unclaimed, pxcProvider())
	pmmController := rtmonitoring.ClassController{Client: c, FieldOwner: "openeverest.io/monitoring-pmm", ClassName: "pmm"}

	reconcileClass(t, c, "pmm")
	require.NoError(t, pmmController.Claim(t.Context()))

	require.NoError(t, c.Create(t.Context(), kserveProvider()))
	class := reconcileClass(t, c, "pmm")

	accepted := findAccepted(t, class.Status.Conditions)
	assert.Equal(t, metav1.ConditionTrue, accepted.Status)
	assert.Equal(t, "claimed by openeverest.io/monitoring-pmm", accepted.Message)
	assert.Equal(t, []monitoringv1alpha1.ProviderCompatibility{
		{Name: "percona-xtradb-cluster", Mode: corev1alpha1.MonitoringExecutionModeProviderManaged, Disruption: monitoringv1alpha1.DisruptionRollingRestart},
		{
			Name:    "provider-kserve",
			Reason:  monitoringv1alpha1.ReasonIntegrationUnsupported,
			Message: `Provider "provider-kserve" does not declare integration "pmm"`,
		},
	}, class.Status.Providers,
	)
}

func TestClassCompatibilityDropsDeletedProviders(t *testing.T) {
	t.Parallel()
	c := newFakeClient(t, prometheusClass(), pxcProvider(), kserveProvider())
	require.Len(t, reconcileClass(t, c, "prometheus").Status.Providers, 2)

	require.NoError(t, c.Delete(t.Context(), kserveProvider()))
	class := reconcileClass(t, c, "prometheus")
	require.Len(t, class.Status.Providers, 1)
	assert.Equal(t, "percona-xtradb-cluster", class.Status.Providers[0].Name)
	assert.Equal(t, metav1.ConditionTrue, findAccepted(t, class.Status.Conditions).Status)

	require.NoError(t, c.Delete(t.Context(), pxcProvider()))
	class = reconcileClass(t, c, "prometheus")
	assert.Empty(t, class.Status.Providers)
	assert.Equal(t, metav1.ConditionTrue, findAccepted(t, class.Status.Conditions).Status)
}

func TestClassCompatibilityEnqueuesEveryClass(t *testing.T) {
	t.Parallel()
	c := newFakeClient(t, pmmClass(), prometheusClass(), victoriaMetricsClass())

	requests := (&ClassCompatibilityReconciler{Client: c}).enqueueAllClasses(t.Context(), pxcProvider())

	assert.ElementsMatch(t, []reconcile.Request{
		{NamespacedName: types.NamespacedName{Name: "pmm"}},
		{NamespacedName: types.NamespacedName{Name: "prometheus"}},
		{NamespacedName: types.NamespacedName{Name: "victoriametrics"}},
	}, requests,
	)
}
