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

package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterObjectName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cluster string
		name    string
		want    string
	}{
		{"prod", "psmdb", "prod/psmdb"},
		{"staging", "pxc", "staging/pxc"},
		{"my-cluster", "my-provider", "my-cluster/my-provider"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ClusterObjectName(tc.cluster, tc.name))
		})
	}
}

func TestClusterNamespacedObjectName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cluster   string
		namespace string
		name      string
		want      string
	}{
		{"prod", "default", "my-db", "prod/default/my-db"},
		{"staging", "team-a", "db1", "staging/team-a/db1"},
		{"us-east-1", "ns1", "instance-xyz", "us-east-1/ns1/instance-xyz"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ClusterNamespacedObjectName(tc.cluster, tc.namespace, tc.name))
		})
	}
}

func TestIsGlobalResource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		resource string
		want     bool
	}{
		{ResourceClusters, true},
		{ResourceNamespaces, true},
		{ResourcePodSchedulingPolicies, true},
		{ResourceLoadBalancerConfigs, true},
		{ResourceDataImporters, true},
		{ResourceInstances, false},
		{ResourceProviders, false},
		{ResourceBackupClasses, false},
		{ResourceBackups, false},
		{ResourceDatabaseClusters, false},
		{"unknown", false},
	}
	for _, tc := range tests {
		t.Run(tc.resource, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, IsGlobalResource(tc.resource))
		})
	}
}

func TestIsClusterScopedResource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		resource string
		want     bool
	}{
		{ResourceProviders, true},
		{ResourceBackupClasses, true},
		{ResourceInstances, false},
		{ResourceClusters, false},
		{ResourceBackupStorages, false},
		{ResourceDatabaseClusters, false},
		{"unknown", false},
	}
	for _, tc := range tests {
		t.Run(tc.resource, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, IsClusterScopedResource(tc.resource))
		})
	}
}

func TestIsClusterNamespacedResource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		resource string
		want     bool
	}{
		{ResourceInstances, true},
		{ResourceBackups, true},
		{ResourceRestores, true},
		{ResourceBackupStorages, true},
		{ResourceMonitoringConfigs, true},
		{ResourceProviders, false},
		{ResourceBackupClasses, false},
		{ResourceClusters, false},
		{ResourceDatabaseClusters, false},
		{"unknown", false},
	}
	for _, tc := range tests {
		t.Run(tc.resource, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, IsClusterNamespacedResource(tc.resource))
		})
	}
}
