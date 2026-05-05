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

package status

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
)

func TestIsDeploymentReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		depl     *appsv1.Deployment
		expected bool
	}{
		{
			name: "Deployment is ready",
			depl: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Status: appsv1.DeploymentStatus{
					Replicas:            1,
					ReadyReplicas:       1,
					UpdatedReplicas:     1,
					UnavailableReplicas: 0,
					ObservedGeneration:  1,
				},
			},
			expected: true,
		},
		{
			name: "Deployment not ready - replicas mismatch",
			depl: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Status: appsv1.DeploymentStatus{
					Replicas:            2,
					ReadyReplicas:       1,
					UpdatedReplicas:     2,
					UnavailableReplicas: 1,
					ObservedGeneration:  1,
				},
			},
			expected: false,
		},
		{
			name: "Deployment not ready - generation mismatch",
			depl: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Status: appsv1.DeploymentStatus{
					Replicas:            1,
					ReadyReplicas:       1,
					UpdatedReplicas:     1,
					UnavailableReplicas: 0,
					ObservedGeneration:  1,
				},
			},
			expected: false,
		},
		{
			name: "Deployment not ready - unavailable replicas",
			depl: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Status: appsv1.DeploymentStatus{
					Replicas:            1,
					ReadyReplicas:       0,
					UpdatedReplicas:     1,
					UnavailableReplicas: 1,
					ObservedGeneration:  1,
				},
			},
			expected: false,
		},
		{
			name: "Deployment ready - multiple replicas",
			depl: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 3},
				Status: appsv1.DeploymentStatus{
					Replicas:            3,
					ReadyReplicas:       3,
					UpdatedReplicas:     3,
					UnavailableReplicas: 0,
					ObservedGeneration:  3,
				},
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isDeploymentReady(tc.depl))
		})
	}
}

func TestCheckDeployment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		deplName  string
		namespace string
		objects   []appsv1.Deployment
		wantReady bool
		wantMsg   string
	}{
		{
			name:      "Deployment found and ready",
			deplName:  "everest-server",
			namespace: "everest-system",
			objects: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "everest-server",
						Namespace:  "everest-system",
						Generation: 1,
					},
					Status: appsv1.DeploymentStatus{
						Replicas:           1,
						ReadyReplicas:      1,
						UpdatedReplicas:    1,
						ObservedGeneration: 1,
					},
				},
			},
			wantReady: true,
		},
		{
			name:      "Deployment found but not ready",
			deplName:  "everest-server",
			namespace: "everest-system",
			objects: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "everest-server",
						Namespace:  "everest-system",
						Generation: 1,
					},
					Status: appsv1.DeploymentStatus{
						Replicas:            2,
						ReadyReplicas:       1,
						UpdatedReplicas:     2,
						UnavailableReplicas: 1,
						ObservedGeneration:  1,
					},
				},
			},
			wantReady: false,
			wantMsg:   "1/2 ready",
		},
		{
			name:      "Deployment not found",
			deplName:  "missing-deployment",
			namespace: "everest-system",
			objects:   []appsv1.Deployment{},
			wantReady: false,
			wantMsg:   "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme())

			for i := range tc.objects {
				builder = builder.WithObjects(&tc.objects[i])
			}

			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).
				WithKubernetesClient(builder.Build())

			s := &Status{
				l:          zap.NewNop().Sugar(),
				kubeClient: k,
			}

			cs := s.checkDeployment(context.Background(), tc.deplName, tc.namespace)
			assert.Equal(t, tc.wantReady, cs.Ready)
			assert.Equal(t, tc.deplName, cs.Name)
			assert.Equal(t, tc.namespace, cs.Namespace)
			if tc.wantMsg != "" {
				assert.Contains(t, cs.Message, tc.wantMsg)
			}
		})
	}
}

func TestCheckCoreComponents(t *testing.T) {
	t.Parallel()

	readyDeployment := func(name, namespace string) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:       name,
				Namespace:  namespace,
				Generation: 1,
			},
			Status: appsv1.DeploymentStatus{
				Replicas:           1,
				ReadyReplicas:      1,
				UpdatedReplicas:    1,
				ObservedGeneration: 1,
			},
		}
	}

	tests := []struct {
		name        string
		objects     []*appsv1.Deployment
		wantHealthy bool
	}{
		{
			name: "All core components healthy",
			objects: []*appsv1.Deployment{
				readyDeployment(common.PerconaEverestDeploymentName, common.SystemNamespace),
				readyDeployment(common.PerconaEverestOperatorDeploymentName, common.SystemNamespace),
			},
			wantHealthy: true,
		},
		{
			name: "Server missing",
			objects: []*appsv1.Deployment{
				readyDeployment(common.PerconaEverestOperatorDeploymentName, common.SystemNamespace),
			},
			wantHealthy: false,
		},
		{
			name:        "Both missing",
			objects:     []*appsv1.Deployment{},
			wantHealthy: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme())
			for _, obj := range tc.objects {
				builder = builder.WithObjects(obj)
			}

			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).
				WithKubernetesClient(builder.Build())

			s := &Status{
				l:          zap.NewNop().Sugar(),
				kubeClient: k,
			}

			result := &OverallStatus{Healthy: true}
			s.checkCoreComponents(context.Background(), result)

			assert.Equal(t, tc.wantHealthy, result.Healthy)
			// Should always check 2 core components.
			assert.Len(t, result.Components, 2)
		})
	}
}

func TestGetManagedNamespaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		namespaces     []*corev1.Namespace
		wantNamespaces []string
	}{
		{
			name: "DB namespaces found",
			namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "everest",
						Labels: map[string]string{common.KubernetesManagedByLabel: common.Everest},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "prod-db",
						Labels: map[string]string{common.KubernetesManagedByLabel: common.Everest},
					},
				},
				// System namespace should be filtered out.
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   common.SystemNamespace,
						Labels: map[string]string{common.KubernetesManagedByLabel: common.Everest},
					},
				},
				// Monitoring namespace should be filtered out.
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   common.MonitoringNamespace,
						Labels: map[string]string{common.KubernetesManagedByLabel: common.Everest},
					},
				},
			},
			wantNamespaces: []string{"everest", "prod-db"},
		},
		{
			name:           "No DB namespaces",
			namespaces:     []*corev1.Namespace{},
			wantNamespaces: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme())
			for _, ns := range tc.namespaces {
				builder = builder.WithObjects(ns)
			}

			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).
				WithKubernetesClient(builder.Build())

			s := &Status{
				l:          zap.NewNop().Sugar(),
				kubeClient: k,
			}

			result := &OverallStatus{}
			s.getManagedNamespaces(context.Background(), result)
			assert.Equal(t, tc.wantNamespaces, result.Namespaces)
		})
	}
}

func TestRun_Healthy(t *testing.T) {
	t.Parallel()

	// Create a minimal healthy cluster state.
	builder := fakeclient.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithObjects(
			// Core deployments.
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:       common.PerconaEverestDeploymentName,
					Namespace:  common.SystemNamespace,
					Generation: 1,
					Labels:     map[string]string{"app.kubernetes.io/name": "everest-server"},
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app.kubernetes.io/name": "everest-server"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app.kubernetes.io/name": "everest-server"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "everest", Image: "percona/everest:1.0.0"}},
						},
					},
				},
				Status: appsv1.DeploymentStatus{
					Replicas:           1,
					ReadyReplicas:      1,
					UpdatedReplicas:    1,
					ObservedGeneration: 1,
				},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:       common.PerconaEverestOperatorDeploymentName,
					Namespace:  common.SystemNamespace,
					Generation: 1,
				},
				Status: appsv1.DeploymentStatus{
					Replicas:           1,
					ReadyReplicas:      1,
					UpdatedReplicas:    1,
					ObservedGeneration: 1,
				},
			},
			// Monitoring deployments.
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:       common.VictoriaMetricsOperatorDeploymentName,
					Namespace:  common.MonitoringNamespace,
					Generation: 1,
				},
				Status: appsv1.DeploymentStatus{
					Replicas:           1,
					ReadyReplicas:      1,
					UpdatedReplicas:    1,
					ObservedGeneration: 1,
				},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:       common.KubeStateMetricsDeploymentName,
					Namespace:  common.MonitoringNamespace,
					Generation: 1,
				},
				Status: appsv1.DeploymentStatus{
					Replicas:           1,
					ReadyReplicas:      1,
					UpdatedReplicas:    1,
					ObservedGeneration: 1,
				},
			},
			// DB namespace.
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "everest",
					Labels: map[string]string{common.KubernetesManagedByLabel: common.Everest},
				},
			},
		)

	k := kubernetes.NewEmpty(zap.NewNop().Sugar()).
		WithKubernetesClient(builder.Build())

	s := &Status{
		l:          zap.NewNop().Sugar(),
		cfg:        StatusConfig{JSON: true},
		kubeClient: k,
	}

	err := s.Run(context.Background())
	require.NoError(t, err)
}

func TestRun_Unhealthy(t *testing.T) {
	t.Parallel()

	// Create a cluster with a missing operator deployment.
	builder := fakeclient.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithObjects(
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:       common.PerconaEverestDeploymentName,
					Namespace:  common.SystemNamespace,
					Generation: 1,
					Labels:     map[string]string{"app.kubernetes.io/name": "everest-server"},
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app.kubernetes.io/name": "everest-server"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app.kubernetes.io/name": "everest-server"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "everest", Image: "percona/everest:1.0.0"}},
						},
					},
				},
				Status: appsv1.DeploymentStatus{
					Replicas:           1,
					ReadyReplicas:      1,
					UpdatedReplicas:    1,
					ObservedGeneration: 1,
				},
			},
			// everest-operator is MISSING - should cause unhealthy status
		)

	k := kubernetes.NewEmpty(zap.NewNop().Sugar()).
		WithKubernetesClient(builder.Build())

	s := &Status{
		l:          zap.NewNop().Sugar(),
		cfg:        StatusConfig{JSON: true},
		kubeClient: k,
	}

	err := s.Run(context.Background())
	require.NoError(t, err)
}
