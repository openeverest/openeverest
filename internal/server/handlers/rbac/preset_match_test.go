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

package rbac

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	apicommon "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

func TestSpecDifferences(t *testing.T) {
	t.Parallel()

	preset := func() corev1alpha1.InstanceSpec {
		return corev1alpha1.InstanceSpec{
			ProviderRef: apicommon.ObjectRef{Name: "mysql"},
			Version:     "8.0",
			Components: map[string]corev1alpha1.ComponentSpec{
				"engine": {
					Replicas:   new(int32(3)),
					Parameters: &runtime.RawExtension{Raw: []byte(`{"maxConnections":100}`)},
					Resources: &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
					},
				},
			},
			Backup: &corev1alpha1.InstanceBackupSpec{
				Enabled:  true,
				ClassRef: apicommon.ObjectRef{Name: "native"},
				Storages: []corev1alpha1.InstanceBackupStorage{{
					StorageRef: apicommon.ObjectRef{Name: "s3"},
					Schedules:  []corev1alpha1.InstanceBackupSchedule{{Name: "daily", Enabled: true, Cron: "0 0 * * *"}},
				}},
			},
		}
	}

	testCases := []struct {
		desc   string
		mutate func(*corev1alpha1.InstanceSpec)
		want   []string
	}{
		{
			desc:   "identical",
			mutate: func(*corev1alpha1.InstanceSpec) {},
		},
		{
			desc:   "top-level field",
			mutate: func(s *corev1alpha1.InstanceSpec) { s.Version = "8.4" },
			want:   []string{"spec.version"},
		},
		{
			desc: "field of a component",
			mutate: func(s *corev1alpha1.InstanceSpec) {
				engine := s.Components["engine"]
				engine.Replicas = new(int32(5))
				s.Components["engine"] = engine
			},
			want: []string{"spec.components[engine].replicas"},
		},
		{
			desc: "quantity in a resource list",
			mutate: func(s *corev1alpha1.InstanceSpec) {
				s.Components["engine"].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("2")
			},
			want: []string{"spec.components[engine].resources.limits[cpu]"},
		},
		{
			desc: "parameters are named as a whole",
			mutate: func(s *corev1alpha1.InstanceSpec) {
				s.Components["engine"].Parameters.Raw = []byte(`{"maxConnections":900}`)
			},
			want: []string{"spec.components[engine].parameters"},
		},
		{
			desc: "added component",
			mutate: func(s *corev1alpha1.InstanceSpec) {
				s.Components["proxy"] = corev1alpha1.ComponentSpec{Type: "haproxy"}
			},
			want: []string{"spec.components[proxy]"},
		},
		{
			desc:   "field the preset leaves unset",
			mutate: func(s *corev1alpha1.InstanceSpec) { s.UserSecretRef = &apicommon.SecretRef{Name: "creds"} },
			want:   []string{"spec.userSecretRef"},
		},
		{
			desc:   "field the preset sets, dropped",
			mutate: func(s *corev1alpha1.InstanceSpec) { s.Backup = nil },
			want:   []string{"spec.backup"},
		},
		{
			desc:   "field of a list element",
			mutate: func(s *corev1alpha1.InstanceSpec) { s.Backup.Storages[0].Schedules[0].Cron = "* * * * *" },
			want:   []string{"spec.backup.storages[0].schedules[0].cron"},
		},
		{
			desc: "added list element names the list",
			mutate: func(s *corev1alpha1.InstanceSpec) {
				s.Backup.Storages = append(s.Backup.Storages, corev1alpha1.InstanceBackupStorage{StorageRef: apicommon.ObjectRef{Name: "gcs"}})
			},
			want: []string{"spec.backup.storages"},
		},
		{
			desc: "every difference, in field order",
			mutate: func(s *corev1alpha1.InstanceSpec) {
				s.Version = "8.4"
				s.Backup.Enabled = false
				s.Components["engine"].Parameters.Raw = []byte(`{}`)
			},
			want: []string{"spec.version", "spec.components[engine].parameters", "spec.backup.enabled"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			spec := preset()
			tc.mutate(&spec)
			assert.Equal(t, tc.want, specDifferences(preset(), spec))
		})
	}
}

