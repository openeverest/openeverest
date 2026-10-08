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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStorageClasses(t *testing.T) {
	t.Parallel()
	now := time.Now()
	testCases := []struct {
		name         string
		storagesList *storagev1.StorageClassList
		result       []string
	}{
		{
			name: "no-default",
			storagesList: &storagev1.StorageClassList{
				Items: []storagev1.StorageClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "local-storage",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cool-storage",
						},
					},
				},
			},
			result: []string{"local-storage", "cool-storage"},
		},
		{
			name: "default is the first item",
			storagesList: &storagev1.StorageClassList{
				Items: []storagev1.StorageClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "local-storage",
							Annotations: map[string]string{
								annotationStorageClassDefault: "true",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cool-storage",
						},
					},
				},
			},
			result: []string{"local-storage", "cool-storage"},
		},
		{
			name: "default is the middle item",
			storagesList: &storagev1.StorageClassList{
				Items: []storagev1.StorageClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cool-storage",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "local-storage",
							Annotations: map[string]string{
								annotationStorageClassDefault: "true",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "another-storage",
						},
					},
				},
			},
			result: []string{"local-storage", "cool-storage", "another-storage"},
		},
		{
			name: "default is the last item",
			storagesList: &storagev1.StorageClassList{
				Items: []storagev1.StorageClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cool-storage",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "another-storage",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "local-storage",
							Annotations: map[string]string{
								annotationStorageClassDefault: "true",
							},
						},
					},
				},
			},
			result: []string{"local-storage", "another-storage", "cool-storage"},
		},
		{
			name: "default annotation set to false is not swapped",
			storagesList: &storagev1.StorageClassList{
				Items: []storagev1.StorageClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cool-storage",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slow-storage",
							Annotations: map[string]string{
								annotationStorageClassDefault: "false",
							},
						},
					},
				},
			},
			result: []string{"cool-storage", "slow-storage"},
		},
		{
			name: "default annotation false does not displace true",
			storagesList: &storagev1.StorageClassList{
				Items: []storagev1.StorageClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cool-storage",
							Annotations: map[string]string{
								annotationStorageClassDefault: "true",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slow-storage",
							Annotations: map[string]string{
								annotationStorageClassDefault: "false",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "another-storage",
						},
					},
				},
			},
			result: []string{"cool-storage", "slow-storage", "another-storage"},
		},
		{
			name:         "nil storagesList returns empty slice",
			storagesList: nil,
			result:       []string{},
		},
		{
			name:         "empty storagesList returns empty slice",
			storagesList: &storagev1.StorageClassList{},
			result:       []string{},
		},
		{
			name: "multiple defaults picks the most recently created (newest is last)",
			storagesList: &storagev1.StorageClassList{
				Items: []storagev1.StorageClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "old-default",
							Annotations: map[string]string{
								annotationStorageClassDefault: annotationStorageClassDefaultValue,
							},
							CreationTimestamp: metav1.Time{Time: now.Add(-2 * time.Hour)},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "standard-storage",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "new-default",
							Annotations: map[string]string{
								annotationStorageClassDefault: annotationStorageClassDefaultValue,
							},
							CreationTimestamp: metav1.Time{Time: now},
						},
					},
				},
			},
			result: []string{"new-default", "standard-storage", "old-default"},
		},
		{
			name: "multiple defaults picks the most recently created (newest is first)",
			storagesList: &storagev1.StorageClassList{
				Items: []storagev1.StorageClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "newest-default",
							Annotations: map[string]string{
								annotationStorageClassDefault: annotationStorageClassDefaultValue,
							},
							CreationTimestamp: metav1.Time{Time: now},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "older-default",
							Annotations: map[string]string{
								annotationStorageClassDefault: annotationStorageClassDefaultValue,
							},
							CreationTimestamp: metav1.Time{Time: now.Add(-1 * time.Hour)},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "oldest-default",
							Annotations: map[string]string{
								annotationStorageClassDefault: annotationStorageClassDefaultValue,
							},
							CreationTimestamp: metav1.Time{Time: now.Add(-3 * time.Hour)},
						},
					},
				},
			},
			result: []string{"newest-default", "older-default", "oldest-default"},
		},
		{
			name: "multiple defaults with equal timestamps preserves deterministic order",
			storagesList: &storagev1.StorageClassList{
				Items: []storagev1.StorageClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "first-default",
							Annotations: map[string]string{
								annotationStorageClassDefault: annotationStorageClassDefaultValue,
							},
							CreationTimestamp: metav1.Time{Time: now},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "second-default",
							Annotations: map[string]string{
								annotationStorageClassDefault: annotationStorageClassDefaultValue,
							},
							CreationTimestamp: metav1.Time{Time: now},
						},
					},
				},
			},
			result: []string{"first-default", "second-default"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, storageClasses(tc.storagesList), tc.result)
		})
	}
}
