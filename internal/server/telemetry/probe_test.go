// everest
// Copyright (C) 2025 Percona LLC
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

package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoolStr(t *testing.T) {
	assert.Equal(t, "true", boolStr(true))
	assert.Equal(t, "false", boolStr(false))
}

func TestJoinSortedSet(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]struct{}
		want string
	}{
		{"empty", map[string]struct{}{}, ""},
		{"single", map[string]struct{}{"a": {}}, "a"},
		{"sorted_dedup", map[string]struct{}{"c": {}, "a": {}, "b": {}}, "a,b,c"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, joinSortedSet(tc.in))
		})
	}
}

func TestProvidersProbe_EmitsReservedKey(t *testing.T) {
	got, err := providersProbe{}.Collect(t.Context(), nil)
	assert.NoError(t, err)
	v, ok := got["providers_installed"]
	assert.True(t, ok, "providers_installed key must be reserved in v1")
	assert.Equal(t, "", v, "v1 must emit an empty value")
}
