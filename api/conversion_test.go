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

package api

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/percona/everest-operator/api/everest/v1alpha1"
)

func TestBackupStorage_FromCR(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name     string
		input    *v1alpha1.BackupStorage
		expected BackupStorage
	}

	tcases := []tcase{
		{
			name: "all fields populated",
			input: &v1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-storage",
					Namespace: "my-ns",
				},
				Spec: v1alpha1.BackupStorageSpec{
					Type:           v1alpha1.BackupStorageTypeS3,
					Bucket:         "my-bucket",
					Region:         "us-east-1",
					EndpointURL:    "https://s3.example.com",
					Description:    "Test storage",
					VerifyTLS:      pointer.To(true),
					ForcePathStyle: pointer.To(false),
				},
			},
			expected: BackupStorage{
				Type:           BackupStorageType(v1alpha1.BackupStorageTypeS3),
				Name:           "my-storage",
				Namespace:      "my-ns",
				BucketName:     "my-bucket",
				Region:         "us-east-1",
				Url:            pointer.To("https://s3.example.com"),
				Description:    pointer.To("Test storage"),
				VerifyTLS:      pointer.To(true),
				ForcePathStyle: pointer.To(false),
			},
		},
		{
			name: "minimal fields - empty optionals",
			input: &v1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "min-storage",
					Namespace: "default",
				},
				Spec: v1alpha1.BackupStorageSpec{
					Type:   v1alpha1.BackupStorageTypeS3,
					Bucket: "bucket",
				},
			},
			expected: BackupStorage{
				Type:        BackupStorageType(v1alpha1.BackupStorageTypeS3),
				Name:        "min-storage",
				Namespace:   "default",
				BucketName:  "bucket",
				Url:         pointer.To(""),
				Description: pointer.To(""),
			},
		},
		{
			name: "azure storage type",
			input: &v1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "azure-storage",
					Namespace: "prod",
				},
				Spec: v1alpha1.BackupStorageSpec{
					Type:        v1alpha1.BackupStorageTypeAzure,
					Bucket:      "azure-container",
					Region:      "eastus",
					EndpointURL: "https://blob.core.windows.net",
					Description: "Azure backup",
					VerifyTLS:   pointer.To(false),
				},
			},
			expected: BackupStorage{
				Type:        BackupStorageType(v1alpha1.BackupStorageTypeAzure),
				Name:        "azure-storage",
				Namespace:   "prod",
				BucketName:  "azure-container",
				Region:      "eastus",
				Url:         pointer.To("https://blob.core.windows.net"),
				Description: pointer.To("Azure backup"),
				VerifyTLS:   pointer.To(false),
			},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out BackupStorage
			out.FromCR(tc.input)
			assert.Equal(t, tc.expected.Type, out.Type)
			assert.Equal(t, tc.expected.Name, out.Name)
			assert.Equal(t, tc.expected.Namespace, out.Namespace)
			assert.Equal(t, tc.expected.BucketName, out.BucketName)
			assert.Equal(t, tc.expected.Region, out.Region)
			require.NotNil(t, out.Url)
			assert.Equal(t, *tc.expected.Url, *out.Url)
			require.NotNil(t, out.Description)
			assert.Equal(t, *tc.expected.Description, *out.Description)
			if tc.expected.VerifyTLS != nil {
				require.NotNil(t, out.VerifyTLS)
				assert.Equal(t, *tc.expected.VerifyTLS, *out.VerifyTLS)
			}
			if tc.expected.ForcePathStyle != nil {
				require.NotNil(t, out.ForcePathStyle)
				assert.Equal(t, *tc.expected.ForcePathStyle, *out.ForcePathStyle)
			}
		})
	}
}

