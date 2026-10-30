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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestValidConnections(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ProviderSpec{
		Components: map[string]v1alpha1.Component{
			"engine": {Type: "mongod"},
			"proxy":  {Type: "mongos"},
			"backup": {Type: "pbm"},
		},
		Monitoring: &v1alpha1.ProviderMonitoring{
			ComponentTypes: map[string]v1alpha1.MonitoringComponentType{
				"mongod": {Kind: "mongodb", Connection: v1alpha1.ConnectionSupported},
				"mongos": {Kind: "mongodb"},
			},
		},
	}
	in := &v1alpha1.Instance{}
	in.Name = "orders"
	monitor := []v1alpha1.MonitoringCredential{{Profile: v1alpha1.CredentialProfileMonitor, SecretRef: common.SecretRef{Name: "orders-monitor"}}}

	valid := v1alpha1.ConnectionEndpoint{
		Component:   "engine",
		Kind:        "mongodb",
		PodSelector: map[string]string{controller.InstanceLabel: "orders", controller.ComponentLabel: "engine"},
		Container:   "mongod",
		Port:        intstr.FromInt32(27017),
		TLS: &v1alpha1.ConnectionTLS{
			CA:                         &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: "orders-ssl", Key: "ca.crt"}},
			ClientCertificateSecretRef: &common.SecretRef{Name: "orders-ssl"},
		},
		Auth: &v1alpha1.MetricsEndpointAuth{CredentialProfile: v1alpha1.CredentialProfileMonitor},
	}
	with := func(mutate func(*v1alpha1.ConnectionEndpoint)) v1alpha1.ConnectionEndpoint {
		ep := *valid.DeepCopy()
		mutate(&ep)
		return ep
	}

	tests := []struct {
		name        string
		credentials []v1alpha1.MonitoringCredential
		connections []v1alpha1.ConnectionEndpoint
		want        []v1alpha1.ConnectionEndpoint
	}{
		{
			name:        "valid endpoint is kept",
			credentials: monitor,
			connections: []v1alpha1.ConnectionEndpoint{valid},
			want:        []v1alpha1.ConnectionEndpoint{valid},
		},
		{
			name:        "kind comes from the declared component type",
			credentials: monitor,
			connections: []v1alpha1.ConnectionEndpoint{with(func(ep *v1alpha1.ConnectionEndpoint) { ep.Kind = "mysql" })},
			want:        []v1alpha1.ConnectionEndpoint{valid},
		},
		{
			name:        "component type declared without connections",
			credentials: monitor,
			connections: []v1alpha1.ConnectionEndpoint{valid, with(func(ep *v1alpha1.ConnectionEndpoint) { ep.Component = "proxy" })},
			want:        []v1alpha1.ConnectionEndpoint{valid},
		},
		{
			name:        "undeclared component type",
			credentials: monitor,
			connections: []v1alpha1.ConnectionEndpoint{valid, with(func(ep *v1alpha1.ConnectionEndpoint) { ep.Component = "backup" })},
			want:        []v1alpha1.ConnectionEndpoint{valid},
		},
		{
			name:        "pod selector without the Instance label",
			credentials: monitor,
			connections: []v1alpha1.ConnectionEndpoint{with(func(ep *v1alpha1.ConnectionEndpoint) {
				ep.PodSelector = map[string]string{controller.ComponentLabel: "engine"}
			})},
		},
		{
			name:        "unpublished credential profile",
			connections: []v1alpha1.ConnectionEndpoint{valid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, validConnections(in, spec, tt.credentials, tt.connections, logr.Discard()))
		})
	}
}
