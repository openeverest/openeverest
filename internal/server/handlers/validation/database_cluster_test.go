package validation

import (
	"context"
	"fmt"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	operatorUtils "github.com/percona/everest-operator/utils"
	"github.com/percona/everest/internal/server/handlers/k8s"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
)

func TestValidateCreateDatabaseClusterRequest(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name  string
		value everestv1alpha1.DatabaseCluster
		err   error
	}

	cases := []testCase{
		{
			name: "empty dbCluster name",
			value: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "",
					Namespace: "ns",
				},
			},
			err: operatorUtils.ErrNameNotRFC1035Compatible("metadata.name"),
		},
		{
			name: "starts with -",
			value: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "-rstrst",
					Namespace: "ns",
				},
			},
			err: operatorUtils.ErrNameNotRFC1035Compatible("metadata.name"),
		},
		{
			name: "ends with -",
			value: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rstrst-",
					Namespace: "ns",
				},
			},
			err: operatorUtils.ErrNameNotRFC1035Compatible("metadata.name"),
		},
		{
			name: "contains uppercase",
			value: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "AAsdf",
					Namespace: "ns",
				},
			},
			err: operatorUtils.ErrNameNotRFC1035Compatible("metadata.name"),
		},
		{
			name: "valid",
			value: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "amsdf-sllla",
					Namespace: "ns",
				},
			},
			err: nil,
		},
		{
			name: "dbCluster name too long",
			value: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "a123456789a123456789a123",
					Namespace: "ns",
				},
			},
			err: operatorUtils.ErrNameTooLong("metadata.name"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := validateCreateDatabaseClusterRequest(&c.value)
			if c.err == nil {
				require.NoError(t, err)
				return
			}
			assert.Equal(t, c.err.Error(), err.Error())
		})
	}
}

func TestValidateProxyType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		engineType everestv1alpha1.EngineType
		proxyType  everestv1alpha1.ProxyType
		err        error
	}{
		{
			name:       "PXC with mongos",
			engineType: everestv1alpha1.DatabaseEnginePXC,
			proxyType:  everestv1alpha1.ProxyTypeMongos,
			err:        errUnsupportedPXCProxy,
		},
		{
			name:       "PXC with pgbouncer",
			engineType: everestv1alpha1.DatabaseEnginePXC,
			proxyType:  everestv1alpha1.ProxyTypePGBouncer,
			err:        errUnsupportedPXCProxy,
		},
		{
			name:       "PXC with haproxy",
			engineType: everestv1alpha1.DatabaseEnginePXC,
			proxyType:  everestv1alpha1.ProxyTypeHAProxy,
			err:        nil,
		},
		{
			name:       "PXC with proxysql",
			engineType: everestv1alpha1.DatabaseEnginePXC,
			proxyType:  everestv1alpha1.ProxyTypeProxySQL,
			err:        nil,
		},
		{
			name:       "psmdb with mongos",
			engineType: everestv1alpha1.DatabaseEnginePSMDB,
			proxyType:  everestv1alpha1.ProxyTypeMongos,
			err:        nil,
		},
		{
			name:       "psmdb with pgbouncer",
			engineType: everestv1alpha1.DatabaseEnginePSMDB,
			proxyType:  everestv1alpha1.ProxyTypePGBouncer,
			err:        errUnsupportedPSMDBProxy,
		},
		{
			name:       "psmdb with haproxy",
			engineType: everestv1alpha1.DatabaseEnginePSMDB,
			proxyType:  everestv1alpha1.ProxyTypeHAProxy,
			err:        errUnsupportedPSMDBProxy,
		},
		{
			name:       "psmdb with proxysql",
			engineType: everestv1alpha1.DatabaseEnginePSMDB,
			proxyType:  everestv1alpha1.ProxyTypeProxySQL,
			err:        errUnsupportedPSMDBProxy,
		},
		{
			name:       "postgresql with mongos",
			engineType: everestv1alpha1.DatabaseEnginePostgresql,
			proxyType:  everestv1alpha1.ProxyTypeMongos,
			err:        errUnsupportedPGProxy,
		},
		{
			name:       "postgresql with pgbouncer",
			engineType: everestv1alpha1.DatabaseEnginePostgresql,
			proxyType:  everestv1alpha1.ProxyTypePGBouncer,
			err:        nil,
		},
		{
			name:       "postgresql with haproxy",
			engineType: everestv1alpha1.DatabaseEnginePostgresql,
			proxyType:  everestv1alpha1.ProxyTypeHAProxy,
			err:        errUnsupportedPGProxy,
		},
		{
			name:       "postgresql with proxysql",
			engineType: everestv1alpha1.DatabaseEnginePostgresql,
			proxyType:  everestv1alpha1.ProxyTypeProxySQL,
			err:        errUnsupportedPGProxy,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := validateProxyType(c.engineType, c.proxyType)
			if c.err == nil {
				require.NoError(t, err)
				return
			}
			assert.Equal(t, c.err.Error(), err.Error())
		})
	}
}

