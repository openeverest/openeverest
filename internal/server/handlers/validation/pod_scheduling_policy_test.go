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

package validation

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	operatorUtils "github.com/percona/everest-operator/utils"
	"github.com/percona/everest/internal/server/handlers/k8s"
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
		},
	}
}

func TestValidate_CreatePodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name           string
		objs           []ctrlclient.Object
		policyToCreate *everestv1alpha1.PodSchedulingPolicy
		wantErr        error
	}

	testCases := []testCase{
		// invalid PodSchedulingPolicy names
		{
			name: "empty podSchedulingPolicy name",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "",
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, operatorUtils.ErrNameNotRFC1035Compatible("metadata.name")),
		},
		{
			name: "name starts with -",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "-rstrst",
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, operatorUtils.ErrNameNotRFC1035Compatible("metadata.name")),
		},
		{
			name: "name ends with -",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "rstrst-",
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, operatorUtils.ErrNameNotRFC1035Compatible("metadata.name")),
		},
		{
			name: "name contains uppercase",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "AAsdf",
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, operatorUtils.ErrNameNotRFC1035Compatible("metadata.name")),
		},
		{
			name: "name too long",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "asldkafaslkdjfalskdfjaslkdjflsakfjdalskfdjaslkfdjaslkfdjsaklfdassksjdfhskdjfskjdfsdfsdflasdkfasdfk",
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, operatorUtils.ErrNameNotRFC1035Compatible("metadata.name")),
		},
		{
			name: "duplicate name",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "everest-existing-name",
					},
				},
			},
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "everest-existing-name",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: k8sError.NewAlreadyExists(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "podschedulingpolicies",
			},
				"everest-existing-name",
			),
		},
		// unsupported engineType
		{
			name: "unsupported engineType",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "unknown-engine-type",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: "unsupported",
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPEngineType("unsupported")),
		},
		// empty affinity configs
		{
			name: "empty PXC affinity config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-affinity",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType:     everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPXCEmpty),
		},
		{
			name: "empty PSMDB affinity config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-affinity",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType:     everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPSMDBEmpty),
		},
		{
			name: "empty Postgresql affinity config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-affinity",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType:     everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPostgresqlEmpty),
		},
		// empty DB components affinity configs
		{
			name: "empty PXC components config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-pxc-components",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPXCComponentsEmpty),
		},
		{
			name: "empty PSMDB components config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-psmdb-components",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPSMDBComponentsEmpty),
		},
		{
			name: "empty PostgreSQL components config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-pg-components",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPostgresqlComponentsEmpty),
		},
		// affinity config mismatches with engineType
		{
			name: "PXC affinity config mismatch PSMDB",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pxc-mismatch-psmdb",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{
							Engine: &corev1.Affinity{},
						},
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPXCWithPSMDB),
		},
		{
			name: "PXC affinity config mismatch PostgreSQL",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pxc-mismatch-pg",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{
							Engine: &corev1.Affinity{},
						},
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPXCWithPostgresql),
		},
		{
			name: "PSMDB affinity config mismatch PXC",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "psmdb-mismatch-pxc",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{},
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPSMDBWithPXC),
		},
		{
			name: "PSMDB affinity config mismatch PosgreSQL",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "psmdb-mismatch-pg",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{
							Engine: &corev1.Affinity{},
						},
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPSMDBWithPostgresql),
		},
		{
			name: "PostgreSQL affinity config mismatch PXC",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pg-mismatch-pxc",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{},
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPostgresqlWithPXC),
		},
		{
			name: "PostgreSQL affinity config mismatch PSMDB",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pg-mismatch-psmdb",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{},
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPostgresqlWithPSMDB),
		},
		// valid simple cases
		{
			name: "PXC valid simple affinity config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pxc-valid-simple-cfg",
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
						},
					},
				},
			},
		},
		{
			name: "PSMDB valid simple affinity config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "psmdb-valid-simple-cfg",
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
						},
					},
				},
			},
		},
		{
			name: "PostgreSQL valid simple affinity config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pg-valid-simple-cfg",
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
						},
					},
				},
			},
		},
		// valid cases with full affinity config
		{
			name: "PXC valid full affinity config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pxc-valid-full-cfg",
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
			},
		},
		{
			name: "PSMDB valid full affinity config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "psmdb-valid-full-cfg",
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
			},
		},
		{
			name: "PostgreSQL valid full affinity config",
			policyToCreate: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pg-valid-full-cfg",
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
			k8sHandler := k8s.New(zap.NewNop().Sugar(), k, "")

			valHandler := New(zap.NewNop().Sugar(), k)
			valHandler.SetNext(k8sHandler)
			_, err := valHandler.CreatePodSchedulingPolicy(context.Background(), tc.policyToCreate)
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			assert.Equal(t, tc.wantErr.Error(), err.Error())
		})
	}
}

