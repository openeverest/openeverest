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
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestK8s_ListBackups(t *testing.T) {
	t.Parallel()

	newBackup := func(name, namespace, class, storage string, origin backupv1alpha1.BackupOriginType) *backupv1alpha1.Backup {
		b := &backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: backupv1alpha1.BackupSpec{
				ClassRef:   common.ObjectRef{Name: class},
				StorageRef: common.ObjectRef{Name: storage},
				Origin:     backupv1alpha1.BackupOrigin{Type: origin},
			},
		}
		if origin == backupv1alpha1.BackupOriginTypeInstance {
			b.Spec.Origin.InstanceRef = &common.ObjectRef{Name: "instance-1"}
		} else {
			b.Spec.Origin.External = &backupv1alpha1.BackupOriginExternal{Path: name}
		}
		return b
	}

	newHandler := func() *k8sHandler {
		backup := &backupv1alpha1.Backup{}
		fakeClient := fake.NewClientBuilder().
			WithScheme(kubernetes.CreateScheme()).
			WithIndex(backup, "spec.classRef.name", func(o ctrlclient.Object) []string {
				b, _ := o.(*backupv1alpha1.Backup)
				return []string{b.Spec.ClassRef.Name}
			}).
			WithIndex(backup, "spec.storageRef.name", func(o ctrlclient.Object) []string {
				b, _ := o.(*backupv1alpha1.Backup)
				return []string{b.Spec.StorageRef.Name}
			}).
			WithIndex(backup, "spec.origin.type", func(o ctrlclient.Object) []string {
				b, _ := o.(*backupv1alpha1.Backup)
				return []string{string(b.Spec.Origin.Type)}
			}).
			WithObjects(
				newBackup("storage1-s3-instance", "ns1", "storage1", "s3", backupv1alpha1.BackupOriginTypeInstance),
				newBackup("storage2-s3-instance", "ns1", "storage2", "s3", backupv1alpha1.BackupOriginTypeInstance),
				newBackup("storage1-s3-external", "ns1", "storage1", "s3", backupv1alpha1.BackupOriginTypeExternal),
				newBackup("storage1-gcs-external", "ns1", "storage1", "gcs", backupv1alpha1.BackupOriginTypeExternal),
				newBackup("other-namespace", "ns2", "storage1", "s3", backupv1alpha1.BackupOriginTypeExternal),
			).
			Build()
		return &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}
	}

	origin := func(o api.ListBackupsParamsBackupOrigin) *api.ListBackupsParamsBackupOrigin { return &o }

	testCases := []struct {
		desc        string
		params      *api.ListBackupsParams
		wantBackups []string
	}{
		{
			desc:   "nil params lists namespace",
			params: nil,
			wantBackups: []string{
				"storage2-s3-instance",
				"storage1-gcs-external",
				"storage1-s3-external",
				"storage1-s3-instance",
			},
		},
		{
			desc:   "empty filters are ignored",
			params: &api.ListBackupsParams{BackupClass: new(""), BackupStorage: new("")},
			wantBackups: []string{
				"storage2-s3-instance",
				"storage1-gcs-external",
				"storage1-s3-external",
				"storage1-s3-instance",
			},
		},
		{
			desc:   "by class",
			params: &api.ListBackupsParams{BackupClass: new("storage2")},
			wantBackups: []string{
				"storage2-s3-instance",
			},
		},
		{
			desc:   "by storage",
			params: &api.ListBackupsParams{BackupStorage: new("gcs")},
			wantBackups: []string{
				"storage1-gcs-external",
			},
		},
		{
			desc: "by origin",
			params: &api.ListBackupsParams{
				BackupOrigin: origin(api.ListBackupsParamsBackupOriginExternal),
			},
			wantBackups: []string{
				"storage1-gcs-external",
				"storage1-s3-external",
			},
		},
		{
			desc: "all filters combined",
			params: &api.ListBackupsParams{
				BackupClass:   new("storage1"),
				BackupStorage: new("s3"),
				BackupOrigin:  origin(api.ListBackupsParamsBackupOriginExternal),
			},
			wantBackups: []string{
				"storage1-s3-external",
			},
		},
		{
			desc:        "no match",
			params:      &api.ListBackupsParams{BackupClass: new("missing")},
			wantBackups: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			list, err := newHandler().ListBackups(context.Background(), "prod", "ns1", tc.params)
			require.NoError(t, err)

			names := make([]string, 0, len(list.Items))
			for _, b := range list.Items {
				names = append(names, b.Name)
			}
			assert.ElementsMatch(t, tc.wantBackups, names)
		})
	}
}