func TestValidateProxy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		cluster everestv1alpha1.DatabaseCluster
		err     error
	}{
		{
			name: "ok: 3 nodes 2 proxies",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:     everestv1alpha1.DatabaseEnginePXC,
						Replicas: 3,
					},
					Proxy: everestv1alpha1.Proxy{
						Type:     everestv1alpha1.ProxyTypeHAProxy,
						Replicas: pointer.ToInt32(2),
					},
				},
			},
			err: nil,
		},
		{
			name: "errMinProxyReplicas",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:     everestv1alpha1.DatabaseEnginePXC,
						Replicas: 3,
					},
					Proxy: everestv1alpha1.Proxy{
						Type:     everestv1alpha1.ProxyTypeHAProxy,
						Replicas: pointer.ToInt32(1),
					},
				},
			},
			err: errMinPXCProxyReplicas,
		},
		{
			name: "ok: 1 node 1 proxy",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:     everestv1alpha1.DatabaseEnginePXC,
						Replicas: 1,
					},
					Proxy: everestv1alpha1.Proxy{
						Type:     everestv1alpha1.ProxyTypeHAProxy,
						Replicas: pointer.ToInt32(1),
					},
				},
			},
			err: nil,
		},
		{
			name: "duplicated sourceRange",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:     everestv1alpha1.DatabaseEnginePXC,
						Replicas: 1,
					},
					Proxy: everestv1alpha1.Proxy{
						Type:     everestv1alpha1.ProxyTypeHAProxy,
						Replicas: pointer.ToInt32(1),
						Expose: everestv1alpha1.Expose{
							IPSourceRanges: []everestv1alpha1.IPSourceRange{
								"192.168.0.1/32",
								"192.168.0.2/32",
								"192.168.0.1/32",
							},
						},
					},
				},
			},
			err: ErrDuplicateSourceRange("192.168.0.1/32"),
		},
		{
			name: "no duplicated sourceRange",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:     everestv1alpha1.DatabaseEnginePXC,
						Replicas: 1,
					},
					Proxy: everestv1alpha1.Proxy{
						Type:     everestv1alpha1.ProxyTypeHAProxy,
						Replicas: pointer.ToInt32(1),
						Expose: everestv1alpha1.Expose{
							IPSourceRanges: []everestv1alpha1.IPSourceRange{
								"192.168.0.1/32",
								"192.168.0.2/32",
								"192.168.0.3/32",
							},
						},
					},
				},
			},
			err: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := validateProxy(&c.cluster)
			assert.Equal(t, c.err, err)
		})
	}
}

func TestValidateBackupSpec(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		cluster *everestv1alpha1.DatabaseCluster
		err     error
	}{
		{
			name:    "empty backup is allowed",
			cluster: &everestv1alpha1.DatabaseCluster{},
			err:     nil,
		},
		{
			name: "allow no schedules",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{},
					},
				},
			},
			err: nil,
		},
		{
			name: "errNoNameInSchedule",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{
								Enabled: true,
							},
						},
					},
				},
			},
			err: errNoNameInSchedule,
		},
		{
			name: "errNoBackupStorageName",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{
								Enabled: true,
								Name:    "name",
							},
						},
					},
				},
			},
			err: errScheduleNoBackupStorageName,
		},
		{
			name: "errDuplicatedSchedules",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{
								Enabled:           true,
								Name:              "name",
								Schedule:          "0 0 * * *",
								BackupStorageName: "some",
							},
							{
								Enabled:           true,
								Name:              "other",
								Schedule:          "0 0 * * *",
								BackupStorageName: "some",
							},
						},
					},
				},
			},
			err: errDuplicatedSchedules,
		},
		{
			name: "valid spec",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{
								Enabled:           true,
								Name:              "name",
								Schedule:          "0 0 * * *",
								BackupStorageName: "some",
							},
						},
					},
				},
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.err, validateBackupSpec(tc.cluster))
		})
	}
}

func TestValidateBackupStoragesFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		cluster everestv1alpha1.DatabaseCluster
		storage everestv1alpha1.BackupStorage
		err     error
	}{
		{
			name: "errPSMDBMultipleStorages",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "psmdb-try",
					Namespace: "ns-1",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{
								Enabled:           true,
								Name:              "name",
								BackupStorageName: "storage1",
							},
							{
								Enabled:           true,
								Name:              "name2",
								BackupStorageName: "storage2",
							},
						},
					},
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
			},
			storage: everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage1",
					Namespace: "ns-1",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Type: everestv1alpha1.BackupStorageTypeS3,
				},
			},
			err: errPSMDBMultipleStorages,
		},
		{
			name: "errPSMDBViolateActiveStorage",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "psmdb-try",
					Namespace: "ns-2",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{
								Enabled:           true,
								Name:              "name2",
								BackupStorageName: "storage2",
							},
						},
					},
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
				Status: everestv1alpha1.DatabaseClusterStatus{
					ActiveStorage: "storage1",
				},
			},
			storage: everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage2",
					Namespace: "ns-2",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Type: everestv1alpha1.BackupStorageTypeS3,
				},
			},
			err: errPSMDBViolateActiveStorage,
		},
		{
			name: "no errPSMDBViolateActiveStorage",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "psmdb-try",
					Namespace: "ns-3",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{
								Enabled:           true,
								Name:              "name2",
								BackupStorageName: "storage1",
							},
						},
					},
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
				Status: everestv1alpha1.DatabaseClusterStatus{
					ActiveStorage: "",
				},
			},
			storage: everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage1",
					Namespace: "ns-3",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Type: everestv1alpha1.BackupStorageTypeS3,
				},
			},
			err: nil,
		},
		{
			name: "errPXCPitrS3Only",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pxc-try",
					Namespace: "ns-4",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						PITR: everestv1alpha1.PITRSpec{
							Enabled:           true,
							BackupStorageName: pointer.To("storage"),
						},
						Schedules: []everestv1alpha1.BackupSchedule{
							{
								Enabled:           true,
								Name:              "otherName",
								BackupStorageName: "storage",
							},
						},
					},
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				Status: everestv1alpha1.DatabaseClusterStatus{
					ActiveStorage: "",
				},
			},
			storage: everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage",
					Namespace: "ns-4",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Type: everestv1alpha1.BackupStorageTypeAzure,
				},
			},
			err: errPXCPitrS3Only,
		},
		{
			name: "errPitrNoBackupStorageName",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pxc-try",
					Namespace: "ns-5",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						PITR: everestv1alpha1.PITRSpec{
							Enabled: true,
						},
						Schedules: []everestv1alpha1.BackupSchedule{
							{
								Enabled:           true,
								Name:              "otherName",
								BackupStorageName: "storage",
							},
						},
					},
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				Status: everestv1alpha1.DatabaseClusterStatus{
					ActiveStorage: "",
				},
			},
			storage: everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage1",
					Namespace: "ns-5",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Type: everestv1alpha1.BackupStorageTypeS3,
				},
			},
			err: errPitrNoBackupStorageName,
		},
		{
			name: "valid",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pxc-try",
					Namespace: "ns-6",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						PITR: everestv1alpha1.PITRSpec{
							Enabled:           true,
							BackupStorageName: pointer.To("storage"),
						},
						Schedules: []everestv1alpha1.BackupSchedule{
							{
								Enabled:           true,
								Name:              "otherName",
								BackupStorageName: "storage",
							},
						},
					},
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
				},
				Status: everestv1alpha1.DatabaseClusterStatus{
					ActiveStorage: "",
				},
			},
			storage: everestv1alpha1.BackupStorage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage",
					Namespace: "ns-6",
				},
				Spec: everestv1alpha1.BackupStorageSpec{
					Type: everestv1alpha1.BackupStorageTypeS3,
				},
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fakeclient.NewClientBuilder().WithScheme(kubernetes.CreateScheme()).WithObjects(&tc.cluster, &tc.storage)
			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient.Build())
			h := validateHandler{
				kubeConnector: k,
			}

			err := h.validateBackupStoragesFor(
				context.Background(),
				tc.cluster.GetNamespace(),
				&tc.cluster,
			)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestValidatePitrSpec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cluster everestv1alpha1.DatabaseCluster
		err     error
	}{
		{
			name: "valid spec pitr enabled",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						PITR: everestv1alpha1.PITRSpec{
							Enabled:           true,
							BackupStorageName: pointer.To("name"),
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "valid spec pitr disabled",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						PITR: everestv1alpha1.PITRSpec{
							Enabled: false,
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "valid spec no pitr",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{},
				},
			},
			err: nil,
		},
		{
			name: "no backup storage pxc",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						PITR: everestv1alpha1.PITRSpec{
							Enabled: true,
						},
					},
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
				},
			},
			err: errPitrNoBackupStorageName,
		},
		{
			name: "no backup storage psmdb",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						PITR: everestv1alpha1.PITRSpec{
							Enabled: true,
						},
					},
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePSMDB,
					},
				},
			},
			err: nil,
		},
		{
			name: "no backup storage pg",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						PITR: everestv1alpha1.PITRSpec{
							Enabled: true,
						},
					},
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			err: nil,
		},
		{
			name: "zero upload interval",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						PITR: everestv1alpha1.PITRSpec{
							Enabled:           true,
							BackupStorageName: pointer.To("name"),
							UploadIntervalSec: pointer.ToInt(0),
						},
					},
				},
			},
			err: errPitrUploadInterval,
		},
		{
			name: "negative upload interval",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						PITR: everestv1alpha1.PITRSpec{
							Enabled:           true,
							BackupStorageName: pointer.To("name"),
							UploadIntervalSec: pointer.ToInt(-100),
						},
					},
				},
			},
			err: errPitrUploadInterval,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.err, validatePitrSpec(&tc.cluster))
		})
	}
}

