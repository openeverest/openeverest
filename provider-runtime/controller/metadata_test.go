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
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func TestEffectiveVersionBundleName(t *testing.T) {
	t.Parallel()

	withBundles := &v1alpha1.ProviderSpec{
		DefaultVersion: "2.0.0",
		Versions: []v1alpha1.VersionBundle{
			{Name: "1.0.0"},
			{Name: "2.0.0"},
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

func TestGetDefaultImageForComponent(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ProviderSpec{
		Components: map[string]v1alpha1.Component{"engine": {Type: "mongod"}},
		ComponentTypes: map[string]v1alpha1.ComponentType{
			"mongod": {
				DefaultVersion: "8.0.12-4",
				Versions: []v1alpha1.ComponentVersion{
					{Version: "7.0.18-11", Image: "psmdb:7.0.18-11"},
					{Version: "8.0.12-4", Image: "psmdb:8.0.12-4"},
				},
			},
		},
	}

	assert.Equal(t, "psmdb:8.0.12-4", GetDefaultImageForComponent(spec, "engine"))
}

func TestValidateProviderSpecDefaultVersion(t *testing.T) {
	t.Parallel()

	valid := func() *v1alpha1.ProviderSpec {
		return &v1alpha1.ProviderSpec{
			Components: map[string]v1alpha1.Component{"engine": {Type: "mongod"}},
			ComponentTypes: map[string]v1alpha1.ComponentType{
				"mongod": {
					DefaultVersion: "8.0.12-4",
					Versions:       []v1alpha1.ComponentVersion{{Version: "8.0.12-4"}},
				},
			},
			DefaultVersion: "8.0.12",
			Versions: []v1alpha1.VersionBundle{
				{Name: "8.0.12", Components: map[string]string{"engine": "8.0.12-4"}},
			},
		}
	}

	tests := []struct {
		name    string
		mutate  func(*v1alpha1.ProviderSpec)
		wantErr string
	}{
		{
			name:   "defaults naming existing entries",
			mutate: func(*v1alpha1.ProviderSpec) {},
		},
		{
			name: "no defaults declared",
			mutate: func(s *v1alpha1.ProviderSpec) {
				s.DefaultVersion = ""
				s.ComponentTypes["mongod"] = v1alpha1.ComponentType{Versions: s.ComponentTypes["mongod"].Versions}
			},
		},
		{
			name:    "defaultVersion names no bundle",
			mutate:  func(s *v1alpha1.ProviderSpec) { s.DefaultVersion = "9.0.0" },
			wantErr: `defaultVersion: version bundle "9.0.0" not found`,
		},
		{
			name: "component type defaultVersion names no catalog entry",
			mutate: func(s *v1alpha1.ProviderSpec) {
				s.ComponentTypes["mongod"] = v1alpha1.ComponentType{
					DefaultVersion: "9.0.0-1",
					Versions:       s.ComponentTypes["mongod"].Versions,
				}
			},
			wantErr: `componentTypes["mongod"]: defaultVersion "9.0.0-1" not found`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := valid()
			tc.mutate(spec)

			err := ValidateProviderSpec(spec)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
