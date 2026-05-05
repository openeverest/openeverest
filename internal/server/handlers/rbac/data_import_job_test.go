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

package rbac

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/rbac"
)

func TestRBAC_DataImportJob(t *testing.T) {
	t.Parallel()

	t.Run("ListDataImportJobs", func(t *testing.T) {
		t.Parallel()

		// Setup test data
		const testNamespace = "default"
		const testDBName = "test-cluster"

		// helper to genereate mock data
		createJob := func(name string) everestv1alpha1.DataImportJob {
			return everestv1alpha1.DataImportJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: testNamespace,
				},
				Spec: everestv1alpha1.DataImportJobSpec{
					TargetClusterName: testDBName,
				},
			}
		}

		data := func() *handlers.MockHandler {
			allJobs := &everestv1alpha1.DataImportJobList{
				Items: []everestv1alpha1.DataImportJob{
					createJob("job1"),
					createJob("job2"),
					createJob("job3"),
				},
			}
			next := &handlers.MockHandler{}
			next.On("ListDataImportJobs", mock.Anything, testNamespace, testDBName).
				Return(allJobs, nil)
			return next
		}

		testCases := []struct {
			desc   string
			policy string
			assert func(*everestv1alpha1.DataImportJobList) bool
		}{
			{
				desc: "access to default/test-cluster",
				policy: newPolicy(
					"p, role:test, data-import-jobs, read, default/test-cluster",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DataImportJobList) bool {
					return len(list.Items) == 3 &&
						slices.ContainsFunc(list.Items, func(job everestv1alpha1.DataImportJob) bool {
							return job.Name == "job1"
						}) &&
						slices.ContainsFunc(list.Items, func(job everestv1alpha1.DataImportJob) bool {
							return job.Name == "job2"
						}) &&
						slices.ContainsFunc(list.Items, func(job everestv1alpha1.DataImportJob) bool {
							return job.Name == "job3"
						})
				},
			},
			{
				desc: "access with wildcard",
				policy: newPolicy(
					"p, role:test, data-import-jobs, read, default/*",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DataImportJobList) bool {
					return len(list.Items) == 3 &&
						slices.ContainsFunc(list.Items, func(job everestv1alpha1.DataImportJob) bool {
							return job.Name == "job1"
						}) &&
						slices.ContainsFunc(list.Items, func(job everestv1alpha1.DataImportJob) bool {
							return job.Name == "job2"
						}) &&
						slices.ContainsFunc(list.Items, func(job everestv1alpha1.DataImportJob) bool {
							return job.Name == "job3"
						})
				},
			},
			{
				desc: "admin access",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *everestv1alpha1.DataImportJobList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:   "no access",
				policy: newPolicy("g, bob, role:view"),
				assert: func(list *everestv1alpha1.DataImportJobList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc: "wrong namespace",
				policy: newPolicy(
					"p, role:test, data-import-jobs, read, other/test-cluster",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DataImportJobList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc: "wrong cluster",
				policy: newPolicy(
					"p, role:test, data-import-jobs, read, default/other-cluster",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DataImportJobList) bool {
					return len(list.Items) == 0
				},
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			tc := tc
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()

				// setup mocks
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := data()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.ListDataImportJobs(ctx, testNamespace, testDBName)
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(result)
				})
			})
		}
	})
}