func TestValidateResourceLimits(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		cluster everestv1alpha1.DatabaseCluster
		err     error
	}{
		{
			name: "success",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Resources: everestv1alpha1.Resources{
							CPU:    resource.MustParse("600m"),
							Memory: resource.MustParse("1G"),
						},
						Storage: everestv1alpha1.Storage{
							Size: resource.MustParse("2G"),
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "errNoResourceDefined",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Storage: everestv1alpha1.Storage{
							Size: resource.MustParse("2G"),
						},
					},
				},
			},
			err: errNoResourceDefined,
		},
		{
			name: "Not enough CPU",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Resources: everestv1alpha1.Resources{
							Memory: resource.MustParse("1G"),
						},
						Storage: everestv1alpha1.Storage{
							Size: resource.MustParse("2G"),
						},
					},
				},
			},
			err: errNotEnoughCPU,
		},
		{
			name: "Not enough memory",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Resources: everestv1alpha1.Resources{
							CPU: resource.MustParse("600m"),
						},
						Storage: everestv1alpha1.Storage{
							Size: resource.MustParse("2G"),
						},
					},
				},
			},
			err: errNotEnoughMemory,
		},
		{
			name: "not enough disk size",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Resources: everestv1alpha1.Resources{
							CPU:    resource.MustParse("600m"),
							Memory: resource.MustParse("1G"),
						},
						Storage: everestv1alpha1.Storage{
							Size: resource.MustParse("512M"),
						},
					},
				},
			},
			err: errNotEnoughDiskSize,
		},
		{
			name: "not enough CPU",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Resources: everestv1alpha1.Resources{
							CPU:    resource.MustParse("200m"),
							Memory: resource.MustParse("1G"),
						},
						Storage: everestv1alpha1.Storage{
							Size: resource.MustParse("512M"),
						},
					},
				},
			},
			err: errNotEnoughCPU,
		},
		{
			name: "not enough Mem",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Resources: everestv1alpha1.Resources{
							CPU:    resource.MustParse("600m"),
							Memory: resource.MustParse("400M"),
						},
						Storage: everestv1alpha1.Storage{
							Size: resource.MustParse("512M"),
						},
					},
				},
			},
			err: errNotEnoughMemory,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.err, validateResourceLimits(&tc.cluster))
		})
	}
}

func TestValidateDataSource(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		source everestv1alpha1.DataSource
		err    error
	}{
		{
			name:   "err none of the data source specified",
			source: everestv1alpha1.DataSource{},
			err:    errDataSourceConfig,
		},
		{
			name: "err both of the data source specified",
			source: everestv1alpha1.DataSource{
				DBClusterBackupName: "some-backup",
				BackupSource: &everestv1alpha1.BackupSource{
					BackupStorageName: "some-name",
					Path:              "some-path",
				},
			},
			err: errDataSourceConfig,
		},
		{
			name: "err no date in pitr",
			source: everestv1alpha1.DataSource{
				DBClusterBackupName: "some-backup",
				PITR:                &everestv1alpha1.PITR{},
			},
			err: errDataSourceNoPitrDateSpecified,
		},
		{
			name: "wrong pitr date format",
			source: everestv1alpha1.DataSource{
				DBClusterBackupName: "some-backup",
				PITR: &everestv1alpha1.PITR{
					Date: &everestv1alpha1.RestoreDate{
						Time: metav1.Time{},
					},
				},
			},
			err: errDataSourceWrongDateFormat,
		},
		{
			name: "correct minimal",
			source: everestv1alpha1.DataSource{
				DBClusterBackupName: "some-backup",
				PITR: &everestv1alpha1.PITR{
					Date: &everestv1alpha1.RestoreDate{
						Time: metav1.Time{Time: mustParseTime(t, "2024-01-02T15:04:05Z")},
					},
				},
			},
			err: nil,
		},
		{
			name: "correct with pitr type",
			source: everestv1alpha1.DataSource{
				DBClusterBackupName: "some-backup",
				PITR: &everestv1alpha1.PITR{
					Type: everestv1alpha1.PITRTypeDate,
					Date: &everestv1alpha1.RestoreDate{
						Time: metav1.Time{Time: mustParseTime(t, "2024-01-02T15:04:05Z")},
					},
				},
			},
			err: nil,
		},
		{
			name: "unsupported pitr type",
			source: everestv1alpha1.DataSource{
				DBClusterBackupName: "some-backup",
				PITR: &everestv1alpha1.PITR{
					Type: "latest",
					Date: &everestv1alpha1.RestoreDate{
						Time: metav1.Now(),
					},
				},
			},
			err: errUnsupportedPitrType,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(
				t,
				tc.err,
				validateDataSource(&tc.source),
			)
		})
	}
}

