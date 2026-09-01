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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func TestEffectiveVersionBundleName(t *testing.T) {
	t.Parallel()

	withBundles := &v1alpha1.ProviderSpec{
		Versions: []v1alpha1.VersionBundle{
			{Name: "1.0.0"},
			{Name: "2.0.0", Default: true},
		},
	}

	tests := []struct {
		name          string
		spec          *v1alpha1.ProviderSpec
		specVersion   string
		statusVersion string
		want          string
	}{
		{
			name:          "spec.version outranks both status.version and the default",
			spec:          withBundles,
			specVersion:   "1.0.0",
			statusVersion: "2.0.0",
			want:          "1.0.0",
		},
		{
			name:          "status.version outranks the default, so a new default does not move an existing Instance",
			spec:          withBundles,
			statusVersion: "1.0.0",
			want:          "1.0.0",
		},
		{
			name: "the default bundle applies on the first reconcile",
			spec: withBundles,
			want: "2.0.0",
		},
		{
			name: "no bundle in force when the spec declares no default",
			spec: &v1alpha1.ProviderSpec{
				Versions: []v1alpha1.VersionBundle{{Name: "1.0.0"}},
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			in := &v1alpha1.Instance{
				Spec:   v1alpha1.InstanceSpec{Version: tc.specVersion},
				Status: v1alpha1.InstanceStatus{Version: tc.statusVersion},
			}
			assert.Equal(t, tc.want, EffectiveVersionBundleName(tc.spec, in))
		})
	}
}