func TestMetadataOutsideAllowlist(t *testing.T) {
	t.Parallel()

	const annotation = common.ClusterInstancePresetAnnotation

	testCases := []struct {
		desc string
		meta metav1.ObjectMeta
		want []string
	}{
		{
			desc: "address and preset annotation",
			meta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1", Annotations: map[string]string{annotation: "small"}},
		},
		{
			desc: "generated name",
			meta: metav1.ObjectMeta{GenerateName: "db-", Namespace: "ns1", Annotations: map[string]string{annotation: "small"}},
		},
		{
			desc: "empty lists and maps",
			meta: metav1.ObjectMeta{Name: "db1", Labels: make(map[string]string), Finalizers: []string{}, OwnerReferences: []metav1.OwnerReference{}},
		},
		{
			desc: "any label",
			meta: metav1.ObjectMeta{Name: "db1", Labels: map[string]string{"team": "a", "app.kubernetes.io/name": "db"}},
			want: []string{"metadata.labels[app.kubernetes.io/name]", "metadata.labels[team]"},
		},
		{
			desc: "another annotation",
			meta: metav1.ObjectMeta{Name: "db1", Annotations: map[string]string{annotation: "small", "example.com/backup-hook": "x"}},
			want: []string{"metadata.annotations[example.com/backup-hook]"},
		},
		{
			desc: "the annotation naming the other kind of preset",
			meta: metav1.ObjectMeta{Name: "db1", Annotations: map[string]string{annotation: "small", common.InstancePresetAnnotation: "large"}},
			want: []string{"metadata.annotations[openeverest.io/instance-preset]"},
		},
		{
			desc: "finalizers and owner references",
			meta: metav1.ObjectMeta{
				Name:            "db1",
				Finalizers:      []string{"example.com/hold"},
				OwnerReferences: []metav1.OwnerReference{{Kind: "ConfigMap", Name: "owner"}},
			},
			want: []string{"metadata.ownerReferences", "metadata.finalizers"},
		},
		{
			desc: "any other field",
			meta: metav1.ObjectMeta{Name: "db1", ResourceVersion: "42"},
			want: []string{"metadata.resourceVersion"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, metadataOutsideAllowlist(tc.meta, annotation))
		})
	}
}

func TestPresetMismatch(t *testing.T) {
	t.Parallel()

	preset := corev1alpha1.InstanceSpec{ProviderRef: apicommon.ObjectRef{Name: "mysql"}, Version: "8.0"}
	instance := func(version string, annotations map[string]string) *corev1alpha1.Instance {
		return &corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1", Annotations: annotations},
			Spec:       corev1alpha1.InstanceSpec{ProviderRef: apicommon.ObjectRef{Name: "mysql"}, Version: version},
		}
	}
	named := map[string]string{common.InstancePresetAnnotation: "small"}

	t.Run("exact copy", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, presetMismatch(instance("8.0", named), preset, "InstancePreset", "small", common.InstancePresetAnnotation))
	})

	t.Run("spec and metadata differ", func(t *testing.T) {
		t.Parallel()
		err := presetMismatch(
			instance("9.0", map[string]string{common.InstancePresetAnnotation: "small", "example.com/hook": "x"}),
			preset, "InstancePreset", "small", common.InstancePresetAnnotation,
		)

		require.ErrorIs(t, err, ErrInsufficientPermissions)
		var mismatch *PresetMismatchError
		require.ErrorAs(t, err, &mismatch)
		assert.Equal(t, []string{"spec.version"}, mismatch.Fields)
		assert.Equal(t, []string{"metadata.annotations[example.com/hook]"}, mismatch.Metadata)
		assert.Equal(t,
			`instance does not match InstancePreset "small": it changes spec.version and sets metadata.annotations[example.com/hook]. `+
				`instances:create-from-preset allows only an exact copy of the preset; changes need instances:create`,
			err.Error(),
		)
		wantCauses := []metav1.StatusCause{
			{Type: metav1.CauseTypeForbidden, Message: `differs from InstancePreset "small"`, Field: "spec.version"},
			{Type: metav1.CauseTypeForbidden, Message: "not allowed on an instance created from a preset", Field: "metadata.annotations[example.com/hook]"},
		}
		assert.Equal(t, wantCauses, mismatch.Causes())
	})

	t.Run("metadata only", func(t *testing.T) {
		t.Parallel()
		err := presetMismatch(
			instance("8.0", map[string]string{common.InstancePresetAnnotation: "small", "example.com/hook": "x"}),
			preset, "InstancePreset", "small", common.InstancePresetAnnotation,
		)
		assert.EqualError(t, err,
			`instance does not match InstancePreset "small": it sets metadata.annotations[example.com/hook]. `+
				`instances:create-from-preset allows only an exact copy of the preset; changes need instances:create`,
		)
	})
}

// mismatchPaths returns the paths err names if it is a PresetMismatchError,
// and nil otherwise.
func mismatchPaths(err error) []string {
	var mismatch *PresetMismatchError
	if !errors.As(err, &mismatch) {
		return nil
	}
	return slices.Concat(mismatch.Fields, mismatch.Metadata)
}