func TestValidatePGReposForAPIDB(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		cluster          everestv1alpha1.DatabaseCluster
		dbClusterBackups []ctrlclient.Object
		err              error
	}{
		{
			name: "ok: no schedules no backups",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some",
					Namespace: "ns-1",
				},
			},
			dbClusterBackups: []ctrlclient.Object{},
			err:              nil,
		},
		{
			name: "ok: 2 schedules 2 backups with the same storages",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some",
					Namespace: "ns-2",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{BackupStorageName: "bs1"},
							{BackupStorageName: "bs2"},
						},
					},
				},
			},
			dbClusterBackups: []ctrlclient.Object{
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-1",
						Namespace: "ns-2",
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs1"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-2",
						Namespace: "ns-2",
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs2"},
				},
			},
			err: nil,
		},
		{
			name: "error: 3 schedules in different bs and 1 backup in another bs",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some",
					Namespace: "ns-3",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{BackupStorageName: "bs1"},
							{BackupStorageName: "bs2"},
							{BackupStorageName: "bs3"},
						},
					},
				},
			},
			dbClusterBackups: []ctrlclient.Object{
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-4",
						Namespace: "ns-3",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs4"},
				},
			},
			err: errTooManyPGStorages,
		},
		{
			name: "ok: 3 schedules",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some",
					Namespace: "ns-4",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{BackupStorageName: "bs1"},
							{BackupStorageName: "bs2"},
							{BackupStorageName: "bs3"},
						},
					},
				},
			},
			dbClusterBackups: []ctrlclient.Object{},
			err:              nil,
		},
		{
			name: "ok: 3 backups with different storages",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some",
					Namespace: "ns-5",
				},
			},
			dbClusterBackups: []ctrlclient.Object{
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-1",
						Namespace: "ns-5",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs1"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-2",
						Namespace: "ns-5",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs2"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-3",
						Namespace: "ns-5",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs3"},
				},
			},
			err: nil,
		},
		{
			name: "ok: 5 backups with repeating storages",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some",
					Namespace: "ns-6",
				},
			},
			dbClusterBackups: []ctrlclient.Object{
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-1",
						Namespace: "ns-6",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs1"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-2",
						Namespace: "ns-6",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs2"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-3",
						Namespace: "ns-6",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs3"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-1-1",
						Namespace: "ns-6",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs1"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-2-1",
						Namespace: "ns-6",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs2"},
				},
			},
			err: nil,
		},
		{
			name: "error: 4 backups with different storages",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some",
					Namespace: "ns-7",
				},
			},
			dbClusterBackups: []ctrlclient.Object{
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-1",
						Namespace: "ns-7",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs1"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-2",
						Namespace: "ns-7",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs2"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-3",
						Namespace: "ns-7",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs3"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-4",
						Namespace: "ns-7",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs4"},
				},
			},
			err: errTooManyPGStorages,
		},
		{
			name: "ok: 4 backups with same storages",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some",
					Namespace: "ns-8",
				},
			},
			dbClusterBackups: []ctrlclient.Object{
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-1-1",
						Namespace: "ns-8",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs1"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-1-2",
						Namespace: "ns-8",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs1"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-2-1",
						Namespace: "ns-8",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs2"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-2-2",
						Namespace: "ns-8",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs2"},
				},
			},
			err: nil,
		},
		{
			name: "error: 4 schedules",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some",
					Namespace: "ns-9",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{BackupStorageName: "bs1"},
							{BackupStorageName: "bs2"},
							{BackupStorageName: "bs3"},
							{BackupStorageName: "bs4"},
						},
					},
				},
			},
			dbClusterBackups: []ctrlclient.Object{},
			err:              errTooManyPGStorages,
		},
		{
			name: "error: 2 schedules 2 backups with different storages",
			cluster: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some",
					Namespace: "ns-10",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{BackupStorageName: "bs1"},
							{BackupStorageName: "bs2"},
						},
					},
				},
			},
			dbClusterBackups: []ctrlclient.Object{
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-3",
						Namespace: "ns-10",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs3"},
				},
				&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bc-bs-4",
						Namespace: "ns-10",
						Labels:    map[string]string{common.DatabaseClusterNameLabel: "some"},
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{BackupStorageName: "bs4"},
				},
			},
			err: errTooManyPGStorages,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockClient := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(&tc.cluster).
				WithObjects(tc.dbClusterBackups...)
			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient.Build())
			assert.Equal(t, tc.err, validatePGReposForAPIDB(context.Background(), &tc.cluster, k.ListDatabaseClusterBackups))
		})
	}
}

