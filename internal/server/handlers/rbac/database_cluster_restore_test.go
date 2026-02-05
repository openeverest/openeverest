package rbac

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/rbac"
)

func TestRBAC_DatabaseClusterRestore(t *testing.T) {
	t.Run("ListDatabaseClusterRestores", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("ListDatabaseClusterRestores", mock.Anything, mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseClusterRestoreList{
				Items: []everestv1alpha1.DatabaseClusterRestore{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "restore1",
							Namespace: "default",
						},
						Spec: everestv1alpha1.DatabaseClusterRestoreSpec{
							DBClusterName: "cluster1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "restore2",
							Namespace: "default",
						},
						Spec: everestv1alpha1.DatabaseClusterRestoreSpec{
							DBClusterName: "cluster1",
						},
					},
				},
			}, nil,
			)
			return h
		}

		testCases := []struct {
			desc   string
			policy string
			assert func(*everestv1alpha1.DatabaseClusterRestoreList) bool
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *everestv1alpha1.DatabaseClusterRestoreList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(restore everestv1alpha1.DatabaseClusterRestore) bool {
							return restore.GetName() == "restore1"
						}) &&
						slices.ContainsFunc(list.Items, func(restore everestv1alpha1.DatabaseClusterRestore) bool {
							return restore.GetName() == "restore2"
						})
				},
			},
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseClusterRestoreList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(restore everestv1alpha1.DatabaseClusterRestore) bool {
							return restore.GetName() == "restore1"
						}) &&
						slices.ContainsFunc(list.Items, func(restore everestv1alpha1.DatabaseClusterRestore) bool {
							return restore.GetName() == "restore2"
						})
				},
			},
			{
				desc:   "missing read permission for database-cluster-restores on 'cluster1'",
				policy: newPolicy(),
				assert: func(list *everestv1alpha1.DatabaseClusterRestoreList) bool {
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
				next := next()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListDatabaseClusterRestores(ctx, "default", "cluster1")
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetDatabaseClusterRestore", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("GetDatabaseClusterRestore", mock.Anything, mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseClusterRestore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "restore1",
					Namespace: "default",
				},
				Spec: everestv1alpha1.DatabaseClusterRestoreSpec{
					DBClusterName: "cluster1",
				},
			}, nil,
			)
			return h
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
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "missing read permission for database-cluster-restores on 'cluster1'",
				policy:  newPolicy(),
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

				_, err = h.GetDatabaseClusterRestore(ctx, "default", "restore1")
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("CreateDatabaseClusterRestore", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("CreateDatabaseClusterRestore", mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseClusterRestore{}, nil)
			return h
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
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, create, default/cluster1",
					"p, role:test, database-cluster-credentials, read, default/cluster1",
					"p, role:test, database-cluster-backups, read, default/cluster1",
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing create permission for database-cluster-restores on 'cluster1'",
				policy: newPolicy(
					"p, role:test, database-cluster-credentials, read, default/cluster1",
					"p, role:test, database-cluster-backups, read, default/cluster1",
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-cluster-credentials on 'cluster1'",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, create, default/cluster1",
					"p, role:test, database-cluster-backups, read, default/cluster1",
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-cluster-backups on 'cluster1'",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, create, default/cluster1",
					"p, role:test, database-cluster-credentials, read, default/cluster1",
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-cluster-restores on 'cluster1'",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, create, default/cluster1",
					"p, role:test, database-cluster-credentials, read, default/cluster1",
					"p, role:test, database-cluster-backups, read, default/cluster1",
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

				_, err = h.CreateDatabaseClusterRestore(ctx, &everestv1alpha1.DatabaseClusterRestore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "restore1",
						Namespace: "default",
					},
					Spec: everestv1alpha1.DatabaseClusterRestoreSpec{
						DBClusterName: "cluster1",
					},
				})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("UpdateDatabaseClusterRestore", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("UpdateDatabaseClusterRestore", mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseClusterRestore{}, nil)
			return h
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
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, update, default/cluster1",
					"p, role:test, database-cluster-credentials, read, default/cluster1",
					"p, role:test, database-cluster-backups, read, default/cluster1",
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing update permission for database-cluster-restores on 'cluster1'",
				policy: newPolicy(
					"p, role:test, database-cluster-credentials, read, default/cluster1",
					"p, role:test, database-cluster-backups, read, default/cluster1",
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "has create permission for database-cluster-restores on 'cluster1'",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, create, default/cluster1",
					"p, role:test, database-cluster-credentials, read, default/cluster1",
					"p, role:test, database-cluster-backups, read, default/cluster1",
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-cluster-credentials on 'cluster1'",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, update, default/cluster1",
					"p, role:test, database-cluster-backups, read, default/cluster1",
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-cluster-backups on 'cluster1'",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, update, default/cluster1",
					"p, role:test, database-cluster-credentials, read, default/cluster1",
					"p, role:test, database-cluster-restores, read, default/cluster1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-cluster-restores on 'cluster1'",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, update, default/cluster1",
					"p, role:test, database-cluster-credentials, read, default/cluster1",
					"p, role:test, database-cluster-backups, read, default/cluster1",
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

				_, err = h.UpdateDatabaseClusterRestore(ctx, &everestv1alpha1.DatabaseClusterRestore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "restore1",
						Namespace: "default",
					},
					Spec: everestv1alpha1.DatabaseClusterRestoreSpec{
						DBClusterName: "cluster1",
					},
				})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("DeleteDatabaseClusterRestore", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("GetDatabaseClusterRestore", mock.Anything, mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseClusterRestore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "restore1",
					Namespace: "default",
				},
				Spec: everestv1alpha1.DatabaseClusterRestoreSpec{
					DBClusterName: "cluster1",
				},
			}, nil,
			)
			h.On("DeleteDatabaseClusterRestore", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			return h
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
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-cluster-restores, delete, default/cluster1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "missing delete permission for database-cluster-restores on 'cluster1'",
				policy:  newPolicy(),
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

				err = h.DeleteDatabaseClusterRestore(ctx, "default", "restore1")
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})
}
