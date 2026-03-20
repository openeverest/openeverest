// everest
// Copyright (C) 2023 Percona LLC
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

// Package kubernetes provides functionality for kubernetes.
package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeNamesspacesEnvVar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ns1  string
		ns2  string
		want string
	}{
		{
			name: "0-0",
			ns1:  "",
			ns2:  "",
			want: "",
		},
		{
			name: "1-0",
			ns1:  "ns1",
			ns2:  "",
			want: "ns1",
		},
		{
			name: "0-1",
			ns1:  "",
			ns2:  "ns2",
			want: "ns2",
		},
		{
			name: "1-1",
			ns1:  "ns1",
			ns2:  "ns2",
			want: "ns1,ns2",
		},
		{
			name: "1-2",
			ns1:  "ns1",
			ns2:  "ns2,ns3",
			want: "ns1,ns2,ns3",
		},
		{
			name: "1-2 unsorted",
			ns1:  "ns2",
			ns2:  "ns3,ns1",
			want: "ns1,ns2,ns3",
		},
		{
			name: "1-2 extra commas",
			ns1:  ",ns1,",
			ns2:  ",,ns2,,,,ns3,,,",
			want: "ns1,ns2,ns3",
		},
		{
			name: "2-2",
			ns1:  "ns4,ns1",
			ns2:  "ns2,ns3",
			want: "ns1,ns2,ns3,ns4",
		},
		{
			name: "0-2",
			ns1:  "",
			ns2:  "ns2,ns3",
			want: "ns2,ns3",
		},
		{
			name: "2-0 unsorted",
			ns1:  "ns4,ns1",
			ns2:  "",
			want: "ns1,ns4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mergedNS := mergeNamespacesEnvVar(tt.ns1, tt.ns2)
			assert.Equal(t, tt.want, mergedNS)
		})
	}
}
