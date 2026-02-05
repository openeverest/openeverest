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

func TestRBAC_DataImporter(t *testing.T) {
	t.Parallel()

	t.Run("ListDataImporters", func(t *testing.T) {
		t.Parallel()

		// Helper to generate mock data
		createImporter := func(name string, engines ...everestv1alpha1.EngineType) everestv1alpha1.DataImporter {
			return everestv1alpha1.DataImporter{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: everestv1alpha1.DataImporterSpec{
					SupportedEngines: engines,
				},
			}
		}

		// Create test data
		data := func() *handlers.MockHandler {
			allImporters := &everestv1alpha1.DataImporterList{
				Items: []everestv1alpha1.DataImporter{
					createImporter("importer1", everestv1alpha1.DatabaseEnginePXC),
					createImporter("importer2", everestv1alpha1.DatabaseEnginePSMDB),
					createImporter("importer3", everestv1alpha1.DatabaseEnginePostgresql),
				},
			}
			next := &handlers.MockHandler{}
			next.On("ListDataImporters", mock.Anything, mock.Anything).
				Return(allImporters, nil)
			return next
		}

		testCases := []struct {
			desc   string
			policy string
			assert func(*everestv1alpha1.DataImporterList) bool
		}{
			{
				desc: "access to all importers",
				policy: newPolicy(
					"p, role:test, data-importers, read, importer1",
					"p, role:test, data-importers, read, importer2",
					"p, role:test, data-importers, read, importer3",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DataImporterList) bool {
					return len(list.Items) == 3 &&
						slices.ContainsFunc(list.Items, func(di everestv1alpha1.DataImporter) bool {
							return di.Name == "importer1"
						}) &&
						slices.ContainsFunc(list.Items, func(di everestv1alpha1.DataImporter) bool {
							return di.Name == "importer2"
						}) &&
						slices.ContainsFunc(list.Items, func(di everestv1alpha1.DataImporter) bool {
							return di.Name == "importer3"
						})
				},
			},
			{
				desc: "access to some importers",
				policy: newPolicy(
					"p, role:test, data-importers, read, importer1",
					"p, role:test, data-importers, read, importer3",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DataImporterList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(di everestv1alpha1.DataImporter) bool {
							return di.Name == "importer1"
						}) &&
						slices.ContainsFunc(list.Items, func(di everestv1alpha1.DataImporter) bool {
							return di.Name == "importer3"
						})
				},
			},
			{
				desc: "access with wildcard",
				policy: newPolicy(
					"p, role:test, data-importers, read, *",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DataImporterList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc: "admin access",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *everestv1alpha1.DataImporterList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:   "no access",
				policy: newPolicy("g, bob, role:view"),
				assert: func(list *everestv1alpha1.DataImporterList) bool {
					return len(list.Items) == 0
				},
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			tc := tc
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()

				// Setup mocks
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

				// Call the function under test with no engine filter
				result, err := h.ListDataImporters(ctx)
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(result)
				})
			})
		}
	})
}