func TestValidate_ListPodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name    string
		objs    []ctrlclient.Object
		assert  func(*everestv1alpha1.PodSchedulingPolicyList) bool
		wantErr error
	}

	testCases := []testCase{
		// policies are absent
		{
			name: "absent policies",
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
		// default policies
		{
			name: "default policies",
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(tc.objs...).
				Build()
			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient)
			k8sHandler := k8s.New(zap.NewNop().Sugar(), k, "")

			valHandler := New(zap.NewNop().Sugar(), k)
			valHandler.SetNext(k8sHandler)
			pspList, err := valHandler.ListPodSchedulingPolicies(context.Background(), nil)
			require.NoError(t, err)
			assert.Condition(t, func() bool {
				return tc.assert(pspList)
			})
		})
	}
}

func TestValidate_GetPodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		objs       []ctrlclient.Object
		policyName string
		wantPolicy *everestv1alpha1.PodSchedulingPolicy
		wantErr    error
	}

	testCases := []testCase{
		// policies are absent
		{
			name:       "no policies",
			policyName: "everest-default-mysql",
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "podschedulingpolicies",
			},
				"everest-default-mysql",
			),
		},
		// get default policy
		{
			name: "get default policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			policyName: "everest-default-mysql",
			wantPolicy: getDefaultPXCPolicy(),
		},
		// get absent policy
		{
			name: "get absent policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			policyName: "non-existing-policy",
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "podschedulingpolicies",
			},
				"non-existing-policy",
			),
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
			k8sHandler := k8s.New(zap.NewNop().Sugar(), k, "")

			valHandler := New(zap.NewNop().Sugar(), k)
			valHandler.SetNext(k8sHandler)
			psp, err := valHandler.GetPodSchedulingPolicy(context.Background(), tc.policyName)
			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantPolicy.GetName(), psp.GetName())
		})
	}
}

func TestValidate_UpdatePodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name          string
		objs          []ctrlclient.Object
		updatedPolicy *everestv1alpha1.PodSchedulingPolicy
		wantPolicy    *everestv1alpha1.PodSchedulingPolicy
		wantErr       error
	}

	testCases := []testCase{
		// policies are absent
		{
			name: "no policies",
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
				},
			},
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "podschedulingpolicies",
			},
				"test-policy",
			),
		},
		// update non-existing policy
		{
			name: "update non-existing policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "non-existing-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "podschedulingpolicies",
			},
				"non-existing-policy",
			),
		},
		// update default policy PXC
		{
			name: "update default PXC policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "everest-default-mysql",
					Finalizers: []string{everestv1alpha1.ReadOnlyFinalizer},
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errUpdateDefaultPSP("everest-default-mysql")),
		},
		// update default policy PSMDB
		{
			name: "update default PSMDB policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "everest-default-mongodb",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errUpdateDefaultPSP("everest-default-mongodb")),
		},
		// update default policy PostgreSQL
		{
			name: "update default PosgreSQL policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "everest-default-postgresql",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errUpdateDefaultPSP("everest-default-postgresql")),
		},
		// affinity config mismatches with engineType
		{
			name: "PXC affinity config mismatch PSMDB",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPXCWithPSMDB),
		},
		{
			name: "PXC affinity config mismatch PostgreSQL",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPXCWithPostgresql),
		},
		{
			name: "PSMDB affinity config mismatch PXC",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPSMDBWithPXC),
		},
		{
			name: "PSMDB affinity config mismatch PosgreSQL",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPSMDBWithPostgresql),
		},
		{
			name: "PostgreSQL affinity config mismatch PXC",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPostgresqlWithPXC),
		},
		{
			name: "PostgreSQL affinity config mismatch PSMDB",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{
							Engine: &corev1.Affinity{},
						},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPostgresqlWithPSMDB),
		},
		// empty affinity configs
		{
			name: "empty PXC affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType:     everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPXCEmpty),
		},
		{
			name: "empty PSMDB affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType:     everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPSMDBEmpty),
		},
		{
			name: "empty Postgresql affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType:     everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPostgresqlEmpty),
		},
		// empty DB components affinity configs
		{
			name: "empty PXC components config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPXCComponentsEmpty),
		},
		{
			name: "empty PSMDB components config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPSMDBComponentsEmpty),
		},
		{
			name: "empty PostgreSQL components config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{},
					},
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errInvalidPSPAffinityPostgresqlComponentsEmpty),
		},
		// change an engine type
		{
			name: "update engine type",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePSMDB,
				},
			},
			wantErr: errors.Join(ErrInvalidRequest, errUpdatePSPEngineType),
		},
		// valid update - add full affinity config
		{
			name: "valid update PXC - add full affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "valid-update-pxc",
						ResourceVersion: "1",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pxc",
					ResourceVersion: "1",
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
			},
			wantPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pxc",
					ResourceVersion: "2",
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
			},
		},
		{
			name: "valid update PSMDB - add full affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "valid-update-psmdb",
						ResourceVersion: "1",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-psmdb",
					ResourceVersion: "1",
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
			},
			wantPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-psmdb",
					ResourceVersion: "2",
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
			},
		},
		{
			name: "valid update PosgreSQL - add full affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "valid-update-pg",
						ResourceVersion: "1",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pg",
					ResourceVersion: "1",
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
			},
			wantPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pg",
					ResourceVersion: "2",
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
			},
		},
		// valid update - add a DB component into affinity config
		{
			name: "valid update PXC - add db component into affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "valid-update-pxc",
						ResourceVersion: "1",
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
							},
						},
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pxc",
					ResourceVersion: "1",
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
			},
			wantPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pxc",
					ResourceVersion: "2",
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
			},
		},
		{
			name: "valid update PSMDB - add db component into affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "valid-update-psmdb",
						ResourceVersion: "1",
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
							},
						},
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-psmdb",
					ResourceVersion: "1",
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
			},
			wantPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-psmdb",
					ResourceVersion: "2",
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
			},
		},
		{
			name: "valid update PosgreSQL - add db component into affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "valid-update-pg",
						ResourceVersion: "1",
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
							},
						},
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pg",
					ResourceVersion: "1",
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
			},
			wantPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pg",
					ResourceVersion: "2",
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
			},
		},
		// valid update - replace a DB component in affinity config
		{
			name: "valid update PXC - replace db component in affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "valid-update-pxc",
						ResourceVersion: "1",
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
							},
						},
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pxc",
					ResourceVersion: "1",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{
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
			},
			wantPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pxc",
					ResourceVersion: "2",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePXC,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PXC: &everestv1alpha1.PXCAffinityConfig{
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
			},
		},
		{
			name: "valid update PSMDB - replace db component in affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "valid-update-psmdb",
						ResourceVersion: "1",
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
							},
						},
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-psmdb",
					ResourceVersion: "1",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{
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
			},
			wantPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-psmdb",
					ResourceVersion: "2",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePSMDB,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PSMDB: &everestv1alpha1.PSMDBAffinityConfig{
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
			},
		},
		{
			name: "valid update PosgreSQL - replace db component in affinity config",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "valid-update-pg",
						ResourceVersion: "1",
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
							},
						},
					},
				},
			},
			updatedPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pg",
					ResourceVersion: "1",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{
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
			},
			wantPolicy: &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "valid-update-pg",
					ResourceVersion: "2",
				},
				Spec: everestv1alpha1.PodSchedulingPolicySpec{
					EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					AffinityConfig: &everestv1alpha1.AffinityConfig{
						PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{
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
			k8sHandler := k8s.New(zap.NewNop().Sugar(), k, "")

			valHandler := New(zap.NewNop().Sugar(), k)
			valHandler.SetNext(k8sHandler)

			psp, err := valHandler.UpdatePodSchedulingPolicy(context.Background(), tc.updatedPolicy)
			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantPolicy, psp)
		})
	}
}

