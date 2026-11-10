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
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	valhandler "github.com/openeverest/openeverest/v2/internal/server/handlers/validation"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_BackupStorageV2(t *testing.T) {
	t.Parallel()

	mockBackupStorages := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListBackupStorages", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupStorageList{
				Items: []backupv1alpha1.BackupStorage{
					{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "gcs-backup", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "azure-archive", Namespace: "ns1"}},
				},
			}, nil,
		)
		h.On("GetBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
			nil,
		)
		h.On("CreateBackupStorage", mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
			nil,
		)
		h.On("UpdateBackupStorage", mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
			nil,
		)
		h.On("PatchBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
			nil,
		)
		h.On("DeleteBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}

	t.Run("ListBackupStorages", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			ns      string
			policy  string
			assert  func(list *backupv1alpha1.BackupStorageList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "all in namespace with wildcard",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "specific storage",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/s3-primary",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 1 && list.Items[0].Name == "s3-primary"
				},
			},
			{
				desc:    "two specific storages",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/s3-primary",
					"p, role:test, backup-storages, read, prod/ns1/gcs-backup",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(bs backupv1alpha1.BackupStorage) bool { return bs.Name == "s3-primary" }) &&
						slices.ContainsFunc(list.Items, func(bs backupv1alpha1.BackupStorage) bool { return bs.Name == "gcs-backup" })
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backup-storages, read, staging/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns2/*",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 0
				},
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockBackupStorages()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListBackupStorages(ctx, tc.cluster, tc.ns, handlers.BackupStorageFilter{})
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetBackupStorage", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:admin"),
			},
			{
				desc:    "exact match",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/s3-primary",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-storages, read, staging/ns1/s3-primary",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:test"),
				wantErr: ErrInsufficientPermissions,
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockBackupStorages()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetBackupStorage(ctx, tc.cluster, "ns1", "s3-primary")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "s3-primary", result.Name)
				}
			})
		}
	})

	t.Run("CreateBackupStorage", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:admin"),
			},
			{
				desc:    "has create permission and uses what it references",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-storages, create, prod/ns1/s3-primary",
					"p, role:test, backup-storage-classes, read, prod/s3",
					"p, role:test, secrets, use, prod/ns1/s3-keys",
					"p, role:test, secrets, use, prod/ns1/s3-ca",
					"g, bob, role:test",
				),
			},
			{
				desc:    "cannot read the backup storage class",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-storages, create, prod/ns1/s3-primary",
					"p, role:test, secrets, use, prod/ns1/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "cannot use the CA bundle Secret",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-storages, create, prod/ns1/s3-primary",
					"p, role:test, backup-storage-classes, read, prod/s3",
					"p, role:test, secrets, use, prod/ns1/s3-keys",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "read on the Secrets is not use",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-storages, create, prod/ns1/s3-primary",
					"p, role:test, backup-storage-classes, read, prod/s3",
					"p, role:test, secrets, read, prod/ns1/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:test"),
				wantErr: ErrInsufficientPermissions,
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockBackupStorages()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.CreateBackupStorage(ctx, tc.cluster, referencingBackupStorage())
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "s3-primary", result.Name)
				}
			})
		}
	})

	// An update needs use on the Secrets only when it changes spec, and then
	// on every Secret the updated storage names, including unchanged ones.
	t.Run("UpdateBackupStorage", func(t *testing.T) {
		t.Parallel()

		updatePolicy := []string{
			"p, role:test, backup-storages, update, prod/ns1/s3-primary",
			"p, role:test, backup-storage-classes, read, prod/*",
			"g, bob, role:test",
		}
		testCases := []struct {
			desc    string
			policy  string
			update  func(bs *backupv1alpha1.BackupStorage)
			wantErr error
			// wantVersion is the resourceVersion the update reaches the next
			// handler with.
			wantVersion string
		}{
			{
				desc:   "admin",
				policy: newPolicy("g, bob, role:admin"),
				update: func(bs *backupv1alpha1.BackupStorage) {
					bs.Spec.Parameters = endpointParameters("https://elsewhere.example.com")
				},
				wantVersion: storedBackupStorageVersion,
			},
			{
				desc:   "labels only needs no use",
				policy: newPolicy(updatePolicy...),
				update: func(bs *backupv1alpha1.BackupStorage) {
					bs.Labels = map[string]string{"team": "a"}
				},
				wantVersion: storedBackupStorageVersion,
			},
			{
				desc:   "unchanged parameters written in another key order need no use",
				policy: newPolicy(updatePolicy...),
				update: func(bs *backupv1alpha1.BackupStorage) {
					bs.Spec.Parameters = &runtime.RawExtension{Raw: []byte(`{ "region": "us-east-1", "endpointURL": "https://s3.example.com" }`)}
				},
				wantVersion: storedBackupStorageVersion,
			},
			{
				desc:   "a parameters change with the same Secrets needs use",
				policy: newPolicy(append(updatePolicy, "p, role:test, secrets, use, prod/ns1/*")...),
				update: func(bs *backupv1alpha1.BackupStorage) {
					bs.Spec.Parameters = endpointParameters("https://elsewhere.example.com")
				},
				wantVersion: storedBackupStorageVersion,
			},
			{
				desc:   "a parameters change without use is refused",
				policy: newPolicy(append(updatePolicy, "p, role:test, secrets, read, prod/ns1/*")...),
				update: func(bs *backupv1alpha1.BackupStorage) {
					bs.Spec.Parameters = endpointParameters("https://elsewhere.example.com")
				},
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:   "a new credentials Secret without use on the CA is refused",
				policy: newPolicy(append(updatePolicy, "p, role:test, secrets, use, prod/ns1/rotated")...),
				update: func(bs *backupv1alpha1.BackupStorage) {
					bs.Spec.Authentication.CredentialsSecretRef.Name = "rotated"
				},
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:   "a caller-supplied version is kept",
				policy: newPolicy(updatePolicy...),
				update: func(bs *backupv1alpha1.BackupStorage) {
					bs.Annotations = map[string]string{"note": "x"}
					bs.SetResourceVersion("41")
				},
				wantVersion: "41",
			},
			{
				desc:    "no permissions",
				policy:  newPolicy("g, bob, role:test"),
				update:  func(*backupv1alpha1.BackupStorage) {},
				wantErr: ErrInsufficientPermissions,
			},
		}

		ctx := testUserContext(rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockStoredBackupStorage()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				bs := storedBackupStorage()
				bs.SetResourceVersion("")
				tc.update(bs)
				result, err := h.UpdateBackupStorage(ctx, "prod", bs)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					next.AssertNotCalled(t, "UpdateBackupStorage", mock.Anything, mock.Anything, mock.Anything)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, "s3-primary", result.Name)
				sent, ok := next.Calls[len(next.Calls)-1].Arguments.Get(2).(*backupv1alpha1.BackupStorage)
				require.True(t, ok)
				assert.Equal(t, tc.wantVersion, sent.GetResourceVersion())
			})
		}
	})

	// A patch is authorised as an update, not as a permission of its own, so
	// read alone must not be enough to patch. use is checked on the storage
	// the patch produces, not on the Secrets the patch body names.
	t.Run("PatchBackupStorage", func(t *testing.T) {
		t.Parallel()

		updatePolicy := []string{
			"p, role:test, backup-storages, update, prod/ns1/s3-primary",
			"g, bob, role:test",
		}
		testCases := []struct {
			desc    string
			policy  string
			patch   string
			wantErr error
			// wantVersion is the resourceVersion the patch reaches the next
			// handler with.
			wantVersion string
		}{
			{
				desc:        "admin",
				policy:      newPolicy("g, bob, role:admin"),
				patch:       `{"spec":{"parameters":{"region":"eu-west-1"}}}`,
				wantVersion: storedBackupStorageVersion,
			},
			{
				desc: "has read but not update",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/s3-primary",
					"g, bob, role:test",
				),
				patch:   `{"metadata":{"annotations":{"note":"x"}}}`,
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:        "metadata only needs no use",
				policy:      newPolicy(updatePolicy...),
				patch:       `{"metadata":{"annotations":{"note":"x"}}}`,
				wantVersion: storedBackupStorageVersion,
			},
			{
				desc:        "a spec patch that changes nothing needs no use",
				policy:      newPolicy(updatePolicy...),
				patch:       `{"spec":{"parameters":{"endpointURL":"https://s3.example.com"},"authentication":{"credentialsSecretRef":{"name":"s3-keys"}}}}`,
				wantVersion: storedBackupStorageVersion,
			},
			{
				desc:    "a patch naming no Secret but changing the endpoint needs use",
				policy:  newPolicy(append(updatePolicy, "p, role:test, secrets, read, prod/ns1/*")...),
				patch:   `{"spec":{"parameters":{"endpointURL":"https://elsewhere.example.com"}}}`,
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:        "a patch naming no Secret but changing the endpoint with use",
				policy:      newPolicy(append(updatePolicy, "p, role:test, secrets, use, prod/ns1/s3-keys", "p, role:test, secrets, use, prod/ns1/s3-ca")...),
				patch:       `{"spec":{"parameters":{"endpointURL":"https://elsewhere.example.com"}}}`,
				wantVersion: storedBackupStorageVersion,
			},
			{
				desc:        "names a Secret it can use, and uses the stored CA",
				policy:      newPolicy(append(updatePolicy, "p, role:test, secrets, use, prod/ns1/rotated", "p, role:test, secrets, use, prod/ns1/s3-ca")...),
				patch:       `{"spec":{"authentication":{"credentialsSecretRef":{"name":"rotated"}}}}`,
				wantVersion: storedBackupStorageVersion,
			},
			{
				desc:    "names a Secret it can use, but not the stored CA",
				policy:  newPolicy(append(updatePolicy, "p, role:test, secrets, use, prod/ns1/rotated")...),
				patch:   `{"spec":{"authentication":{"credentialsSecretRef":{"name":"rotated"}}}}`,
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "names a Secret it cannot use",
				policy:  newPolicy(append(updatePolicy, "p, role:test, secrets, use, prod/ns1/s3-keys")...),
				patch:   `{"spec":{"tls":{"ca":{"secretKeyRef":{"name":"other-ca","key":"ca.crt"}}}}}`,
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:        "dropping the CA needs use only on the credentials",
				policy:      newPolicy(append(updatePolicy, "p, role:test, secrets, use, prod/ns1/s3-keys")...),
				patch:       `{"spec":{"tls":null}}`,
				wantVersion: storedBackupStorageVersion,
			},
			{
				desc:        "a caller-supplied version is kept",
				policy:      newPolicy(updatePolicy...),
				patch:       `{"metadata":{"resourceVersion":"41","annotations":{"note":"x"}}}`,
				wantVersion: "41",
			},
			{
				desc:    "is not JSON",
				policy:  newPolicy(updatePolicy...),
				patch:   `{"spec":`,
				wantErr: valhandler.ErrInvalidRequest,
			},
			{
				desc:    "is not an object",
				policy:  newPolicy(updatePolicy...),
				patch:   `null`,
				wantErr: valhandler.ErrInvalidRequest,
			},
			{
				desc:    "does not fit the type",
				policy:  newPolicy(updatePolicy...),
				patch:   `{"spec":{"authentication":"AccessKey"}}`,
				wantErr: valhandler.ErrInvalidRequest,
			},
		}

		ctx := testUserContext(rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockStoredBackupStorage()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.PatchBackupStorage(ctx, "prod", "ns1", "s3-primary", []byte(tc.patch))
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					next.AssertNotCalled(t, "PatchBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, "s3-primary", result.Name)

				sent, ok := next.Calls[len(next.Calls)-1].Arguments.Get(4).([]byte)
				require.True(t, ok)
				var doc map[string]any
				require.NoError(t, json.Unmarshal(sent, &doc))
				var want map[string]any
				require.NoError(t, json.Unmarshal([]byte(tc.patch), &want))
				// The patch is relayed as sent, pinned to the version checked.
				metadata, _ := want["metadata"].(map[string]any)
				if metadata == nil {
					metadata = make(map[string]any)
					want["metadata"] = metadata
				}
				metadata["resourceVersion"] = tc.wantVersion
				assert.Equal(t, want, doc)
			})
		}
	})

	t.Run("PatchBackupStorage of a missing storage", func(t *testing.T) {
		t.Parallel()

		ctx := testUserContext(rbac.User{Subject: "bob"})
		enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(newPolicy("g, bob, role:admin")), zap.NewNop().Sugar())
		require.NoError(t, err)
		notFound := k8serrors.NewNotFound(backupv1alpha1.GroupVersion.WithResource("backupstorages").GroupResource(), "s3-primary")
		next := &handlers.MockHandler{}
		next.On("GetBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, notFound)

		h := &rbacHandler{
			next:       next,
			log:        zap.NewNop().Sugar(),
			enforcer:   enf,
			userGetter: testUserGetter,
		}

		_, err = h.PatchBackupStorage(ctx, "prod", "ns1", "s3-primary", []byte(`{"metadata":{"annotations":{"note":"x"}}}`))
		require.True(t, k8serrors.IsNotFound(err))
		next.AssertNotCalled(t, "PatchBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("DeleteBackupStorage", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:admin"),
			},
			{
				desc:    "has delete permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, prod/ns1/s3-primary",
					"g, bob, role:test",
				),
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:test"),
				wantErr: ErrInsufficientPermissions,
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockBackupStorages()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				err = h.DeleteBackupStorage(ctx, tc.cluster, "ns1", "s3-primary")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}

const storedBackupStorageVersion = "7"

// storedBackupStorage is referencingBackupStorage as stored, with parameters
// and a resourceVersion.
func storedBackupStorage() *backupv1alpha1.BackupStorage {
	bs := referencingBackupStorage()
	bs.SetResourceVersion(storedBackupStorageVersion)
	bs.Spec.Parameters = endpointParameters("https://s3.example.com")
	return bs
}

func endpointParameters(endpoint string) *runtime.RawExtension {
	return &runtime.RawExtension{Raw: []byte(`{"endpointURL":"` + endpoint + `","region":"us-east-1"}`)}
}

// mockStoredBackupStorage returns a handler that has storedBackupStorage.
func mockStoredBackupStorage() *handlers.MockHandler {
	h := &handlers.MockHandler{}
	h.On("GetBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(storedBackupStorage(), nil)
	h.On("UpdateBackupStorage", mock.Anything, mock.Anything, mock.Anything).Return(
		&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
		nil,
	)
	h.On("PatchBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
		nil,
	)
	return h
}

func referencingBackupStorage() *backupv1alpha1.BackupStorage {
	return &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"},
		Spec: backupv1alpha1.BackupStorageSpec{
			BackupStorageClassRef: corev1alpha1.ObjectRef{Name: "s3"},
			Authentication: corev1alpha1.Authentication{
				Mode:                 "AccessKey",
				CredentialsSecretRef: &corev1alpha1.SecretRef{Name: "s3-keys"},
			},
			TLS: &corev1alpha1.TLSConfig{
				CA: &corev1alpha1.CABundleSource{SecretKeyRef: &corev1alpha1.SecretKeyRef{Name: "s3-ca", Key: "ca.crt"}},
			},
		},
	}
}
