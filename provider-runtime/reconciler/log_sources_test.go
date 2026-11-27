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

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestValidLogs(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ProviderSpec{
		ComponentTypes: map[string]v1alpha1.ComponentType{
			"pxc":     {Kind: "mysql"},
			"haproxy": {Kind: "haproxy"},
			"pbm":     {Kind: "percona-backup-mongodb"},
		},
		Components: map[string]v1alpha1.Component{
			"engine": {Type: "pxc"},
			"proxy":  {Type: "haproxy"},
			"backup": {Type: "pbm"},
			"hidden": {Type: "unkinded"},
		},
		Monitoring: &v1alpha1.ProviderMonitoring{
			ComponentTypes: map[string]v1alpha1.MonitoringComponentType{
				"pxc":      {Logs: v1alpha1.LogsSourceSidecar},
				"haproxy":  {Metrics: v1alpha1.MetricsSourceNative},
				"unkinded": {Logs: v1alpha1.LogsSourceNative},
			},
		},
	}
	in := &v1alpha1.Instance{}
	in.Name = "orders"

	valid := v1alpha1.LogStream{
		Component:      "engine",
		Kind:           "mysql",
		PodSelector:    map[string]string{controller.InstanceLabel: "orders", controller.ComponentLabel: "engine"},
		Container:      "pxc",
		Type:           "error",
		Format:         v1alpha1.LogFormatText,
		MultilineStart: `^\d{4}-\d{2}-\d{2}T`,
	}
	with := func(mutate func(*v1alpha1.LogStream)) v1alpha1.LogStream {
		ls := *valid.DeepCopy()
		mutate(&ls)
		return ls
	}

	tests := []struct {
		name    string
		streams []v1alpha1.LogStream
		want    []v1alpha1.LogStream
	}{
		{
			name:    "valid stream is kept",
			streams: []v1alpha1.LogStream{valid},
			want:    []v1alpha1.LogStream{valid},
		},
		{
			name:    "empty kind is set to the declared kind of the component type",
			streams: []v1alpha1.LogStream{with(func(ls *v1alpha1.LogStream) { ls.Kind = "" })},
			want:    []v1alpha1.LogStream{valid},
		},
		{
			name:    "kind other than the declared kind of the component type",
			streams: []v1alpha1.LogStream{valid, with(func(ls *v1alpha1.LogStream) { ls.Kind = "mongodb" })},
			want:    []v1alpha1.LogStream{valid},
		},
		{
			name:    "component type declared without logs",
			streams: []v1alpha1.LogStream{valid, with(func(ls *v1alpha1.LogStream) { ls.Component = "proxy" })},
			want:    []v1alpha1.LogStream{valid},
		},
		{
			name:    "undeclared component type",
			streams: []v1alpha1.LogStream{valid, with(func(ls *v1alpha1.LogStream) { ls.Component = "backup" })},
			want:    []v1alpha1.LogStream{valid},
		},
		{
			name:    "component type declared without a kind",
			streams: []v1alpha1.LogStream{valid, with(func(ls *v1alpha1.LogStream) { ls.Component = "hidden" })},
			want:    []v1alpha1.LogStream{valid},
		},
		{
			name: "pod selector without the Instance label",
			streams: []v1alpha1.LogStream{with(func(ls *v1alpha1.LogStream) {
				ls.PodSelector = map[string]string{controller.InstanceLabel: "other", controller.ComponentLabel: "engine"}
			})},
		},
		{
			name:    "no container",
			streams: []v1alpha1.LogStream{with(func(ls *v1alpha1.LogStream) { ls.Container = "" })},
		},
		{
			name:    "unsupported format",
			streams: []v1alpha1.LogStream{with(func(ls *v1alpha1.LogStream) { ls.Format = "logfmt" })},
		},
		{
			name:    "multilineStart that does not compile",
			streams: []v1alpha1.LogStream{with(func(ls *v1alpha1.LogStream) { ls.MultilineStart = `^(\d` })},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, validLogs(in, spec, tt.streams, logr.Discard()))
		})
	}
}