func TestCheckStorageDuplicates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		cluster everestv1alpha1.DatabaseCluster
		err     error
	}{
		{
			name:    "ok: no schedules no backups",
			cluster: everestv1alpha1.DatabaseCluster{},
			err:     nil,
		},
		{
			name: "ok: no duplicated storages",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{BackupStorageName: "bs1"},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "error duplicated storage",
			cluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{BackupStorageName: "bs1"},
							{BackupStorageName: "bs2"},
							{BackupStorageName: "bs1"},
						},
					},
				},
			},
			err: errDuplicatedStoragePG,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.err, checkStorageDuplicates(&tc.cluster))
		})
	}
}

func TestCheckSchedulesChanges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                   string
		oldCluster, newCluster everestv1alpha1.DatabaseCluster
		err                    error
	}{
		{
			name:       "ok: no schedules no backups",
			oldCluster: everestv1alpha1.DatabaseCluster{},
			newCluster: everestv1alpha1.DatabaseCluster{},
			err:        nil,
		},
		{
			name: "ok: added storage",
			oldCluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{Name: "A", BackupStorageName: "bs1"},
						},
					},
				},
			},
			newCluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{Name: "A", BackupStorageName: "bs1"},
							{Name: "B", BackupStorageName: "bs2"},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "ok: deleted storage",
			oldCluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{Name: "A", BackupStorageName: "bs1"},
							{Name: "B", BackupStorageName: "bs2"},
						},
					},
				},
			},
			newCluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{Name: "A", BackupStorageName: "bs1"},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "ok: deleted storage and new added",
			oldCluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{Name: "A", BackupStorageName: "bs1"},
							{Name: "B", BackupStorageName: "bs2"},
						},
					},
				},
			},
			newCluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{Name: "A", BackupStorageName: "bs1"},
							{Name: "C", BackupStorageName: "bs2"},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "error: edited storage",
			oldCluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{Name: "A", BackupStorageName: "bs1"},
							{Name: "B", BackupStorageName: "bs2"},
						},
					},
				},
			},
			newCluster: everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Backup: everestv1alpha1.Backup{
						Schedules: []everestv1alpha1.BackupSchedule{
							{Name: "A", BackupStorageName: "bs1"},
							{Name: "B", BackupStorageName: "bs3"},
						},
					},
				},
			},
			err: errStorageChangePG,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.err, checkSchedulesChanges(&tc.oldCluster, &tc.newCluster))
		})
	}
}

func TestValidateShardingOnUpdate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc         string
		expected     error
		updated, old *everestv1alpha1.DatabaseCluster
	}{
		{
			desc:    "disabled",
			updated: &everestv1alpha1.DatabaseCluster{},
			old: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Sharding: &everestv1alpha1.Sharding{
						Enabled: false,
					},
				},
			},
			expected: nil,
		},
		{
			desc:    "try to disable - no sharding section",
			updated: &everestv1alpha1.DatabaseCluster{},
			old: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
					},
				},
			},
			expected: errDisableShardingNotSupported,
		},
		{
			desc: "try to disable - enabled false",
			updated: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Sharding: &everestv1alpha1.Sharding{
						Enabled: false,
					},
				},
			},
			old: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
					},
				},
			},
			expected: errDisableShardingNotSupported,
		},
		{
			desc: "try to enable - if there was no sharding section",
			updated: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
						Shards:  3,
						ConfigServer: everestv1alpha1.ConfigServer{
							Replicas: 2,
						},
					},
				},
			},
			old: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{},
			},
			expected: errShardingEnablingNotSupported,
		},
		{
			desc: "try to enable - if sharding 'enabled' was false",
			updated: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
						Shards:  3,
						ConfigServer: everestv1alpha1.ConfigServer{
							Replicas: 2,
						},
					},
				},
			},
			old: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Sharding: &everestv1alpha1.Sharding{
						Enabled: false,
					},
				},
			},
			expected: errShardingEnablingNotSupported,
		},
		{
			desc: "ok",
			updated: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:    everestv1alpha1.DatabaseEnginePSMDB,
						Version: "1.17.0",
					},
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
						Shards:  5,
						ConfigServer: everestv1alpha1.ConfigServer{
							Replicas: 3,
						},
					},
				},
			},
			old: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
						ConfigServer: everestv1alpha1.ConfigServer{
							Replicas: 3,
						},
						Shards: 5,
					},
				},
			},
			expected: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, validateShardingOnUpdate(tc.updated, tc.old))
		})
	}
}

