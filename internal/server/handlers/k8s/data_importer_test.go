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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/kubernetes"
)

func TestListDataImporters(t *testing.T) {
	t.Parallel()

	// Helper function to create a DataImporter
	createDataImporter := func(name string, supportedEngines ...everestv1alpha1.EngineType) *everestv1alpha1.DataImporter {
		return &everestv1alpha1.DataImporter{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: everestv1alpha1.DataImporterSpec{
				SupportedEngines: supportedEngines,
				Description:      "Test importer " + name,
			},
		}
	}

	testCases := []struct {
		name             string
		objs             []ctrlclient.Object
		supportedEngines []string
		assert           func(*everestv1alpha1.DataImporterList) bool
	}{
		{
			name: "no importers exist",
			assert: func(list *everestv1alpha1.DataImporterList) bool {
				return len(list.Items) == 0
			},
		},
		{
			name: "all importers without filter",
			objs: []ctrlclient.Object{
				createDataImporter("importer-1", everestv1alpha1.DatabaseEnginePXC),
				createDataImporter("importer-2", everestv1alpha1.DatabaseEnginePSMDB),
				createDataImporter("importer-3", everestv1alpha1.DatabaseEnginePostgresql),
			},
			assert: func(list *everestv1alpha1.DataImporterList) bool {
				return len(list.Items) == 3
			},
		},
		{
			name: "filter by PXC engine",
			objs: []ctrlclient.Object{
				createDataImporter("importer-1", everestv1alpha1.DatabaseEnginePXC),
				createDataImporter("importer-2", everestv1alpha1.DatabaseEnginePSMDB),
				createDataImporter("importer-3", everestv1alpha1.DatabaseEnginePostgresql),
			},
			supportedEngines: []string{"pxc"},
			assert: func(list *everestv1alpha1.DataImporterList) bool {
				return len(list.Items) == 1 &&
					list.Items[0].GetName() == "importer-1" &&
					slices.Contains(list.Items[0].Spec.SupportedEngines, everestv1alpha1.DatabaseEnginePXC)
			},
		},
		{
			name: "filter by PSMDB engine",
			objs: []ctrlclient.Object{
				createDataImporter("importer-1", everestv1alpha1.DatabaseEnginePXC),
				createDataImporter("importer-2", everestv1alpha1.DatabaseEnginePSMDB),
				createDataImporter("importer-3", everestv1alpha1.DatabaseEnginePostgresql),
			},
			supportedEngines: []string{"psmdb"},
			assert: func(list *everestv1alpha1.DataImporterList) bool {
				return len(list.Items) == 1 &&
					list.Items[0].GetName() == "importer-2" &&
					slices.Contains(list.Items[0].Spec.SupportedEngines, everestv1alpha1.DatabaseEnginePSMDB)
			},
		},
		{
			name: "filter by PostgreSQL engine",
			objs: []ctrlclient.Object{
				createDataImporter("importer-1", everestv1alpha1.DatabaseEnginePXC),
				createDataImporter("importer-2", everestv1alpha1.DatabaseEnginePSMDB),
				createDataImporter("importer-3", everestv1alpha1.DatabaseEnginePostgresql),
			},
			supportedEngines: []string{"postgresql"},
			assert: func(list *everestv1alpha1.DataImporterList) bool {
				return len(list.Items) == 1 &&
					list.Items[0].GetName() == "importer-3" &&
					slices.Contains(list.Items[0].Spec.SupportedEngines, everestv1alpha1.DatabaseEnginePostgresql)
			},
		},
		{
			name: "filter by multiple engines",
			objs: []ctrlclient.Object{
				createDataImporter("importer-1", everestv1alpha1.DatabaseEnginePXC),
				createDataImporter("importer-2", everestv1alpha1.DatabaseEnginePSMDB),
				createDataImporter("importer-3", everestv1alpha1.DatabaseEnginePostgresql),
			},
			supportedEngines: []string{"pxc", "postgresql"},
			assert: func(list *everestv1alpha1.DataImporterList) bool {
				slices.SortFunc(list.Items, func(a, b everestv1alpha1.DataImporter) int {
					return slices.Compare([]string{a.GetName()}, []string{b.GetName()})
				})
				return len(list.Items) == 2 &&
					list.Items[0].GetName() == "importer-1" &&
					list.Items[1].GetName() == "importer-3"
			},
		},
		{
			name: "importer supporting multiple engines",
			objs: []ctrlclient.Object{
				createDataImporter("multi-engine", everestv1alpha1.DatabaseEnginePXC, everestv1alpha1.DatabaseEnginePSMDB),
				createDataImporter("single-engine", everestv1alpha1.DatabaseEnginePostgresql),
			},
			supportedEngines: []string{"pxc"},
			assert: func(list *everestv1alpha1.DataImporterList) bool {
				return len(list.Items) == 1 &&
					list.Items[0].GetName() == "multi-engine" &&
					list.Items[0].Spec.SupportedEngines[0] == everestv1alpha1.DatabaseEnginePXC
			},
		},
		{
			name: "filter with no matching engines",
			objs: []ctrlclient.Object{
				createDataImporter("importer-1", everestv1alpha1.DatabaseEnginePXC),
				createDataImporter("importer-2", everestv1alpha1.DatabaseEnginePSMDB),
			},
			supportedEngines: []string{"postgresql"},
			assert: func(list *everestv1alpha1.DataImporterList) bool {
				return len(list.Items) == 0
			},
		},
		{
			name: "no duplicate importers due to multiple engine filters",
			objs: []ctrlclient.Object{
				createDataImporter("multi-engine", everestv1alpha1.DatabaseEnginePXC, everestv1alpha1.DatabaseEnginePSMDB),
			},
			supportedEngines: []string{"pxc", "psmdb"},
			assert: func(list *everestv1alpha1.DataImporterList) bool {
				return len(list.Items) == 1 &&
					list.Items[0].GetName() == "multi-engine"
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create mock client with test objects
			mockClient := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(tc.objs...).
				Build()

			// Create k8s handler with mock client
			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient)
			k8sH := New(zap.NewNop().Sugar(), k, "")

			// Call the function under test
			importerList, err := k8sH.ListDataImporters(context.Background(), tc.supportedEngines...)

			// Assert results
			require.NoError(t, err)
			assert.Condition(t, func() bool {
				return tc.assert(importerList)
			})
		})
	}
}
