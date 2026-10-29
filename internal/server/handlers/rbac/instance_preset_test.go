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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	apicommon "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_InstancePreset(t *testing.T) {
	t.Parallel()

	preset := func(name string) *corev1alpha1.InstancePreset {
		return &corev1alpha1.InstancePreset{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: name}}
	}

	mockInstancePresets := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListInstancePresets", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.InstancePresetList{
				Items: []corev1alpha1.InstancePreset{
					*preset("small"),
					*preset("medium"),
					*preset("large"),
				},
			}, nil,
		)
		h.On("GetInstancePreset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(preset("small"), nil)
		h.On("CreateInstancePreset", mock.Anything, mock.Anything, mock.Anything).Return(preset("small"), nil)
		h.On("UpdateInstancePreset", mock.Anything, mock.Anything, mock.Anything).Return(preset("small"), nil)
		h.On("DeleteInstancePreset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}

	newHandler := func(t *testing.T, ctx context.Context, policy string) *rbacHandler {
		t.Helper()
		enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(policy), zap.NewNop().Sugar())
		require.NoError(t, err)
		return &rbacHandler{
			next:       mockInstancePresets(),
			log:        zap.NewNop().Sugar(),
			enforcer:   enf,
			userGetter: testUserGetter,
		}
	}

	//nolint:staticcheck // test usage only
	ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})

	t.Run("ListInstancePresets", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc   string
			policy string
			want   []string
		}{
			{
				desc:   "admin",
				policy: newPolicy("g, bob, role:admin"),
				want:   []string{"small", "medium", "large"},
			},
			{
				desc: "all presets in namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/default/*",
					"g, bob, role:test",
				),
				want: []string{"small", "medium", "large"},
			},
			{
				desc: "specific preset",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/default/small",
					"g, bob, role:test",
				),
				want: []string{"small"},
			},
			{
				desc: "create only",
				policy: newPolicy(
					"p, role:test, instance-presets, create, prod/default/*",
					"g, bob, role:test",
				),
				want: []string{},
			},
			{
				desc: "update only",
				policy: newPolicy(
					"p, role:test, instance-presets, update, prod/default/*",
					"g, bob, role:test",
				),
				want: []string{},
			},
			{
				desc: "delete only",
				policy: newPolicy(
					"p, role:test, instance-presets, delete, prod/default/*",
					"g, bob, role:test",
				),
				want: []string{},
			},
			{
				desc: "wrong namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/other/*",
					"g, bob, role:test",
				),
				want: []string{},
			},
			{
				desc: "wrong cluster",
				policy: newPolicy(
					"p, role:test, instance-presets, read, staging/default/*",
					"g, bob, role:test",
				),
				want: []string{},
			},
			{
				desc:   "no permissions",
				policy: newPolicy("g, bob, role:test"),
				want:   []string{},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(t, ctx, tc.policy)

				list, err := h.ListInstancePresets(ctx, "prod", "default", "")
				require.NoError(t, err)

				names := make([]string, 0, len(list.Items))
				for _, item := range list.Items {
					names = append(names, item.Name)
				}
				assert.Equal(t, tc.want, names)
			})
		}
	})

	t.Run("GetInstancePreset", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc:   "admin",
				policy: newPolicy("g, bob, role:admin"),
			},
			{
				desc: "all actions",
				policy: newPolicy(
					"p, role:test, instance-presets, *, prod/default/small",
					"g, bob, role:test",
				),
			},
			{
				desc: "exact match",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/default/small",
					"g, bob, role:test",
				),
			},
			{
				desc: "wildcard in namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "create only",
				policy: newPolicy(
					"p, role:test, instance-presets, create, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "update only",
				policy: newPolicy(
					"p, role:test, instance-presets, update, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "delete only",
				policy: newPolicy(
					"p, role:test, instance-presets, delete, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "wrong preset",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/default/medium",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "wrong namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/other/small",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				policy:  newPolicy("g, bob, role:test"),
				wantErr: ErrInsufficientPermissions,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(t, ctx, tc.policy)

				_, err := h.GetInstancePreset(ctx, "prod", "default", "small")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					return
				}
				require.NoError(t, err)
			})
		}
	})

	t.Run("CreateInstancePreset", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc:   "admin",
				policy: newPolicy("g, bob, role:admin"),
			},
			{
				desc: "all actions",
				policy: newPolicy(
					"p, role:test, instance-presets, *, prod/default/small",
					"g, bob, role:test",
				),
			},
			{
				desc: "exact match",
				policy: newPolicy(
					"p, role:test, instance-presets, create, prod/default/small",
					"g, bob, role:test",
				),
			},
			{
				desc: "wildcard in namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, create, prod/default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "read only",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "update only",
				policy: newPolicy(
					"p, role:test, instance-presets, update, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "delete only",
				policy: newPolicy(
					"p, role:test, instance-presets, delete, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "wrong preset",
				policy: newPolicy(
					"p, role:test, instance-presets, create, prod/default/medium",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "wrong namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, create, prod/other/small",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				policy:  newPolicy("g, bob, role:test"),
				wantErr: ErrInsufficientPermissions,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(t, ctx, tc.policy)

				_, err := h.CreateInstancePreset(ctx, "prod", preset("small"))
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					return
				}
				require.NoError(t, err)
			})
		}
	})

	t.Run("UpdateInstancePreset", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc:   "admin",
				policy: newPolicy("g, bob, role:admin"),
			},
			{
				desc: "all actions",
				policy: newPolicy(
					"p, role:test, instance-presets, *, prod/default/small",
					"g, bob, role:test",
				),
			},
			{
				desc: "exact match",
				policy: newPolicy(
					"p, role:test, instance-presets, update, prod/default/small",
					"g, bob, role:test",
				),
			},
			{
				desc: "wildcard in namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, update, prod/default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "read only",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "create only",
				policy: newPolicy(
					"p, role:test, instance-presets, create, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "delete only",
				policy: newPolicy(
					"p, role:test, instance-presets, delete, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "wrong preset",
				policy: newPolicy(
					"p, role:test, instance-presets, update, prod/default/medium",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "wrong namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, update, prod/other/small",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				policy:  newPolicy("g, bob, role:test"),
				wantErr: ErrInsufficientPermissions,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(t, ctx, tc.policy)

				_, err := h.UpdateInstancePreset(ctx, "prod", preset("small"))
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					return
				}
				require.NoError(t, err)
			})
		}
	})

	t.Run("DeleteInstancePreset", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc:   "admin",
				policy: newPolicy("g, bob, role:admin"),
			},
			{
				desc: "all actions",
				policy: newPolicy(
					"p, role:test, instance-presets, *, prod/default/small",
					"g, bob, role:test",
				),
			},
			{
				desc: "exact match",
				policy: newPolicy(
					"p, role:test, instance-presets, delete, prod/default/small",
					"g, bob, role:test",
				),
			},
			{
				desc: "wildcard in namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, delete, prod/default/*",
					"g, bob, role:test",
				),
			},
			{
				desc: "read only",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "create only",
				policy: newPolicy(
					"p, role:test, instance-presets, create, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "update only",
				policy: newPolicy(
					"p, role:test, instance-presets, update, prod/default/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "wrong preset",
				policy: newPolicy(
					"p, role:test, instance-presets, delete, prod/default/medium",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "wrong namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, delete, prod/other/small",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				policy:  newPolicy("g, bob, role:test"),
				wantErr: ErrInsufficientPermissions,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(t, ctx, tc.policy)

				err := h.DeleteInstancePreset(ctx, "prod", "default", "small")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					return
				}
				require.NoError(t, err)
			})
		}
	})
}