func TestValidateSharding(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc     string
		expected error
		cluster  *everestv1alpha1.DatabaseCluster
	}{
		{
			desc: "old psmdb version",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:    everestv1alpha1.DatabaseEnginePSMDB,
						Version: "1.15.0",
					},
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
						Shards:  5,
						ConfigServer: everestv1alpha1.ConfigServer{
							Replicas: 3,
						},
					},
				},
			},
			expected: errShardingVersion,
		},
		{
			desc: "pxc - not supported",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
					},
				},
			},
			expected: errShardingIsNotSupported,
		},
		{
			desc: "pg - not supported",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePostgresql,
					},
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
					},
				},
			},
			expected: errShardingIsNotSupported,
		},
		{
			desc: "even configservers",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:    everestv1alpha1.DatabaseEnginePSMDB,
						Version: "1.17.0",
					},
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
						Shards:  1,
						ConfigServer: everestv1alpha1.ConfigServer{
							Replicas: 4,
						},
					},
				},
			},
			expected: errEvenServersNumber,
		},
		{
			desc: "insufficient configservers",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:     everestv1alpha1.DatabaseEnginePSMDB,
						Version:  "1.17.0",
						Replicas: 3,
					},
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
						Shards:  1,
						ConfigServer: everestv1alpha1.ConfigServer{
							Replicas: 1,
						},
					},
				},
			},
			expected: errInsufficientCfgSrvNumber,
		},
		{
			desc: "insufficient configservers 1 node",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:     everestv1alpha1.DatabaseEnginePSMDB,
						Version:  "1.17.0",
						Replicas: 1,
					},
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
						Shards:  1,
						ConfigServer: everestv1alpha1.ConfigServer{
							Replicas: 0,
						},
					},
				},
			},
			expected: errInsufficientCfgSrvNumber1Node,
		},
		{
			desc: "insufficient shards number",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:    everestv1alpha1.DatabaseEnginePSMDB,
						Version: "1.17.0",
					},
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
						Shards:  0,
						ConfigServer: everestv1alpha1.ConfigServer{
							Replicas: 3,
						},
					},
				},
			},
			expected: errInsufficientShardsNumber,
		},
		{
			desc: "ok",
			cluster: &everestv1alpha1.DatabaseCluster{
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type:    everestv1alpha1.DatabaseEnginePSMDB,
						Version: "1.17.0",
					},
					Sharding: &everestv1alpha1.Sharding{
						Enabled: true,
						Shards:  1,
						ConfigServer: everestv1alpha1.ConfigServer{
							Replicas: 3,
						},
					},
				},
			},
			expected: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, validateSharding(tc.cluster))
		})
	}
}

func TestIsDatabaseClusterUpdateAllowed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		currentDB *everestv1alpha1.DatabaseCluster
		expected  bool
	}{
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStateUnknown),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStateUnknown,
				},
			},
			expected: true,
		},
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStateCreating),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStateCreating,
				},
			},
			expected: true,
		},
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStateInit),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStateInit,
				},
			},
			expected: true,
		},
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStatePaused),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStatePaused,
				},
			},
			expected: true,
		},
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStateReady),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStateReady,
				},
			},
			expected: true,
		},
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStateError),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStateError,
				},
			},
			expected: true,
		},
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStateRestoring),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStateRestoring,
				},
			},
			expected: false,
		},
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStateDeleting),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStateDeleting,
				},
			},
			expected: false,
		},
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStateUpgrading),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStateUpgrading,
				},
			},
			expected: false,
		},
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStateNew),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStateNew,
				},
			},
			expected: true,
		},
		{
			name: fmt.Sprintf("db_state_%s", everestv1alpha1.AppStateResizingVolumes),
			currentDB: &everestv1alpha1.DatabaseCluster{
				Status: everestv1alpha1.DatabaseClusterStatus{
					Status: everestv1alpha1.AppStateResizingVolumes,
				},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isDatabaseClusterUpdateAllowed(tt.currentDB))
		})
	}
}

func TestValidatePodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name      string
		objs      []ctrlclient.Object
		dbCluster *everestv1alpha1.DatabaseCluster
		wantErr   error
	}

	testCases := []testCase{
		// no policy used
		{
			name: "no policy used",
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-no-policy",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
				},
			},
		},
		// absent policy used
		{
			name: "absent policy used",
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-absent-policy",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
					PodSchedulingPolicyName: "absent-policy",
				},
			},
			wantErr: k8sError.NewNotFound(schema.GroupResource{
				Group:    everestv1alpha1.GroupVersion.Group,
				Resource: "podschedulingpolicies",
			},
				"absent-policy",
			),
		},
		// engineType mismatches
		{
			name: "engineType mismatch PXC",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "everest-default-postgresql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-policy-mismatch",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
					PodSchedulingPolicyName: "everest-default-postgresql",
				},
			},
			wantErr: errDBClusterPSPEngineTypeMismatch("everest-default-postgresql", everestv1alpha1.DatabaseEnginePXC),
		},
		{
			name: "engineType mismatch PSMDB",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "everest-default-postgresql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
					},
				},
			},
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-policy-mismatch",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePSMDB,
					},
					PodSchedulingPolicyName: "everest-default-postgresql",
				},
			},
			wantErr: errDBClusterPSPEngineTypeMismatch("everest-default-postgresql", everestv1alpha1.DatabaseEnginePSMDB),
		},
		{
			name: "engineType mismatch PostgreSQL",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "everest-default-mysql",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
			},
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-policy-mismatch",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePostgresql,
					},
					PodSchedulingPolicyName: "everest-default-mysql",
				},
			},
			wantErr: errDBClusterPSPEngineTypeMismatch("everest-default-mysql", everestv1alpha1.DatabaseEnginePostgresql),
		},
		// affinityConfig is absent
		{
			name: "affinityConfig is absent",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
					},
				},
			},
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-affinityConfig-empty",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
					PodSchedulingPolicyName: "test-pxc-policy",
				},
			},
			wantErr: errDBClusterInvalidPSPAffinityConfig("test-pxc-policy"),
		},
		// affinityConfig.PXC is absent
		{
			name: "affinityConfig.PXC is absent",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType:     everestv1alpha1.DatabaseEnginePXC,
						AffinityConfig: &everestv1alpha1.AffinityConfig{},
					},
				},
			},
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-affinityConfig-pxc-empty",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
					PodSchedulingPolicyName: "test-pxc-policy",
				},
			},
			wantErr: errDBClusterInvalidPSPAffinityPXCEmpty("test-pxc-policy"),
		},
		// affinityConfig.PXC DB components are absent
		{
			name: "affinityConfig.PXC DB components are absent",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pxc-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePXC,
						AffinityConfig: &everestv1alpha1.AffinityConfig{
							PXC: &everestv1alpha1.PXCAffinityConfig{},
						},
					},
				},
			},
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-affinityConfig-pxc-components",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePXC,
					},
					PodSchedulingPolicyName: "test-pxc-policy",
				},
			},
			wantErr: errDBClusterInvalidPSPAffinityPXCComponentsEmpty("test-pxc-policy"),
		},
		// affinityConfig.PSMDB is absent
		{
			name: "affinityConfig.PSMDB is absent",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-psmdb-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType:     everestv1alpha1.DatabaseEnginePSMDB,
						AffinityConfig: &everestv1alpha1.AffinityConfig{},
					},
				},
			},
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-affinityConfig-psmdb-empty",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePSMDB,
					},
					PodSchedulingPolicyName: "test-psmdb-policy",
				},
			},
			wantErr: errDBClusterInvalidPSPAffinityPSMDBEmpty("test-psmdb-policy"),
		},
		// affinityConfig.PSMDB DB components are absent
		{
			name: "affinityConfig.PSMDB DB components are absent",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-psmdb-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePSMDB,
						AffinityConfig: &everestv1alpha1.AffinityConfig{
							PSMDB: &everestv1alpha1.PSMDBAffinityConfig{},
						},
					},
				},
			},
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-affinityConfig-psmdb-components",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePSMDB,
					},
					PodSchedulingPolicyName: "test-psmdb-policy",
				},
			},
			wantErr: errDBClusterInvalidPSPAffinityPSMDBComponentsEmpty("test-psmdb-policy"),
		},
		// affinityConfig.PostgreSQL is absent
		{
			name: "affinityConfig.PostgreSQL is absent",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pg-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType:     everestv1alpha1.DatabaseEnginePostgresql,
						AffinityConfig: &everestv1alpha1.AffinityConfig{},
					},
				},
			},
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-affinityConfig-pg-empty",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePostgresql,
					},
					PodSchedulingPolicyName: "test-pg-policy",
				},
			},
			wantErr: errDBClusterInvalidPSPAffinityPostgresqlEmpty("test-pg-policy"),
		},
		// affinityConfig.PostgreSQL DB components are absent
		{
			name: "affinityConfig.PostgreSQL DB components are absent",
			objs: []ctrlclient.Object{
				&everestv1alpha1.PodSchedulingPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pg-policy",
					},
					Spec: everestv1alpha1.PodSchedulingPolicySpec{
						EngineType: everestv1alpha1.DatabaseEnginePostgresql,
						AffinityConfig: &everestv1alpha1.AffinityConfig{
							PostgreSQL: &everestv1alpha1.PostgreSQLAffinityConfig{},
						},
					},
				},
			},
			dbCluster: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-affinityConfig-pg-components",
					Namespace: "test-ns",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine: everestv1alpha1.Engine{
						Type: everestv1alpha1.DatabaseEnginePostgresql,
					},
					PodSchedulingPolicyName: "test-pg-policy",
				},
			},
			wantErr: errDBClusterInvalidPSPAffinityPostgresqlComponentsEmpty("test-pg-policy"),
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

			valHandler := &validateHandler{
				log:           zap.NewNop().Sugar(),
				kubeConnector: k,
			}
			valHandler.SetNext(k8sHandler)

			err := valHandler.validatePodSchedulingPolicy(context.Background(), tc.dbCluster)
			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