func TestValidate_DeletePodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name            string
		objs            []ctrlclient.Object
		pspNameToDelete string
		wantErr         error
	}
	testCases := []testCase{
		// no policies
		{
			name:            "no policies",
			pspNameToDelete: "test-policy",
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "podschedulingpolicies",
			},
				"test-policy",
			),
		},
		// delete non-existing policy
		{
			name: "delete non-existing policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			pspNameToDelete: "non-existing-policy",
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "podschedulingpolicies",
			},
				"non-existing-policy",
			),
		},
		// delete default PXC policy
		{
			name: "delete default PXC policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			pspNameToDelete: "everest-default-mysql",
			wantErr:         errors.Join(ErrInvalidRequest, errDeleteDefaultPSP("everest-default-mysql")),
		},
		// delete default PSMDB policy
		{
			name: "delete default PSMDB policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			pspNameToDelete: "everest-default-mongodb",
			wantErr:         errors.Join(ErrInvalidRequest, errDeleteDefaultPSP("everest-default-mongodb")),
		},
		// delete default PostgreSQL policy
		{
			name: "delete default PostgreSQL policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
			},
			pspNameToDelete: "everest-default-postgresql",
			wantErr:         errors.Join(ErrInvalidRequest, errDeleteDefaultPSP("everest-default-postgresql")),
		},
		// delete non-used policy
		{
			name: "delete non-used policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{},
				},
			},
			pspNameToDelete: "test-policy",
		},
		// delete used policy
		{
			name: "delete used policy",
			objs: []ctrlclient.Object{
				getDefaultPXCPolicy(),
				getDefaultPGPolicy(),
				getDefaultPSMDBPolicy(),
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-policy",
						Finalizers: []string{everestv1alpha1.InUseResourceFinalizer},
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{},
				},
			},
			pspNameToDelete: "test-policy",
			wantErr:         errors.Join(ErrInvalidRequest, errDeleteInUsePSP("test-policy")),
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
			k8sHandler := k8s.New(zap.NewNop().Sugar(), k, "")

			valHandler := New(zap.NewNop().Sugar(), k)
			valHandler.SetNext(k8sHandler)

			err := valHandler.DeletePodSchedulingPolicy(context.Background(), tc.pspNameToDelete)
			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
