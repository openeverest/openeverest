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
	"github.com/percona/everest/internal/server/handlers/k8s"
	"github.com/percona/everest/pkg/kubernetes"
)

const (
	mcNamespace = "test-ns"
)

func TestValidate_DeleteMonitoringInstance(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name            string
		objs            []ctrlclient.Object
		objNameToDelete string
		wantErr         error
	}
	testCases := []testCase{
		// no monitoring instances
		{
			name:            "no monitoring instances",
			objNameToDelete: "test-monitoring-instance",
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "monitoringconfigs",
			},
				"test-monitoring-instance",
			),
		},
		// delete non-existing monitoring instance
		{
			name: "delete non-existing monitoring instance",
			objs: []ctrlclient.Object{
				&everestv1alpha1.MonitoringConfig{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: mcNamespace,
						Name:      "test-monitoring-instance",
					},
					Spec: everestv1alpha1.MonitoringConfigSpec{},
				},
			},
			objNameToDelete: "non-existing-monitoring-instance",
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "monitoringconfigs",
			},
				"non-existing-monitoring-instance",
			),
		},
		// delete non-used monitoring instance
		{
			name: "delete non-used monitoring instance",
			objs: []ctrlclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: mcNamespace,
						Name:      "test-monitoring-instance",
					},
				},
				&everestv1alpha1.MonitoringConfig{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: mcNamespace,
						Name:      "test-monitoring-instance",
					},
					Spec: everestv1alpha1.MonitoringConfigSpec{},
				},
			},
			objNameToDelete: "test-monitoring-instance",
		},
		// delete used monitoring instance
		{
			name: "delete used monitoring instance",
			objs: []ctrlclient.Object{
				&everestv1alpha1.MonitoringConfig{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  mcNamespace,
						Name:       "test-monitoring-instance",
						Finalizers: []string{everestv1alpha1.InUseResourceFinalizer},
					},
					Spec: everestv1alpha1.MonitoringConfigSpec{},
				},
			},
			objNameToDelete: "test-monitoring-instance",
			wantErr:         errors.Join(ErrInvalidRequest, errDeleteInUseMonitoringConfig(mcNamespace, "test-monitoring-instance")),
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

			err := valHandler.DeleteMonitoringInstance(context.Background(), mcNamespace, tc.objNameToDelete)
			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