func TestRBAC_InstancePresetReferences(t *testing.T) {
	t.Parallel()

	const basePolicy = "p, role:test, instance-presets, *, prod/ns1/*\ng, bob, role:test"

	storage := func(name string) corev1alpha1.InstanceBackupStorage {
		return corev1alpha1.InstanceBackupStorage{
			StorageRef: apicommon.ObjectRef{
				Name: name,
			},
		}
	}
	withStorages := func(storages ...corev1alpha1.InstanceBackupStorage) *corev1alpha1.InstanceBackupSpec {
		return &corev1alpha1.InstanceBackupSpec{
			ClassRef: apicommon.ObjectRef{
				Name: "pbm",
			},
			Storages: storages,
		}
	}
	scheduled := func() *corev1alpha1.InstanceBackupSpec {
		s := storage("s1")
		s.Schedules = []corev1alpha1.InstanceBackupSchedule{
			{
				Name:    "daily",
				Enabled: true,
				Cron:    "0 0 * * *",
			},
		}
		b := withStorages(s)
		b.Enabled = true
		return b
	}
	presetWith := func(spec corev1alpha1.InstanceSpec) *corev1alpha1.InstancePreset {
		spec.ProviderRef = apicommon.ObjectRef{
			Name: "mysql",
		}
		return &corev1alpha1.InstancePreset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "small",
				Namespace: "ns1",
			},
			Spec: corev1alpha1.InstancePresetSpec{
				InstanceSpec: spec,
			},
		}
	}
	monitoringTo := func(names ...string) *corev1alpha1.InstanceMonitoringSpec {
		m := &corev1alpha1.InstanceMonitoringSpec{}
		for _, name := range names {
			m.Destinations = append(m.Destinations, corev1alpha1.InstanceMonitoringDestination{
				DestinationRef: apicommon.MonitoringDestinationRef{
					Name: name,
				},
			})
		}
		return m
	}
	userSecret := func(name string) *apicommon.SecretRef {
		return &apicommon.SecretRef{
			Name: name,
		}
	}
	// stored is what GetInstancePreset returns: already pointing at references
	// the caller in most cases below cannot read.
	stored := func() *corev1alpha1.InstancePreset {
		return presetWith(corev1alpha1.InstanceSpec{
			Version:       "8.0",
			UserSecretRef: userSecret("secret1"),
			Backup:        withStorages(storage("s1")),
			Monitoring:    monitoringTo("pmm1"),
		})
	}

	newHandler := func(t *testing.T, extraPolicy ...string) *rbacHandler {
		t.Helper()
		ctx := testUserContext(rbac.User{Subject: "bob"})
		enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(newPolicy(append([]string{basePolicy}, extraPolicy...)...)), zap.NewNop().Sugar())
		require.NoError(t, err)
		next := &handlers.MockHandler{}
		next.On("GetInstancePreset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(stored(), nil)
		next.On("CreateInstancePreset", mock.Anything, mock.Anything, mock.Anything).Return(stored(), nil)
		next.On("UpdateInstancePreset", mock.Anything, mock.Anything, mock.Anything).Return(stored(), nil)
		next.On("GetBackup", mock.Anything, mock.Anything, mock.Anything, "backup-1").Return(
			&backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backup-1",
					Namespace: "ns1",
				},
				Spec: backupv1alpha1.BackupSpec{
					Origin: backupv1alpha1.BackupOrigin{
						Type: backupv1alpha1.BackupOriginTypeInstance,
						InstanceRef: &apicommon.ObjectRef{
							Name: "source-db",
						},
					},
				},
			},
			nil,
		)
		return &rbacHandler{
			next:       next,
			log:        zap.NewNop().Sugar(),
			enforcer:   enf,
			userGetter: testUserGetter,
		}
	}

	fromBackup := &backupv1alpha1.DataSource{
		Type: backupv1alpha1.DataSourceTypeBackup,
		Backup: &backupv1alpha1.DataSourceBackup{
			BackupRef: apicommon.ObjectRef{
				Name: "backup-1",
			},
		},
	}
	fromPITR := &backupv1alpha1.DataSource{
		Type: backupv1alpha1.DataSourceTypePointInTime,
		PointInTime: &backupv1alpha1.DataSourcePointInTime{
			Source: backupv1alpha1.StreamSource{
				StorageRef: apicommon.ObjectRef{
					Name: "s1",
				},
				InstanceRef: &apicommon.ObjectRef{
					Name: "source-db",
				},
			},
		},
	}
	const (
		readProvider = "p, role:test, providers, read, prod/mysql"
		readClass    = "p, role:test, backup-classes, read, prod/pbm"
		readStorage1 = "p, role:test, backup-storages, read, prod/ns1/s1"
		readStorage2 = "p, role:test, backup-storages, read, prod/ns1/s2"
		readSecret1  = "p, role:test, secrets, read, prod/ns1/secret1" //nolint:gosec // test usage only
		readSecret2  = "p, role:test, secrets, read, prod/ns1/secret2" //nolint:gosec // test usage only
		readPMM1     = "p, role:test, monitoring-destinations, read, prod/ns1/pmm1"
		readPMM2     = "p, role:test, monitoring-destinations, read, prod/ns1/pmm2"
		readBackup   = "p, role:test, backups, read, prod/ns1/source-db"
		readConn     = "p, role:test, instances, read-connection, prod/ns1/source-db"
	)

	t.Run("CreateInstancePreset", func(t *testing.T) {
		t.Parallel()

		createCases := []struct {
			name    string
			spec    corev1alpha1.InstanceSpec
			policy  []string
			wantErr bool
		}{
			{
				name:   "provider readable",
				policy: []string{readProvider},
			},
			{
				name:    "provider not readable",
				wantErr: true,
			},
			{
				name:   "user secret readable",
				policy: []string{readProvider, readSecret1},
				spec: corev1alpha1.InstanceSpec{
					UserSecretRef: userSecret("secret1"),
				},
			},
			{
				name:   "user secret not readable",
				policy: []string{readProvider},
				spec: corev1alpha1.InstanceSpec{
					UserSecretRef: userSecret("secret1"),
				},
				wantErr: true,
			},
			{
				name:   "backup class readable",
				policy: []string{readProvider, readClass, readStorage1},
				spec: corev1alpha1.InstanceSpec{
					Backup: withStorages(storage("s1")),
				},
			},
			{
				name:   "backup class not readable",
				policy: []string{readProvider, readStorage1},
				spec: corev1alpha1.InstanceSpec{
					Backup: withStorages(storage("s1")),
				},
				wantErr: true,
			},
			{
				name:   "backup storage not readable",
				policy: []string{readProvider, readClass},
				spec: corev1alpha1.InstanceSpec{
					Backup: withStorages(storage("s1")),
				},
				wantErr: true,
			},
			{
				name:   "backup schedules do not require backups create",
				policy: []string{readProvider, readClass, readStorage1},
				spec: corev1alpha1.InstanceSpec{
					Backup: scheduled(),
				},
			},
			{
				name:   "monitoring destination readable",
				policy: []string{readProvider, readPMM1},
				spec: corev1alpha1.InstanceSpec{
					Monitoring: monitoringTo("pmm1"),
				},
			},
			{
				name:   "monitoring destination not readable",
				policy: []string{readProvider},
				spec: corev1alpha1.InstanceSpec{
					Monitoring: monitoringTo("pmm1"),
				},
				wantErr: true,
			},
			{
				name:   "data source backup readable",
				policy: []string{readProvider, readBackup, readConn},
				spec: corev1alpha1.InstanceSpec{
					DataSource: fromBackup,
				},
			},
			{
				name:   "data source backup without read-connection on the source",
				policy: []string{readProvider, readBackup},
				spec: corev1alpha1.InstanceSpec{
					DataSource: fromBackup,
				},
				wantErr: true,
			},
			{
				name:   "data source backup not readable",
				policy: []string{readProvider},
				spec: corev1alpha1.InstanceSpec{
					DataSource: fromBackup,
				},
				wantErr: true,
			},
			{
				name:   "data source PITR readable",
				policy: []string{readProvider, readStorage1, readBackup, readConn},
				spec: corev1alpha1.InstanceSpec{
					DataSource: fromPITR,
				},
			},
			{
				name:   "data source PITR storage not readable",
				policy: []string{readProvider, readBackup},
				spec: corev1alpha1.InstanceSpec{
					DataSource: fromPITR,
				},
				wantErr: true,
			},
			{
				name:   "data source PITR source backups not readable",
				policy: []string{readProvider, readStorage1},
				spec: corev1alpha1.InstanceSpec{
					DataSource: fromPITR,
				},
				wantErr: true,
			},
		}
		for _, tc := range createCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newHandler(t, tc.policy...).CreateInstancePreset(testUserContext(rbac.User{Subject: "bob"}), "prod", presetWith(tc.spec))
				if tc.wantErr {
					require.ErrorIs(t, err, ErrInsufficientPermissions)
					return
				}
				require.NoError(t, err)
			})
		}
	})

	t.Run("UpdateInstancePreset", func(t *testing.T) {
		t.Parallel()

		updateCases := []struct {
			name    string
			mutate  func(*corev1alpha1.InstancePreset)
			policy  []string
			wantErr bool
		}{
			{
				name: "untouched unreadable references do not block an unrelated edit",
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.Version = "8.4"
				},
			},
			{
				name: "changing provider requires reading it",
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.ProviderRef.Name = "postgres"
				},
				wantErr: true,
			},
			{
				name: "changing user secret requires reading it",
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.UserSecretRef = userSecret("secret2")
				},
				wantErr: true,
			},
			{
				name:   "changing to a readable user secret",
				policy: []string{readSecret2},
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.UserSecretRef = userSecret("secret2")
				},
			},
			{
				name: "adding a storage requires reading it",
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.Backup.Storages = append(p.Spec.Backup.Storages, storage("s2"))
				},
				wantErr: true,
			},
			{
				name:   "adding a readable storage",
				policy: []string{readStorage2},
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.Backup.Storages = append(p.Spec.Backup.Storages, storage("s2"))
				},
			},
			{
				name: "adding backup schedules does not require backups create",
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.Backup = scheduled()
				},
			},
			{
				name: "changing backup class requires reading it",
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.Backup.ClassRef.Name = "pgbackrest"
				},
				wantErr: true,
			},
			{
				name: "adding a monitoring destination requires reading it",
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.Monitoring = monitoringTo("pmm1", "pmm2")
				},
				wantErr: true,
			},
			{
				name:   "adding a readable monitoring destination",
				policy: []string{readPMM2},
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.Monitoring = monitoringTo("pmm1", "pmm2")
				},
			},
			{
				name: "adding a data source requires reading its backup",
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.DataSource = fromBackup
				},
				wantErr: true,
			},
			{
				name:   "adding a readable data source",
				policy: []string{readBackup, readConn},
				mutate: func(p *corev1alpha1.InstancePreset) {
					p.Spec.DataSource = fromBackup
				},
			},
		}
		for _, tc := range updateCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				preset := stored()
				tc.mutate(preset)
				_, err := newHandler(t, tc.policy...).UpdateInstancePreset(testUserContext(rbac.User{Subject: "bob"}), "prod", preset)
				if tc.wantErr {
					require.ErrorIs(t, err, ErrInsufficientPermissions)
					return
				}
				require.NoError(t, err)
			})
		}
	})
}
