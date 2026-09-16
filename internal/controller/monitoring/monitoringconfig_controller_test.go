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

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

func pmmConfig(name, url string) monitoringv1alpha1.MonitoringConfig {
	return monitoringv1alpha1.MonitoringConfig{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "everest-monitoring"},
		Spec: monitoringv1alpha1.MonitoringConfigSpec{
			Type: monitoringv1alpha1.PMMMonitoringType,
			PMM: &monitoringv1alpha1.PMMMonitoringSpec{
				URL:                  url,
				CredentialsSecretRef: common.SecretRef{Name: name + "-secret"},
			},
		},
	}
}

func TestGenVMAgentSpec_DeterministicRemoteWriteOrder(t *testing.T) {
	t.Parallel()

	r := &MonitoringConfigReconciler{MonitoringNamespace: "everest-monitoring"}
	mc1 := pmmConfig("test-mc-1", "http://localhost-test-mc-1-test-1")
	mc2 := pmmConfig("test-mc-2", "http://localhost-test-mc-2-test-2")

	// The informer cache returns lists in no particular order; the generated
	// spec must not depend on it.
	forward, err := r.genVMAgentSpec(&monitoringv1alpha1.MonitoringConfigList{
		Items: []monitoringv1alpha1.MonitoringConfig{mc1, mc2},
	}, "cluster-id")
	require.NoError(t, err)
	reverse, err := r.genVMAgentSpec(&monitoringv1alpha1.MonitoringConfigList{
		Items: []monitoringv1alpha1.MonitoringConfig{mc2, mc1},
	}, "cluster-id")
	require.NoError(t, err)

	require.Len(t, forward.RemoteWrite, 2)
	assert.Equal(t, "http://localhost-test-mc-1-test-1/victoriametrics/api/v1/write", forward.RemoteWrite[0].URL)
	assert.Equal(t, "http://localhost-test-mc-2-test-2/victoriametrics/api/v1/write", forward.RemoteWrite[1].URL)
	assert.Equal(t, forward.RemoteWrite, reverse.RemoteWrite)
}
