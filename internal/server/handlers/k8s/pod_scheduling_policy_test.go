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

package k8s

import (
	"context"
	"slices"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/pkg/kubernetes"
)

func getDefaultPXCPolicy() *everestv1alpha1.PodSchedulingPolicy {
	return &everestv1alpha1.PodSchedulingPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "everest-default-mysql",
			Finalizers: []string{everestv1alpha1.ReadOnlyFinalizer},
		},
		Spec: everestv1alpha1.PodSchedulingPolicySpec{
			EngineType: everestv1alpha1.DatabaseEnginePXC,
			AffinityConfig: &everestv1alpha1.AffinityConfig{
				PXC: &everestv1alpha1.PXCAffinityConfig{
					Engine: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					Proxy: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
		},
	}
}

func getDefaultPGPolicy() *everestv1alpha1.PodSchedulingPolicy {
	return &everestv1alpha1.PodSchedulingPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "everest-default-postgresql",
			Finalizers: []string{everestv1alpha1.ReadOnlyFinalizer},
		},
		Spec: everestv1alpha1.PodSchedulingPolicySpec{
			EngineType: everestv1alpha1.DatabaseEnginePostgresql,
			AffinityConfig: &everestv1alpha1.AffinityConfig{
				PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{
					Engine: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					Proxy: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
		},
	}
}

func getDefaultPSMDBPolicy() *everestv1alpha1.PodSchedulingPolicy {
	return &everestv1alpha1.PodSchedulingPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "everest-default-mongodb",
			Finalizers: []string{everestv1alpha1.ReadOnlyFinalizer},
		},
		Spec: everestv1alpha1.PodSchedulingPolicySpec{
			EngineType: everestv1alpha1.DatabaseEnginePSMDB,
			AffinityConfig: &everestv1alpha1.AffinityConfig{
				PSMDB: &everestv1alpha1.PSMDBAffinityConfig{
					Engine: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					Proxy: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					ConfigServer: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestValidate_ListPodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		objs       []ctrlclient.Object
		listParams *api.ListPodSchedulingPolicyParams
		assert     func(*everestv1alpha1.PodSchedulingPolicyList) bool
		wantErr    error
	}
	testCases := []testCase{
		// policies are absent
		{
			name: "absent policies",
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
		// default policies no filter
		{
			name: "default policies without filter",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 3 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mysql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-postgresql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mongodb"
					})
			},
		},
		// default policies hasRules filter
		{
			name: "default policies hasRules filter",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				HasRules: pointer.To(true),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 3 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mysql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-postgresql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mongodb"
					})
			},
		},
		// default policies PXC filter
		{
			name: "default policies PXC filter",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Pxc),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mysql"
					})
			},
		},
		// default policies PXC hasRules filters
		{
			name: "default policies PXC hasRules filters",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Pxc),
				HasRules:   pointer.To(true),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mysql"
					})
			},
		},
		// default policies PSMDB filter
		{
			name: "default policies PSMDB filter",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Psmdb),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mongodb"
					})
			},
		},
		// default policies PSMDB hasRules filters
		{
			name: "default policies PSMDB hasRules filters",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Psmdb),
				HasRules:   pointer.To(true),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mongodb"
					})
			},
		},
		// default policies PostgreSQL filter
		{
			name: "default policies PostgreSQL filter",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Postgresql),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-postgresql"
					})
			},
		},
		// default policies PostgreSQL hasRules filters
		{
			name: "default policies PostgreSQL hasRules filters",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Postgresql),
				HasRules:   pointer.To(true),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-postgresql"
					})
			},
		},
		// empty policies no filter
		{
			name: "empty policies no filter",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-psmdb",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-postgresql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 3 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "test-pxc"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "test-postgresql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "test-psmdb"
					})
			},
		},
		// empty policies hasRules filter
		{
			name: "empty policies hasRules filter",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-psmdb",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-postgresql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				HasRules: pointer.To(true),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
		// empty policies PXC filter
		{
			name: "empty policies",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-psmdb",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-postgresql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Pxc),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "test-pxc"
					})
			},
		},
		// empty policies PXC hasRules filters
		{
			name: "empty policies PXC hasRules filters",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-psmdb",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-postgresql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Pxc),
				HasRules:   pointer.To(true),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
		// empty policies PSMDB filter
		{
			name: "empty policies PSMDB filter",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-psmdb",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-postgresql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Psmdb),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "test-psmdb"
					})
			},
		},
		// empty policies PSMDB hasRules filters
		{
			name: "empty policies PSMDB hasRules filters",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-psmdb",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-postgresql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Psmdb),
				HasRules:   pointer.To(true),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
		// empty policies PostgreSQL filter
		{
			name: "empty policies PostgreSQL filter",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-psmdb",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-postgresql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Postgresql),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "test-postgresql"
					})
			},
		},
		// empty policies PostgreSQL hasRules filters
		{
			name: "empty policies PostgreSQL hasRules filters",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-psmdb",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-postgresql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			listParams: &api.ListPodSchedulingPolicyParams{
				EngineType: pointer.To(api.Postgresql),
				HasRules:   pointer.To(true),
			},
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(tc.objs...).
				Build()

			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient)
			k8sH := New(zap.NewNop().Sugar(), k, "")

			pspList, err := k8sH.ListPodSchedulingPolicies(context.Background(), tc.listParams)
			require.NoError(t, err)
			assert.Condition(t, func() bool {
				return tc.assert(pspList)
			})
		})
	}
}
