package k8s

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

func TestStorageClasses(t *testing.T) {
	t.Parallel()
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
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, storageClasses(tc.storagesList), tc.result)
		})
	}
}

func TestCreatedAtSort(t *testing.T) {
	t.Parallel()
	now := time.Now()

	input := []everestv1alpha1.DatabaseClusterBackup{
		{
			Status: everestv1alpha1.DatabaseClusterBackupStatus{
				CreatedAt: &metav1.Time{Time: now.Add(-2 * time.Second)},
			},
		},
		{
			Status: everestv1alpha1.DatabaseClusterBackupStatus{
				CreatedAt: &metav1.Time{Time: now.Add(-1 * time.Second)},
			},
		},
		{
			Status: everestv1alpha1.DatabaseClusterBackupStatus{
				CreatedAt: &metav1.Time{Time: now},
			},
		},
	}
	sorted := []everestv1alpha1.DatabaseClusterBackup{
		{
			Status: everestv1alpha1.DatabaseClusterBackupStatus{
				CreatedAt: &metav1.Time{Time: now},
			},
		},
		{
			Status: everestv1alpha1.DatabaseClusterBackupStatus{
				CreatedAt: &metav1.Time{Time: now.Add(-1 * time.Second)},
			},
		},
		{
			Status: everestv1alpha1.DatabaseClusterBackupStatus{
				CreatedAt: &metav1.Time{Time: now.Add(-2 * time.Second)},
			},
		},
	}

	slices.SortFunc(input, sortFunc)
	assert.Equal(t, sorted, input)
}
