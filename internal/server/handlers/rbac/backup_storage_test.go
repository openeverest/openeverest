package rbac

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
	"github.com/percona/everest/pkg/rbac"
)

func TestRBAC_BackupStorage(t *testing.T) {
	t.Parallel()

	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("ListBackupStorages",
			mock.Anything,
			mock.Anything,
		).Return(
			&everestv1alpha1.BackupStorageList{
				Items: []everestv1alpha1.BackupStorage{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backup-storage-1",
							Namespace: "default",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backup-storage-2",
							Namespace: "default",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backup-storage-3",
							Namespace: "default",
						},
					},
				},
			},
			nil,
		)
		return &next
	}

	t.Run("ListBackupStorages", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc   string
			policy string
			assert func(list *everestv1alpha1.BackupStorageList) bool
		}{
			{
				desc: "read-only for backup-storage-1 in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/backup-storage-1",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 1 &&
						slices.ContainsFunc(list.Items, func(bs everestv1alpha1.BackupStorage) bool {
							return bs.GetName() == "backup-storage-1"
						})
				},
			},
			{
				desc: "read-only for backup-storage-1 and backup-storage-2 only in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/backup-storage-1",
					"p, role:test, backup-storages, read, default/backup-storage-2",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(bs everestv1alpha1.BackupStorage) bool {
							return bs.GetName() == "backup-storage-1"
						}) &&
						slices.ContainsFunc(list.Items, func(bs everestv1alpha1.BackupStorage) bool {
							return bs.GetName() == "backup-storage-2"
						})
				},
			},
			{
				desc: "read-only for all in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/*",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 3 &&
						slices.ContainsFunc(list.Items, func(bs everestv1alpha1.BackupStorage) bool {
							return bs.GetName() == "backup-storage-1"
						}) &&
						slices.ContainsFunc(list.Items, func(bs everestv1alpha1.BackupStorage) bool {
							return bs.GetName() == "backup-storage-2"
						}) &&
						slices.ContainsFunc(list.Items, func(bs everestv1alpha1.BackupStorage) bool {
							return bs.GetName() == "backup-storage-3"
						})
				},
			},
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *everestv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 3 &&
						slices.ContainsFunc(list.Items, func(bs everestv1alpha1.BackupStorage) bool {
							return bs.GetName() == "backup-storage-1"
						}) &&
						slices.ContainsFunc(list.Items, func(bs everestv1alpha1.BackupStorage) bool {
							return bs.GetName() == "backup-storage-2"
						}) &&
						slices.ContainsFunc(list.Items, func(bs everestv1alpha1.BackupStorage) bool {
							return bs.GetName() == "backup-storage-3"
						})
				},
			},
			{
				desc:   "no policy",
				policy: newPolicy(),
				assert: func(list *everestv1alpha1.BackupStorageList) bool {
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
				next := data()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListBackupStorages(ctx, "default")
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetBackupStorage", func(t *testing.T) {
		t.Parallel()

		data := func() *handlers.MockHandler {
			next := handlers.MockHandler{}
			next.On("GetBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
				&everestv1alpha1.BackupStorage{}, nil,
			)
			return &next
		}

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc: "all actions for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, *, */*",
					"g, bob, role:test",
				),
			},
			{
				desc: "all actions for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, *, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "read-only for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "read-only for 'inaccessible-storage' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/inaccessible-storage",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "read-only for all backupstorages in kube-system namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, kube-system/*",
					"g, bob, role:test",
				),
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

				next := data()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}
				_, err = h.GetBackupStorage(ctx, "default", "backup-storage-1")
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("CreateBackupStorage", func(t *testing.T) {
		t.Parallel()
		next := func() *handlers.MockHandler {
			next := handlers.MockHandler{}
			next.On("CreateBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
				&everestv1alpha1.BackupStorage{}, nil,
			)
			return &next
		}

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc: "all actions for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, *, */*",
					"g, bob, role:test",
				),
			},
			{
				desc: "all actions for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, *, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "all actions for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, *, default/backup-storage-1",
					"g, bob, role:test",
				),
			},
			{
				desc: "create only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, create, */*",
					"g, bob, role:test",
				),
			},
			{
				desc: "create only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, create, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "create only action for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, create, default/backup-storage-1",
					"g, bob, role:test",
				),
			},
			{
				desc: "create only action for some in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, create, default/some",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "update only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, update, */*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "update only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, update, default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "update only action for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, update, default/backup-storage-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "read only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, read, */*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "read only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "read only action for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/backup-storage-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "delete only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, */*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "delete only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "delete only action for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, default/backup-storage-1",
					"g, bob, role:test",
				),
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
				next := next()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}
				_, err = h.CreateBackupStorage(ctx, "default", &api.CreateBackupStorageParams{
					Name: "backup-storage-1",
				})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("UpdateBackupStorage", func(t *testing.T) {
		t.Parallel()
		next := func() *handlers.MockHandler {
			next := handlers.MockHandler{}
			next.On("UpdateBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
				&everestv1alpha1.BackupStorage{}, nil,
			)
			return &next
		}

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc: "all actions for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, *, */*",
					"g, bob, role:test",
				),
			},
			{
				desc: "all actions for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, *, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "all actions for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, *, default/backup-storage-1",
					"g, bob, role:test",
				),
			},
			{
				desc: "update only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, update, */*",
					"g, bob, role:test",
				),
			},
			{
				desc: "update only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, update, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "update only action for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, update, default/backup-storage-1",
					"g, bob, role:test",
				),
			},
			{
				desc: "update only action for some in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, update, default/some",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "read only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, read, */*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "read only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "read only action for some backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/some",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "create only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, create, */*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "create only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, create, default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "create only action for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, create, default/backup-storage-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "delete only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, */*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "delete only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "delete only action for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, default/backup-storage-1",
					"g, bob, role:test",
				),
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
				next := next()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}
				_, err = h.UpdateBackupStorage(ctx, "default", "backup-storage-1", &api.UpdateBackupStorageParams{})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("DeleteBackupStorage", func(t *testing.T) {
		t.Parallel()
		next := func() *handlers.MockHandler {
			next := handlers.MockHandler{}
			next.On("DeleteBackupStorage", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			return &next
		}

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc: "all actions for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, *, */*",
					"g, bob, role:test",
				),
			},
			{
				desc: "all actions for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, *, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "all actions for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, *, default/backup-storage-1",
					"g, bob, role:test",
				),
			},
			{
				desc: "delete only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, */*",
					"g, bob, role:test",
				),
			},
			{
				desc: "delete only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "delete only action for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, default/backup-storage-1",
					"g, bob, role:test",
				),
			},
			{
				desc: "delete only action for some in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, delete, default/some",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "update only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, update, */*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "update only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, update, default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "update only action for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, update, default/backup-storage-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "read only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, read, */*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "read only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "read only action for 'backup-storage-1' in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/backup-storage-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "create only action for all backupstorages in all namespaces",
				policy: newPolicy(
					"p, role:test, backup-storages, create, */*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "create only action for all backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, create, default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "create only action for some backupstorages in default namespace",
				policy: newPolicy(
					"p, role:test, backup-storages, create, default/some",
					"g, bob, role:test",
				),
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
				next := next()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}
				err = h.DeleteBackupStorage(ctx, "default", "backup-storage-1")
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})
}

func newConfigMapMock(policy string) kubernetes.KubernetesConnector {
	mockClient := fakeclient.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithObjects(newConfigMapPolicy(policy))
	return kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient.Build())
}

func newPolicy(lines ...string) string {
	return strings.Join(lines, "\n")
}

func newConfigMapPolicy(policy string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.SystemNamespace,
			Name:      common.EverestRBACConfigMapName,
		},
		Data: map[string]string{
			"enabled":    "true",
			"policy.csv": policy,
		},
	}
}

func testUserGetter(ctx context.Context) (rbac.User, error) {
	user, ok := ctx.Value(common.UserCtxKey).(rbac.User)
	if !ok {
		return rbac.User{}, errors.New("user not found in context")
	}
	return user, nil
}
