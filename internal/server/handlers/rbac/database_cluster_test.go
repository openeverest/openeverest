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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/rbac"
)

//nolint:gocyclo
func TestRBAC_DatabaseCluster(t *testing.T) {
	t.Parallel()

	t.Run("CreateDatabaseCluster - PXC", func(t *testing.T) {
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
		}{
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"g, bob, role:test",
				),
			},
			{
				desc: "success (wildcards)",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/*",
					"p, role:test, database-engines, read, default/*",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/*",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/*",
					"p, role:test, database-cluster-backups, read, default/*",
					"p, role:test, database-cluster-restores, read, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "success (admin)",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "missing create permission for database-cluster",
				policy:  newPolicy(), // no policy
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create permission for database-cluster (wrong namespace)",
				policy: newPolicy(
					"p, role:test, database-clusters, create, some/*",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create permission for database-cluster (wrong object)",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/some",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permissions for database-engine",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create database-cluster-backups permissions",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read backup-storages permissions",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create database-cluster-restore permissions",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing database-cluster-credentials permissions on source cluster",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read database-cluster-backups permissions on source cluster",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read database-cluster-restores permissions on source cluster",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases { //nolint:dupl
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)

				next := &handlers.MockHandler{}
				next.On("GetDatabaseClusterBackup", mock.Anything, mock.Anything, mock.Anything).Return(
					&everestv1alpha1.DatabaseClusterBackup{
						Spec: everestv1alpha1.DatabaseClusterBackupSpec{
							DBClusterName: "source-cluster",
						},
					}, nil,
				)
				next.On("CreateDatabaseCluster", mock.Anything, mock.Anything).
					Return(&everestv1alpha1.DatabaseCluster{}, nil)

				h := &rbacHandler{
					next:       next,
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				_, err = h.CreateDatabaseCluster(ctx, &everestv1alpha1.DatabaseCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Spec: everestv1alpha1.DatabaseClusterSpec{
						Engine: everestv1alpha1.Engine{
							Type: everestv1alpha1.DatabaseEnginePXC,
						},
						Backup: everestv1alpha1.Backup{
							Schedules: []everestv1alpha1.BackupSchedule{
								{
									BackupStorageName: "test-backup-storage",
								},
							},
						},
						DataSource: &everestv1alpha1.DataSource{
							DBClusterBackupName: "test-backup",
						},
					},
				})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("CreateDatabaseCluster - PSMDB", func(t *testing.T) {
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
		}{
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
			},
			{
				desc: "success (wildcards)",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/*",
					"p, role:test, database-engines, read, default/*",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/*",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/*",
					"p, role:test, database-cluster-backups, read, default/*",
					"p, role:test, database-cluster-restores, read, default/*",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "success (admin)",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "missing create permission for database-cluster",
				policy:  newPolicy(), // no policy
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create permission for database-cluster (wrong namespace)",
				policy: newPolicy(
					"p, role:test, database-clusters, create, some/*",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create permission for database-cluster (wrong object)",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/some",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permissions for database-engine",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create database-cluster-backups permissions",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read backup-storages permissions",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create database-cluster-restore permissions",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing database-cluster-credentials permissions on source cluster",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read database-cluster-backups permissions on source cluster",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-restores, read, default/source-cluster",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read database-cluster-restores permissions on source cluster",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read enginefeatures/split-horizon-dns-configs permissions on source cluster",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, database-cluster-restores, create, default/*",
					"p, role:test, database-cluster-credentials, read, default/source-cluster",
					"p, role:test, database-cluster-backups, read, default/source-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases { //nolint:dupl
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)

				next := &handlers.MockHandler{}
				next.On("GetDatabaseClusterBackup", mock.Anything, mock.Anything, mock.Anything).Return(
					&everestv1alpha1.DatabaseClusterBackup{
						Spec: everestv1alpha1.DatabaseClusterBackupSpec{
							DBClusterName: "source-cluster",
						},
					}, nil,
				)
				next.On("CreateDatabaseCluster", mock.Anything, mock.Anything).
					Return(&everestv1alpha1.DatabaseCluster{}, nil)

				h := &rbacHandler{
					next:       next,
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				_, err = h.CreateDatabaseCluster(ctx, &everestv1alpha1.DatabaseCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Spec: everestv1alpha1.DatabaseClusterSpec{
						Engine: everestv1alpha1.Engine{
							Type: everestv1alpha1.DatabaseEnginePSMDB,
						},
						Backup: everestv1alpha1.Backup{
							Schedules: []everestv1alpha1.BackupSchedule{
								{
									BackupStorageName: "test-backup-storage",
								},
							},
						},
						DataSource: &everestv1alpha1.DataSource{
							DBClusterBackupName: "test-backup",
						},
						EngineFeatures: &everestv1alpha1.EngineFeatures{
							PSMDB: &everestv1alpha1.PSMDBEngineFeatures{
								SplitHorizonDNSConfigName: "split-horizon-dns-config-1",
							},
						},
					},
				})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("UpdateDatabaseCluster - PXC", func(t *testing.T) {
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
		}{
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-clusters, update, default/test-cluster",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"g, bob, role:test",
				),
			},
			{
				desc: "success (admin)",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "missing update permission for database-cluster",
				policy:  newPolicy(), // no policy
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-engine",
				policy: newPolicy(
					"p, role:test, database-clusters, update, default/test-cluster",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create permission for database-cluster-backups (schedules updated)",
				policy: newPolicy(
					"p, role:test, database-clusters, update, default/test-cluster",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for backup-storages",
				policy: newPolicy(
					"p, role:test, database-clusters, update, default/test-cluster",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, database-cluster-backups, create, default/*",
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

				next := &handlers.MockHandler{}
				next.On("GetDatabaseCluster", mock.Anything, mock.Anything, mock.Anything).
					Return(&everestv1alpha1.DatabaseCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-cluster",
							Namespace: "default",
						},
					}, nil,
					)
				next.On("UpdateDatabaseCluster", mock.Anything, mock.Anything).
					Return(&everestv1alpha1.DatabaseCluster{}, nil)

				h := &rbacHandler{
					next:       next,
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				_, err = h.UpdateDatabaseCluster(ctx, &everestv1alpha1.DatabaseCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster",
						Namespace: "default",
					},
					Spec: everestv1alpha1.DatabaseClusterSpec{
						Engine: everestv1alpha1.Engine{
							Type: everestv1alpha1.DatabaseEnginePXC,
						},
						Backup: everestv1alpha1.Backup{
							Schedules: []everestv1alpha1.BackupSchedule{
								{
									BackupStorageName: "test-backup-storage",
								},
							},
						},
					},
				})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("UpdateDatabaseCluster - PSMDB", func(t *testing.T) {
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
		}{
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-clusters, update, default/test-cluster",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
			},
			{
				desc: "success (admin)",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "missing update permission for database-cluster",
				policy:  newPolicy(), // no policy
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-engine",
				policy: newPolicy(
					"p, role:test, database-clusters, update, default/test-cluster",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create permission for database-cluster-backups (schedules updated)",
				policy: newPolicy(
					"p, role:test, database-clusters, update, default/test-cluster",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for backup-storages",
				policy: newPolicy(
					"p, role:test, database-clusters, update, default/test-cluster",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for enginefeatures/split-horizon-dns-configs",
				policy: newPolicy(
					"p, role:test, database-clusters, update, default/test-cluster",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, database-cluster-backups, create, default/*",
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

				next := &handlers.MockHandler{}
				next.On("GetDatabaseCluster", mock.Anything, mock.Anything, mock.Anything).
					Return(&everestv1alpha1.DatabaseCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-cluster",
							Namespace: "default",
						},
					}, nil,
					)
				next.On("UpdateDatabaseCluster", mock.Anything, mock.Anything).
					Return(&everestv1alpha1.DatabaseCluster{}, nil)

				h := &rbacHandler{
					next:       next,
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				_, err = h.UpdateDatabaseCluster(ctx, &everestv1alpha1.DatabaseCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster",
						Namespace: "default",
					},
					Spec: everestv1alpha1.DatabaseClusterSpec{
						Engine: everestv1alpha1.Engine{
							Type: everestv1alpha1.DatabaseEnginePSMDB,
						},
						Backup: everestv1alpha1.Backup{
							Schedules: []everestv1alpha1.BackupSchedule{
								{
									BackupStorageName: "test-backup-storage",
								},
							},
						},
						EngineFeatures: &everestv1alpha1.EngineFeatures{
							PSMDB: &everestv1alpha1.PSMDBEngineFeatures{
								SplitHorizonDNSConfigName: "split-horizon-dns-config-1",
							},
						},
					},
				})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("GetDatabaseCluster - PXC", func(t *testing.T) { //nolint:dupl
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
		}{
			{
				desc: "success (admin)",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing read permission for database-cluster",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for backup-storages",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for backup-storages (pitr)",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for monitoring-instances",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-engines",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for enginefeatures/split-horizon-dns-configs",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"g, bob, role:test",
				),
				wantErr: nil,
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)

				next := &handlers.MockHandler{}
				next.On("GetDatabaseCluster", mock.Anything, mock.Anything, mock.Anything).
					Return(&everestv1alpha1.DatabaseCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-cluster",
							Namespace: "default",
						},
						Spec: everestv1alpha1.DatabaseClusterSpec{
							Engine: everestv1alpha1.Engine{
								Type: everestv1alpha1.DatabaseEnginePXC,
							},
							Backup: everestv1alpha1.Backup{
								Schedules: []everestv1alpha1.BackupSchedule{
									{
										BackupStorageName: "test-backup-storage",
									},
								},
								PITR: everestv1alpha1.PITRSpec{
									BackupStorageName: pointer.To("test-backup-storage-pitr"),
								},
							},
							Monitoring: &everestv1alpha1.Monitoring{
								MonitoringConfigName: "test-monitoring-instance",
							},
							EngineFeatures: &everestv1alpha1.EngineFeatures{
								PSMDB: &everestv1alpha1.PSMDBEngineFeatures{
									SplitHorizonDNSConfigName: "split-horizon-dns-config-1",
								},
							},
						},
					}, nil,
					)

				h := &rbacHandler{
					next:       next,
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				_, err = h.GetDatabaseCluster(ctx, "default", "test-cluster")
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("GetDatabaseCluster - PSMDB", func(t *testing.T) { //nolint:dupl
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
		}{
			{
				desc: "success (admin)",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-1",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing read permission for database-cluster",
				policy: newPolicy(
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for backup-storages",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for backup-storages (pitr)",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for monitoring-instances",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-engines",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for enginefeatures/split-horizon-dns-configs",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, backup-storages, read, default/test-backup-storage",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
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

				next := &handlers.MockHandler{}
				next.On("GetDatabaseCluster", mock.Anything, mock.Anything, mock.Anything).
					Return(&everestv1alpha1.DatabaseCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-cluster",
							Namespace: "default",
						},
						Spec: everestv1alpha1.DatabaseClusterSpec{
							Engine: everestv1alpha1.Engine{
								Type: everestv1alpha1.DatabaseEnginePSMDB,
							},
							Backup: everestv1alpha1.Backup{
								Schedules: []everestv1alpha1.BackupSchedule{
									{
										BackupStorageName: "test-backup-storage",
									},
								},
								PITR: everestv1alpha1.PITRSpec{
									BackupStorageName: pointer.To("test-backup-storage-pitr"),
								},
							},
							Monitoring: &everestv1alpha1.Monitoring{
								MonitoringConfigName: "test-monitoring-instance",
							},
							EngineFeatures: &everestv1alpha1.EngineFeatures{
								PSMDB: &everestv1alpha1.PSMDBEngineFeatures{
									SplitHorizonDNSConfigName: "split-horizon-dns-config-1",
								},
							},
						},
					}, nil,
					)

				h := &rbacHandler{
					next:       next,
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				_, err = h.GetDatabaseCluster(ctx, "default", "test-cluster")
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("ListDatabaseClusters", func(t *testing.T) {
		// setup mocks.
		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("ListDatabaseClusters", mock.Anything, mock.Anything).Return(
				&everestv1alpha1.DatabaseClusterList{
					Items: []everestv1alpha1.DatabaseCluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-cluster-1",
								Namespace: "default",
							},
							Spec: everestv1alpha1.DatabaseClusterSpec{
								Engine: everestv1alpha1.Engine{
									Type: everestv1alpha1.DatabaseEnginePXC,
								},
								Backup: everestv1alpha1.Backup{
									Schedules: []everestv1alpha1.BackupSchedule{
										{
											BackupStorageName: "test-backup-storage-1",
										},
									},
									PITR: everestv1alpha1.PITRSpec{
										BackupStorageName: pointer.To("test-backup-storage-pitr-1"),
									},
								},
								Monitoring: &everestv1alpha1.Monitoring{
									MonitoringConfigName: "test-monitoring-instance-1",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-cluster-2",
								Namespace: "default",
							},
							Spec: everestv1alpha1.DatabaseClusterSpec{
								Engine: everestv1alpha1.Engine{
									Type: everestv1alpha1.DatabaseEnginePostgresql,
								},
								Backup: everestv1alpha1.Backup{
									Schedules: []everestv1alpha1.BackupSchedule{
										{
											BackupStorageName: "test-backup-storage-2",
										},
									},
									PITR: everestv1alpha1.PITRSpec{
										BackupStorageName: pointer.To("test-backup-storage-pitr-2"),
									},
								},
								Monitoring: &everestv1alpha1.Monitoring{
									MonitoringConfigName: "test-monitoring-instance-2",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-cluster-3",
								Namespace: "default",
							},
							Spec: everestv1alpha1.DatabaseClusterSpec{
								Engine: everestv1alpha1.Engine{
									Type: everestv1alpha1.DatabaseEnginePSMDB,
								},
								Backup: everestv1alpha1.Backup{
									Schedules: []everestv1alpha1.BackupSchedule{
										{
											BackupStorageName: "test-backup-storage-3",
										},
									},
									PITR: everestv1alpha1.PITRSpec{
										BackupStorageName: pointer.To("test-backup-storage-pitr-3"),
									},
								},
								Monitoring: &everestv1alpha1.Monitoring{
									MonitoringConfigName: "test-monitoring-instance-3",
								},
								EngineFeatures: &everestv1alpha1.EngineFeatures{
									PSMDB: &everestv1alpha1.PSMDBEngineFeatures{
										SplitHorizonDNSConfigName: "split-horizon-dns-config-3",
									},
								},
							},
						},
					},
				},
				nil,
			)
			return h
		}

		testCases := []struct {
			desc   string
			policy string
			assert func(res *everestv1alpha1.DatabaseClusterList) bool
		}{
			{
				desc: "all permissions",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-1",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-1",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",

					"p, role:test, database-clusters, read, default/test-cluster-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-2",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-2",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",

					"p, role:test, database-clusters, read, default/test-cluster-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-3",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-3",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-3",

					"g, bob, role:test",
				),
				assert: func(res *everestv1alpha1.DatabaseClusterList) bool {
					mustContain := []string{"test-cluster-1", "test-cluster-2", "test-cluster-3"}
					for _, name := range mustContain {
						if ok := slices.ContainsFunc(res.Items, func(item everestv1alpha1.DatabaseCluster) bool {
							return item.Name == name
						}); !ok {
							return false
						}
					}
					return len(res.Items) == 3
				},
			},
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(res *everestv1alpha1.DatabaseClusterList) bool {
					mustContain := []string{"test-cluster-1", "test-cluster-2", "test-cluster-3"}
					for _, name := range mustContain {
						if ok := slices.ContainsFunc(res.Items, func(item everestv1alpha1.DatabaseCluster) bool {
							return item.Name == name
						}); !ok {
							return false
						}
					}
					return len(res.Items) == 3
				},
			},
			{
				desc: "no permission for pxc database-engine",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-1",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-1",

					"p, role:test, database-clusters, read, default/test-cluster-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-2",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-2",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",

					"p, role:test, database-clusters, read, default/test-cluster-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-3",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-3",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-3",

					"g, bob, role:test",
				),
				assert: func(res *everestv1alpha1.DatabaseClusterList) bool {
					mustContain := []string{"test-cluster-2", "test-cluster-3"}
					for _, name := range mustContain {
						if ok := slices.ContainsFunc(res.Items, func(item everestv1alpha1.DatabaseCluster) bool {
							return item.Name == name
						}); !ok {
							return false
						}
					}
					return len(res.Items) == 2
				},
			},
			{
				desc: "no permission for pg database-engine",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-1",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-1",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",

					"p, role:test, database-clusters, read, default/test-cluster-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-2",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-2",

					"p, role:test, database-clusters, read, default/test-cluster-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-3",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-3",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-3",

					"g, bob, role:test",
				),
				assert: func(res *everestv1alpha1.DatabaseClusterList) bool {
					mustContain := []string{"test-cluster-1", "test-cluster-3"}
					for _, name := range mustContain {
						if ok := slices.ContainsFunc(res.Items, func(item everestv1alpha1.DatabaseCluster) bool {
							return item.Name == name
						}); !ok {
							return false
						}
					}
					return len(res.Items) == 2
				},
			},
			{
				desc: "no permission for psmdb database-engine",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-1",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-1",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",

					"p, role:test, database-clusters, read, default/test-cluster-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-2",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-2",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",

					"p, role:test, database-clusters, read, default/test-cluster-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-3",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-3",

					"g, bob, role:test",
				),
				assert: func(res *everestv1alpha1.DatabaseClusterList) bool {
					mustContain := []string{"test-cluster-1", "test-cluster-2"}
					for _, name := range mustContain {
						if ok := slices.ContainsFunc(res.Items, func(item everestv1alpha1.DatabaseCluster) bool {
							return item.Name == name
						}); !ok {
							return false
						}
					}
					return len(res.Items) == 2
				},
			},
			{
				desc: "no permission for test-backup-storage-1",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-1",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-1",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",

					"p, role:test, database-clusters, read, default/test-cluster-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-2",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-2",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",

					"p, role:test, database-clusters, read, default/test-cluster-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-3",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-3",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-3",

					"g, bob, role:test",
				),
				assert: func(res *everestv1alpha1.DatabaseClusterList) bool {
					mustContain := []string{"test-cluster-2", "test-cluster-3"}
					for _, name := range mustContain {
						if ok := slices.ContainsFunc(res.Items, func(item everestv1alpha1.DatabaseCluster) bool {
							return item.Name == name
						}); !ok {
							return false
						}
					}
					return len(res.Items) == 2
				},
			},
			{
				desc: "no permission for test-backup-storage-pitr-1",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-1",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-1",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",

					"p, role:test, database-clusters, read, default/test-cluster-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-2",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-2",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",

					"p, role:test, database-clusters, read, default/test-cluster-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-3",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-3",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-3",

					"g, bob, role:test",
				),
				assert: func(res *everestv1alpha1.DatabaseClusterList) bool {
					mustContain := []string{"test-cluster-2", "test-cluster-3"}
					for _, name := range mustContain {
						if ok := slices.ContainsFunc(res.Items, func(item everestv1alpha1.DatabaseCluster) bool {
							return item.Name == name
						}); !ok {
							return false
						}
					}
					return len(res.Items) == 2
				},
			},
			{
				desc: "no permission for test-monitoring-instance-1",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-1",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",

					"p, role:test, database-clusters, read, default/test-cluster-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-2",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-2",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",

					"p, role:test, database-clusters, read, default/test-cluster-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-3",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-3",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",
					"p, role:test, enginefeatures/split-horizon-dns-configs, read, default/split-horizon-dns-config-3",

					"g, bob, role:test",
				),
				assert: func(res *everestv1alpha1.DatabaseClusterList) bool {
					mustContain := []string{"test-cluster-2", "test-cluster-3"}
					for _, name := range mustContain {
						if ok := slices.ContainsFunc(res.Items, func(item everestv1alpha1.DatabaseCluster) bool {
							return item.Name == name
						}); !ok {
							return false
						}
					}
					return len(res.Items) == 2
				},
			},
			{
				desc: "no permission for split-horizon-dns-config-3",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-1",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-1",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-1",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",

					"p, role:test, database-clusters, read, default/test-cluster-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-2",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-2",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-2",
					"p, role:test, database-engines, read, default/percona-postgresql-operator",

					"p, role:test, database-clusters, read, default/test-cluster-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-3",
					"p, role:test, backup-storages, read, default/test-backup-storage-pitr-3",
					"p, role:test, monitoring-instances, read, default/test-monitoring-instance-3",
					"p, role:test, database-engines, read, default/percona-server-mongodb-operator",

					"g, bob, role:test",
				),
				assert: func(res *everestv1alpha1.DatabaseClusterList) bool {
					mustContain := []string{"test-cluster-1", "test-cluster-2"}
					for _, name := range mustContain {
						if ok := slices.ContainsFunc(res.Items, func(item everestv1alpha1.DatabaseCluster) bool {
							return item.Name == name
						}); !ok {
							return false
						}
					}
					return len(res.Items) == 2
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

				h := &rbacHandler{
					next:       next(),
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				res, err := h.ListDatabaseClusters(ctx, "default")
				assert.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(res)
				})
			})
		}
	})

	t.Run("DeleteDatabaseCluster", func(t *testing.T) {
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
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
					"p, role:test, database-clusters, delete, default/test-cluster",
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing delete permission for database-cluster",
				policy: newPolicy(
					"p, role:test, database-engines, read, default/percona-xtradb-cluster-operator",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-engine",
				policy: newPolicy(
					"p, role:test, database-clusters, delete, default/test-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("DeleteDatabaseCluster", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			h.On("GetDatabaseCluster", mock.Anything, mock.Anything, mock.Anything).Return(
				&everestv1alpha1.DatabaseCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster",
						Namespace: "default",
					},
					Spec: everestv1alpha1.DatabaseClusterSpec{
						Engine: everestv1alpha1.Engine{
							Type: everestv1alpha1.DatabaseEnginePXC,
						},
					},
				},
				nil,
			)
			return h
		}
		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)

				h := &rbacHandler{
					next:       next(),
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				err = h.DeleteDatabaseCluster(ctx, "default", "test-cluster", &api.DeleteDatabaseClusterParams{})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("GetDatabaseClusterCredentials", func(t *testing.T) {
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
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
					"p, role:test, database-clusters, read, default/test-cluster",
					"p, role:test, database-cluster-credentials, read, default/test-cluster",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing read permission for database-cluster",
				policy: newPolicy(
					"p, role:test, database-cluster-credentials, read, default/test-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing read permission for database-cluster-credentials",
				policy: newPolicy(
					"p, role:test, database-clusters, read, default/test-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("GetDatabaseClusterCredentials", mock.Anything, mock.Anything, mock.Anything).Return(
				&api.DatabaseClusterCredential{}, nil)
			return h
		}
		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)

				h := &rbacHandler{
					next:       next(),
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				_, err = h.GetDatabaseClusterCredentials(ctx, "default", "test-cluster")
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("GetDatabaseClusterComponents", func(t *testing.T) {
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
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
					"p, role:test, database-clusters, read, default/test-cluster",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing read permission for database-cluster",
				policy: newPolicy(
					"p, role:test, database-cluster-credentials, read, default/test-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("GetDatabaseClusterComponents", mock.Anything, mock.Anything, mock.Anything).Return(
				[]api.DatabaseClusterComponent{}, nil)
			return h
		}
		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)

				h := &rbacHandler{
					next:       next(),
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				_, err = h.GetDatabaseClusterComponents(ctx, "default", "test-cluster")
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("GetDatabaseClusterPitr", func(t *testing.T) {
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
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
					"p, role:test, database-clusters, read, default/test-cluster",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing read permission for database-cluster",
				policy: newPolicy(
					"p, role:test, database-cluster-credentials, read, default/test-cluster",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("GetDatabaseClusterPitr", mock.Anything, mock.Anything, mock.Anything).Return(
				&api.DatabaseClusterPitr{}, nil)
			return h
		}
		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)

				h := &rbacHandler{
					next:       next(),
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				_, err = h.GetDatabaseClusterPitr(ctx, "default", "test-cluster")
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("CreateDatabaseClusterSecret", func(t *testing.T) {
		testCases := []struct {
			desc    string
			wantErr error
			policy  string
		}{
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/test",
					"g, bob, role:test",
				),
			},
			{
				desc: "success (wildcards)",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "success (admin)",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "missing create permission for database-cluster",
				policy:  newPolicy(), // no policy
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create permission for database-cluster (wrong namespace)",
				policy: newPolicy(
					"p, role:test, database-clusters, create, some/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "missing create permission for database-cluster (wrong object)",
				policy: newPolicy(
					"p, role:test, database-clusters, create, default/some",
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

				next := &handlers.MockHandler{}
				next.On("CreateDatabaseClusterSecret",
					mock.Anything, "default", "test", &corev1.Secret{},
				).
					Return(&corev1.Secret{}, nil)

				h := &rbacHandler{
					next:       next,
					enforcer:   enf,
					log:        zap.NewNop().Sugar(),
					userGetter: testUserGetter,
				}
				_, err = h.CreateDatabaseClusterSecret(ctx, "default", "test", &corev1.Secret{})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})
}
