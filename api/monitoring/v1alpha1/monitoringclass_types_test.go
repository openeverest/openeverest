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

package v1alpha1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

func TestPodMetadataPolicyAllowsAnnotation(t *testing.T) {
	t.Parallel()

	policy := &monitoringv1alpha1.PodMetadataPolicy{AnnotationPrefixes: []string{"coroot.com/", "ad.datadoghq.com/"}}
	tests := []struct {
		name   string
		policy *monitoringv1alpha1.PodMetadataPolicy
		key    string
		want   bool
	}{
		{name: "key under a prefix", policy: policy, key: "coroot.com/mongodb-scrape", want: true},
		{name: "key under the second prefix", policy: policy, key: "ad.datadoghq.com/mongod.checks", want: true},
		{name: "bare prefix", policy: policy, key: "coroot.com/", want: false},
		{name: "other prefix", policy: policy, key: "sidecar.istio.io/inject", want: false},
		{name: "prefix without the slash", policy: policy, key: "coroot.community/x", want: false},
		{name: "no policy", policy: nil, key: "coroot.com/mongodb-scrape", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.policy.AllowsAnnotation(tt.key))
		})
	}
}
