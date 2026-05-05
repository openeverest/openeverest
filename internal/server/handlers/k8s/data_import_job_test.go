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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
)

func TestListDataImportJobs(t *testing.T) {
	t.Parallel()

	// Define sample namespace and database name
	const (
		testNamespace = "test-namespace"
		testDBName    = "test-db"
		otherDBName   = "other-db"
	)

	// Create a simple data import job for testing
	createDataImportJob := func(name, namespace, dbName string) *everestv1alpha1.DataImportJob {
		return &everestv1alpha1.DataImportJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					common.DatabaseClusterNameLabel: dbName,
				},
			},
			Spec: everestv1alpha1.DataImportJobSpec{
				TargetClusterName:     dbName,
				DataImportJobTemplate: &everestv1alpha1.DataImportJobTemplate{},
			},
			Status: everestv1alpha1.DataImportJobStatus{
				State: everestv1alpha1.DataImportJobStateRunning,
			},
		}
	}

	testCases := []struct {
		name   string
		objs   []ctrlclient.Object
		dbName string
		assert func(*everestv1alpha1.DataImportJobList) bool
	}{
		{
			name:   "no import jobs",
			dbName: testDBName,
			assert: func(list *everestv1alpha1.DataImportJobList) bool {
				return len(list.Items) == 0
			},
		},
		{
			name: "one import job for given DB",
			objs: []ctrlclient.Object{
				createDataImportJob("job-1", testNamespace, testDBName),
			},
			dbName: testDBName,
			assert: func(list *everestv1alpha1.DataImportJobList) bool {
				return len(list.Items) == 1 &&
					list.Items[0].GetName() == "job-1" &&
					list.Items[0].Spec.TargetClusterName == testDBName
			},
		},
		{
			name: "multiple import jobs with only one for given DB",
			objs: []ctrlclient.Object{
				createDataImportJob("job-1", testNamespace, testDBName),
				createDataImportJob("job-2", testNamespace, otherDBName),
				createDataImportJob("job-3", "other-namespace", testDBName),
			},
			dbName: testDBName,
			assert: func(list *everestv1alpha1.DataImportJobList) bool {
				return len(list.Items) == 1 &&
					list.Items[0].GetName() == "job-1" &&
					list.Items[0].Spec.TargetClusterName == testDBName
			},
		},
		{
			name: "job exists but not for given DB",
			objs: []ctrlclient.Object{
				createDataImportJob("job-1", testNamespace, otherDBName),
			},
			dbName: testDBName,
			assert: func(list *everestv1alpha1.DataImportJobList) bool {
				return len(list.Items) == 0
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
			jobList, err := k8sH.ListDataImportJobs(context.Background(), testNamespace, tc.dbName)

			require.NoError(t, err)
			assert.Condition(t, func() bool {
				return tc.assert(jobList)
			})
		})
	}
}
