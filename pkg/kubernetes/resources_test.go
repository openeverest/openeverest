// everest
// Copyright (C) 2023 Percona LLC
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetEKSVolumeCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		node       corev1.Node
		want       uint64
		wantErrMsg string
	}{
		{
			name: "z1d uses reduced limit",
			node: eksNode("z1d.large", corev1.ConditionFalse),
			want: 25,
		},
		{
			name: "t3 uses reduced limit",
			node: eksNode("t3.medium", corev1.ConditionFalse),
			want: 25,
		},
		{
			name: "m5 uses reduced limit",
			node: eksNode("m5.xlarge", corev1.ConditionFalse),
			want: 25,
		},
		{
			name: "m4 uses default limit",
			node: eksNode("m4.large", corev1.ConditionFalse),
			want: eksVolumeLimitPerNode,
		},
		{
			name: "disk pressure returns zero",
			node: eksNode("z1d.large", corev1.ConditionTrue),
			want: 0,
		},
		{
			name:       "missing instance type label",
			node:       corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}},
			wantErrMsg: "beta.kubernetes.io/instance-type",
		},
		{
			name:       "invalid instance type format",
			node:       eksNode("invalid", corev1.ConditionFalse),
			wantErrMsg: "failed to parse EKS node type",
		},
	}

	k := &Kubernetes{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := k.getEKSVolumeCount(tt.node)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func eksNode(instanceType string, diskPressure corev1.ConditionStatus) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
			Labels: map[string]string{
				"beta.kubernetes.io/instance-type": instanceType,
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeDiskPressure,
					Status: diskPressure,
				},
			},
		},
	}
}