func TestMonitoringInstance_FromCR(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name     string
		input    *v1alpha1.MonitoringConfig
		expected MonitoringInstance
	}

	allowedNS := []string{"ns1", "ns2"}

	tcases := []tcase{
		{
			name: "pmm monitoring with all fields",
			input: &v1alpha1.MonitoringConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pmm-monitor",
					Namespace: "monitoring-ns",
				},
				Spec: v1alpha1.MonitoringConfigSpec{
					Type:              v1alpha1.PMMMonitoringType,
					AllowedNamespaces: allowedNS,
					VerifyTLS:         pointer.To(true),
					PMM: v1alpha1.PMMConfig{
						URL: "https://pmm.example.com",
					},
				},
			},
			expected: MonitoringInstance{
				Name:              "pmm-monitor",
				Namespace:         "monitoring-ns",
				Url:               "https://pmm.example.com",
				Type:              MonitoringInstanceBaseWithNameType(v1alpha1.PMMMonitoringType),
				AllowedNamespaces: &allowedNS,
				VerifyTLS:         pointer.To(true),
			},
		},
		{
			name: "minimal monitoring config",
			input: &v1alpha1.MonitoringConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "min-monitor",
					Namespace: "default",
				},
				Spec: v1alpha1.MonitoringConfigSpec{
					Type: v1alpha1.PMMMonitoringType,
					PMM: v1alpha1.PMMConfig{
						URL: "https://pmm.local",
					},
				},
			},
			expected: MonitoringInstance{
				Name:              "min-monitor",
				Namespace:         "default",
				Url:               "https://pmm.local",
				Type:              MonitoringInstanceBaseWithNameType(v1alpha1.PMMMonitoringType),
				AllowedNamespaces: nil,
			},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out MonitoringInstance
			out.FromCR(tc.input)
			assert.Equal(t, tc.expected.Name, out.Name)
			assert.Equal(t, tc.expected.Namespace, out.Namespace)
			assert.Equal(t, tc.expected.Url, out.Url)
			assert.Equal(t, tc.expected.Type, out.Type)
			if tc.expected.VerifyTLS != nil {
				require.NotNil(t, out.VerifyTLS)
				assert.Equal(t, *tc.expected.VerifyTLS, *out.VerifyTLS)
			}
			require.NotNil(t, out.AllowedNamespaces)
		})
	}
}

func TestStorageClass_FromCR(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name     string
		input    *v1.StorageClass
		expAllow bool
		expName  string
	}

	tcases := []tcase{
		{
			name: "with volume expansion enabled",
			input: &v1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "fast-ssd",
					Annotations: map[string]string{
						"storageclass.kubernetes.io/is-default-class": "true",
					},
					Labels: map[string]string{
						"tier": "premium",
					},
				},
				AllowVolumeExpansion: pointer.To(true),
			},
			expAllow: true,
			expName:  "fast-ssd",
		},
		{
			name: "with volume expansion disabled",
			input: &v1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "standard",
				},
				AllowVolumeExpansion: pointer.To(false),
			},
			expAllow: false,
			expName:  "standard",
		},
		{
			name: "nil volume expansion defaults to false",
			input: &v1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
			expAllow: false,
			expName:  "default",
		},
		{
			name: "with nil annotations and labels",
			input: &v1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bare-class",
				},
				AllowVolumeExpansion: pointer.To(true),
			},
			expAllow: true,
			expName:  "bare-class",
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out StorageClass
			out.FromCR(tc.input)

			// Verify AllowVolumeExpansion
			require.NotNil(t, out.AllowVolumeExpansion)
			assert.Equal(t, tc.expAllow, *out.AllowVolumeExpansion)

			// Verify Metadata
			require.NotNil(t, out.Metadata)
			meta := *out.Metadata
			assert.Equal(t, tc.expName, meta["name"])

			// Annotations and labels should always be present in metadata
			// (even if nil from the source object)
			_, hasAnnotations := meta["annotations"]
			assert.True(t, hasAnnotations, "metadata should contain 'annotations' key")

			_, hasLabels := meta["labels"]
			assert.True(t, hasLabels, "metadata should contain 'labels' key")

			// Verify annotations content when present
			if tc.input.GetAnnotations() != nil {
				annotations, ok := meta["annotations"].(map[string]string)
				require.True(t, ok)
				assert.Equal(t, tc.input.GetAnnotations(), annotations)
			}
		})
	}
}
