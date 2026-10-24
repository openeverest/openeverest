// everest
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

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetObjectsDecodingErrors(t *testing.T) {
	t.Parallel()

	validConfigMap := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
`
	validSecret := `apiVersion: v1
kind: Secret
metadata:
  name: test-secret
stringData:
  password: s3cret
`

	tests := []struct {
		name     string
		input    string
		wantObjs int
		wantErr  bool
	}{
		{
			name:     "malformed YAML returns error",
			input:    ":",
			wantObjs: 0,
			wantErr:  true,
		},
		{
			name:     "valid document followed by malformed YAML returns error",
			input:    validConfigMap + "---\n:",
			wantObjs: 0,
			wantErr:  true,
		},
		{
			name:     "single valid document succeeds",
			input:    validConfigMap,
			wantObjs: 1,
			wantErr:  false,
		},
		{
			name:     "two valid documents both decoded",
			input:    validConfigMap + "---\n" + validSecret,
			wantObjs: 2,
			wantErr:  false,
		},
		{
			name:     "empty input returns no objects and no error",
			input:    "",
			wantObjs: 0,
			wantErr:  false,
		},
	}

	k := NewEmpty(zap.NewNop().Sugar(), "test-ns")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			objs, err := k.getObjects([]byte(tt.input))

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, objs)
			} else {
				require.NoError(t, err)
				assert.Len(t, objs, tt.wantObjs)
			}
		})
	}
}
