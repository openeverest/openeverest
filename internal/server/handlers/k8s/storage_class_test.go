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
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

const (
	annotationStorageClassDefault      = "storageclass.kubernetes.io/is-default-class"
	annotationStorageClassDefaultValue = "true"
)

func TestK8s_StorageClasses(t *testing.T) {
	t.Parallel()

	reclaimPolicy := corev1.PersistentVolumeReclaimRetain
	bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
	standard := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "standard",
			Annotations: map[string]string{annotationStorageClassDefault: annotationStorageClassDefaultValue},
		},
		Provisioner:          "ebs.csi.aws.com",
		Parameters:           map[string]string{"type": "gp3"},
		ReclaimPolicy:        &reclaimPolicy,
		AllowVolumeExpansion: new(true),
		VolumeBindingMode:    &bindingMode,
	}
	local := &storagev1.StorageClass{
		ObjectMeta:  metav1.ObjectMeta{Name: "local-path"},
		Provisioner: "rancher.io/local-path",
	}
	h := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(
			fake.NewClientBuilder().WithScheme(kubernetes.CreateScheme()).WithObjects(standard, local).Build(),
		),
		log: zap.NewNop().Sugar(),
	}

	t.Run("list returns every storage class as stored", func(t *testing.T) {
		t.Parallel()

		list, err := h.ListStorageClasses(context.Background(), "prod")
		require.NoError(t, err)
		require.Len(t, list.Items, 2)
		byName := make(map[string]storagev1.StorageClass, len(list.Items))
		for _, sc := range list.Items {
			byName[sc.Name] = sc
		}
		require.Contains(t, byName, "standard")
		require.Contains(t, byName, "local-path")
		got := byName["standard"]
		assert.Equal(t, standard.Provisioner, got.Provisioner)
		assert.Equal(t, standard.Parameters, got.Parameters)
		assert.Equal(t, standard.ReclaimPolicy, got.ReclaimPolicy)
		assert.Equal(t, standard.AllowVolumeExpansion, got.AllowVolumeExpansion)
		assert.Equal(t, standard.VolumeBindingMode, got.VolumeBindingMode)
		assert.Equal(t, standard.Annotations, got.Annotations)
	})

	t.Run("get returns the named storage class", func(t *testing.T) {
		t.Parallel()

		got, err := h.GetStorageClass(context.Background(), "prod", "local-path")
		require.NoError(t, err)
		assert.Equal(t, "local-path", got.Name)
		assert.Equal(t, "rancher.io/local-path", got.Provisioner)
	})

	t.Run("get of a missing storage class is not found", func(t *testing.T) {
		t.Parallel()

		_, err := h.GetStorageClass(context.Background(), "prod", "does-not-exist")
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}
