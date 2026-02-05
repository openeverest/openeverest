package rbac

import (
	"context"
	"slices"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/rbac"
)

func TestRBAC_DatabaseEngines(t *testing.T) {
	t.Run("ListDatabaseEngines", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := handlers.MockHandler{}
			h.On("ListDatabaseEngines", mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseEngineList{
				Items: []everestv1alpha1.DatabaseEngine{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      common.MySQLOperatorName,
							Namespace: "default",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      common.PostgreSQLOperatorName,
							Namespace: "default",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      common.MongoDBOperatorName,
							Namespace: "default",
						},
					},
				},
			}, nil,
			)
			return &h
		}

		testCases := []struct {
			desc   string
			policy string
			assert func(*everestv1alpha1.DatabaseEngineList) bool
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *everestv1alpha1.DatabaseEngineList) bool {
					return len(list.Items) == 3 &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MySQLOperatorName
						}) &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.PostgreSQLOperatorName
						}) &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MongoDBOperatorName
						})
				},
			},
			{
				desc: "read all in default namespace",
				policy: newPolicy(
					"p, role:test, database-engines, read, default/*",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseEngineList) bool {
					return len(list.Items) == 3 &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MySQLOperatorName
						}) &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.PostgreSQLOperatorName
						}) &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MongoDBOperatorName
						})
				},
			},
			{
				desc: "read all in all namespaces",
				policy: newPolicy(
					"p, role:test, database-engines, read, */*",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseEngineList) bool {
					return len(list.Items) == 3 &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MySQLOperatorName
						}) &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.PostgreSQLOperatorName
						}) &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MongoDBOperatorName
						})
				},
			},
			{
				desc: "read only pxc in default namespaces",
				policy: newPolicy(
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseEngineList) bool {
					return len(list.Items) == 1 &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MySQLOperatorName
						})
				},
			},
			{
				desc: "read only pg in default namespaces",
				policy: newPolicy(
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseEngineList) bool {
					return len(list.Items) == 1 &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.PostgreSQLOperatorName
						})
				},
			},
			{
				desc: "read only psmdb in default namespaces",
				policy: newPolicy(
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseEngineList) bool {
					return len(list.Items) == 1 &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MongoDBOperatorName
						})
				},
			},
			{
				desc: "read only psmdb & pg in default namespaces",
				policy: newPolicy(
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseEngineList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MongoDBOperatorName
						}) &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MongoDBOperatorName
						})
				},
			},
			{
				desc: "read only psmdb in all namespaces",
				policy: newPolicy(
					"p, role:test, database-engines, read, */percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseEngineList) bool {
					return len(list.Items) == 1 &&
						slices.ContainsFunc(list.Items, func(engine everestv1alpha1.DatabaseEngine) bool {
							return engine.GetName() == common.MongoDBOperatorName
						})
				},
			},
			{
				desc: "read all in kube-system namespace",
				policy: newPolicy(
					"p, role:test, database-engines, read, kube-system/*",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseEngineList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc: "no policy",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseEngineList) bool {
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

				list, err := h.ListDatabaseEngines(ctx, "default")
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetDatabaseEngine", func(t *testing.T) {
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
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing read permission on database-engine",
				policy: newPolicy(
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		next := func() *handlers.MockHandler {
			h := handlers.MockHandler{}
			h.On("GetDatabaseEngine", mock.Anything, mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseEngine{}, nil)
			return &h
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

				_, err = h.GetDatabaseEngine(ctx, "default", common.MySQLOperatorName)
				assert.ErrorIs(t, tc.wantErr, err)
			})
		}
	})

	t.Run("UpdateDatabaseEngine", func(t *testing.T) {
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
					"p, role:test, database-engines, update, default/percona-xtradb-cluster-operator",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing update permission on database-engine",
				policy: newPolicy(
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		next := func() *handlers.MockHandler {
			h := handlers.MockHandler{}
			h.On("UpdateDatabaseEngine", mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseEngine{}, nil)
			return &h
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

				_, err = h.UpdateDatabaseEngine(ctx, &everestv1alpha1.DatabaseEngine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      common.MySQLOperatorName,
						Namespace: "default",
					},
				})
				assert.ErrorIs(t, tc.wantErr, err)
			})
		}
	})

	t.Run("GetUpgradePlan", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := handlers.MockHandler{}
			h.On("GetUpgradePlan", mock.Anything, "default").Return(&api.UpgradePlan{
				Upgrades: &[]api.Upgrade{
					{Name: pointer.ToString(common.MySQLOperatorName)},
					{Name: pointer.ToString(common.PostgreSQLOperatorName)},
					{Name: pointer.ToString(common.MongoDBOperatorName)},
				},
			}, nil,
			)
			return &h
		}

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
			assert  func(*api.UpgradePlan) bool
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(up *api.UpgradePlan) bool {
					upgrades := pointer.Get(up.Upgrades)
					return len(upgrades) == 3 &&
						slices.ContainsFunc(upgrades, func(upgrade api.Upgrade) bool {
							return pointer.GetString(upgrade.Name) == common.MySQLOperatorName
						}) &&
						slices.ContainsFunc(upgrades, func(upgrade api.Upgrade) bool {
							return pointer.GetString(upgrade.Name) == common.PostgreSQLOperatorName
						}) &&
						slices.ContainsFunc(upgrades, func(upgrade api.Upgrade) bool {
							return pointer.GetString(upgrade.Name) == common.MongoDBOperatorName
						})
				},
			},
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/*",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				assert: func(up *api.UpgradePlan) bool {
					upgrades := pointer.Get(up.Upgrades)
					return len(upgrades) == 3 &&
						slices.ContainsFunc(upgrades, func(upgrade api.Upgrade) bool {
							return pointer.GetString(upgrade.Name) == common.MySQLOperatorName
						}) &&
						slices.ContainsFunc(upgrades, func(upgrade api.Upgrade) bool {
							return pointer.GetString(upgrade.Name) == common.PostgreSQLOperatorName
						}) &&
						slices.ContainsFunc(upgrades, func(upgrade api.Upgrade) bool {
							return pointer.GetString(upgrade.Name) == common.MongoDBOperatorName
						})
				},
			},
			{
				desc: "missing read permission on all(*) database-cluster",
				policy: newPolicy(
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission on all(*) database-cluster (only on some)",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/some",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission on one or more database-engines (psmdb)",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/*",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"g, bob, role:test",
				),
				assert: func(up *api.UpgradePlan) bool {
					return len(pointer.Get(up.Upgrades)) == 0
				},
			},
			{
				desc: "missing read permission on one or more database-engines (pg)",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/*",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				assert: func(up *api.UpgradePlan) bool {
					return len(pointer.Get(up.Upgrades)) == 0
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

				res, err := h.GetUpgradePlan(ctx, "default")
				assert.ErrorIs(t, tc.wantErr, err)
				if tc.assert != nil {
					assert.Condition(t, func() bool {
						return tc.assert(res)
					})
				}
			})
		}
	})

	t.Run("ApproveUpgradePlan", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := handlers.MockHandler{}
			h.On("GetUpgradePlan", mock.Anything, "default").Return(&api.UpgradePlan{
				Upgrades: &[]api.Upgrade{
					{Name: pointer.ToString(common.MySQLOperatorName)},
					{Name: pointer.ToString(common.PostgreSQLOperatorName)},
					{Name: pointer.ToString(common.MongoDBOperatorName)},
				},
			}, nil,
			)
			h.On("ApproveUpgradePlan", mock.Anything, "default").Return(nil)
			return &h
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
					"p, role:test, database-clusters, read, default/*",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",

					"p, role:test, database-engines, update, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, update, default/percona-postgresql-operator",
					"p, role:test, database-engines, update, default/percona-server-mongodb-operator",

					"g, bob, role:test",
				),
			},
			{
				desc: "missing read permission on all(*) database-cluster",
				policy: newPolicy(
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission on all(*) database-cluster (only on some)",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/some",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing update permission on all(*) database-engines",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/some",

					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",

					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing update permission on some database-engines",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/some",

					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",

					"p, role:test, database-engines, update, default/percona-xtradb-cluster-operator",
					"p, role:test, database-engines, update, default/percona-server-mongodb-operator",

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

				err = h.ApproveUpgradePlan(ctx, "default")
				assert.ErrorIs(t, tc.wantErr, err)
			})
		}
	})
}
