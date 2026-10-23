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

package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestK8s_ListInstanceBackups(t *testing.T) {
	t.Parallel()

	newInstanceBackup := func(name, namespace, instance string, labels map[string]string) *backupv1alpha1.Backup {
		return &backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
			Spec: backupv1alpha1.BackupSpec{
				Origin: backupv1alpha1.BackupOrigin{
					Type:        backupv1alpha1.BackupOriginTypeInstance,
					InstanceRef: &common.ObjectRef{Name: instance},
				},
			},
		}
	}

	newExternalBackup := func(name, namespace string) *backupv1alpha1.Backup {
		return &backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: backupv1alpha1.BackupSpec{
				Origin: backupv1alpha1.BackupOrigin{
					Type:     backupv1alpha1.BackupOriginTypeExternal,
					External: &backupv1alpha1.BackupOriginExternal{Path: name},
				},
			},
		}
	}

	newHandler := func() *k8sHandler {
		fakeClient := fake.NewClientBuilder().
			WithScheme(kubernetes.CreateScheme()).
			WithIndex(&backupv1alpha1.Backup{}, "spec.origin.instanceRef.name", func(o ctrlclient.Object) []string {
				b, _ := o.(*backupv1alpha1.Backup)
				if b.Spec.Origin.InstanceRef == nil {
					return nil
				}
				return []string{b.Spec.Origin.InstanceRef.Name}
			}).
			WithObjects(
				// Freshly created via the API: the controller has not stamped the
				// instanceName label yet, but the spec already names the instance.
				newInstanceBackup("instance-1-unlabeled", "ns1", "instance-1", nil),
				newInstanceBackup("instance-1-labeled", "ns1", "instance-1", map[string]string{"instanceName": "instance-1"}),
				newInstanceBackup("instance-2", "ns1", "instance-2", map[string]string{"instanceName": "instance-2"}),
				newInstanceBackup("other-namespace", "ns2", "instance-1", map[string]string{"instanceName": "instance-1"}),
				newExternalBackup("external", "ns1"),
			).
			Build()
		return &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}
	}

	testCases := []struct {
		desc        string
		instance    string
		wantBackups []string
	}{
		{
			desc:        "lists by spec regardless of label",
			instance:    "instance-1",
			wantBackups: []string{"instance-1-unlabeled", "instance-1-labeled"},
		},
		{
			desc:        "other instance",
			instance:    "instance-2",
			wantBackups: []string{"instance-2"},
		},
		{
			desc:        "no match",
			instance:    "missing",
			wantBackups: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			list, err := newHandler().ListInstanceBackups(context.Background(), "prod", "ns1", tc.instance)
			require.NoError(t, err)

			names := make([]string, 0, len(list.Items))
			for _, b := range list.Items {
				names = append(names, b.Name)
			}
			assert.ElementsMatch(t, tc.wantBackups, names)
		})
	}
}
