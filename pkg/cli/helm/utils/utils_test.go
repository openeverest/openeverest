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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/cli/values"
)

func TestMergeVals(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		flagsOpt values.Options
		mapOpt   map[string]string
		merged   map[string]any
	}{
		{
			flagsOpt: values.Options{
				Values: []string{
					"key1=val1",
					"key2.enabled=true",
				},
			},
			merged: map[string]any{
				"key1": "val1",
				"key2": map[string]any{
					"enabled": true,
				},
			},
		},
		{
			flagsOpt: values.Options{
				Values: []string{
					"key1=val1",
					"key2.enabled=true",
				},
			},
			mapOpt: map[string]string{
				"key2.enabled": "false",
			},
			merged: map[string]any{
				"key1": "val1",
				"key2": map[string]any{
					"enabled": true,
				},
			},
		},
		{
			flagsOpt: values.Options{
				Values: []string{
					"key1=val1",
					"key2.enabled=true",
				},
			},
			mapOpt: map[string]string{
				"key3.subkey": "value3",
			},
			merged: map[string]any{
				"key1": "val1",
				"key2": map[string]any{
					"enabled": true,
				},
				"key3": map[string]any{
					"subkey": "value3",
				},
			},
		},
	}

	for _, tc := range testCases {
		merged, err := MergeVals(tc.flagsOpt, tc.mapOpt)
		require.NoError(t, err)
		assert.Equal(t, tc.merged, merged)
	}
}
