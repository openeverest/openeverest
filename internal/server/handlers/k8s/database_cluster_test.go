package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
)

func TestLatestRestorableDate(t *testing.T) {
	t.Parallel()
	type tCase struct {
		uploadInterval   int
		latestBackupTime time.Time
		now              time.Time
		expected         *time.Time
		name             string
	}

	now := time.Date(2024, 3, 12, 12, 0, 0, 0, time.UTC)
	cases := []tCase{
		{
			name:             "backup 5 min ago, upload interval 10 min",
			uploadInterval:   600,
			latestBackupTime: now.Add(-300 * time.Second),
			now:              now,
			expected:         nil,
		},
		{
			name:             "backup 15 min ago, upload interval 10 min",
			uploadInterval:   600,
			latestBackupTime: now.Add(-900 * time.Second),
			now:              now,
			expected:         pointer.ToTime(now.Add(-600 * time.Second)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, latestRestorableDate(tc.now, tc.latestBackupTime, tc.uploadInterval))
		})
	}
}

func TestGetDefaultUploadInterval(t *testing.T) {
	t.Parallel()
	type tCase struct {
		name     string
		engine   everestv1alpha1.Engine
		interval *int
		expected int
	}
	cases := []tCase{
		{
			name:     "old pxc, no interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePXC, Version: "1.13.0"},
			interval: nil,
			expected: pxcDefaultUploadInterval,
		},
		{
			name:     "old pxc, interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePXC, Version: "1.13.0"},
			interval: pointer.ToInt(1000),
			expected: 1000,
		},
		{
			name:     "new pxc, no interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePXC, Version: "1.14.0"},
			interval: nil,
			expected: 0,
		},
		{
			name:     "new pxc, interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePXC, Version: "1.14.0"},
			interval: pointer.ToInt(1000),
			expected: 0,
		},
		{
			name:     "newer pxc",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePXC, Version: "1.15.1"},
			interval: nil,
			expected: 0,
		},
		{
			name:     "old psmdb, no interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePSMDB, Version: "1.15.0"},
			interval: nil,
			expected: psmdbDefaultUploadInterval,
		},
		{
			name:     "old psmdb, interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePSMDB, Version: "1.15.0"},
			interval: pointer.ToInt(1000),
			expected: 1000,
		},
		{
			name:     "new psmdb, no interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePSMDB, Version: "1.16.0"},
			interval: nil,
			expected: psmdbDefaultUploadInterval,
		},
		{
			name:     "new psmdb, interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePSMDB, Version: "1.16.0"},
			interval: pointer.ToInt(1000),
			expected: 1000,
		},
		{
			name:     "newer psmdb",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePSMDB, Version: "1.16.1"},
			interval: nil,
			expected: psmdbDefaultUploadInterval,
		},

		{
			name:     "old pg, no interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePostgresql, Version: "2.3.1"},
			interval: nil,
			expected: pgDefaultUploadInterval,
		},
		{
			name:     "old pg, interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePostgresql, Version: "2.3.1"},
			interval: pointer.ToInt(1000),
			expected: 1000,
		},
		{
			name:     "new pg, no interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePostgresql, Version: "2.4.0"},
			interval: nil,
			expected: pgDefaultUploadInterval,
		},
		{
			name:     "new pg, interval is set",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePostgresql, Version: "2.4.0"},
			interval: pointer.ToInt(1000),
			expected: 1000,
		},
		{
			name:     "newer pg",
			engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePostgresql, Version: "2.4.1"},
			interval: nil,
			expected: pgDefaultUploadInterval,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, getDefaultUploadInterval(tc.engine, tc.interval))
		})
	}
}

func TestConnectionURL(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		db   everestv1alpha1.DatabaseCluster
		user,
		password,
		expected string
	}

	cases := []testCase{
		{
			name: "non-sharded psmdb 1 node",
			db: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "psmdb-try",
					Namespace: "ns-1",
				},
				Spec:   everestv1alpha1.DatabaseClusterSpec{Engine: everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePSMDB}},
				Status: everestv1alpha1.DatabaseClusterStatus{Hostname: "mongodb-56u-rs0-0.mongodb-56u-rs0.ns-1.svc.cluster.local:27017", Port: 27017},
			},
			user:     "databaseAdmin",
			password: "azoE4FwvDRVycH83CO",
			expected: "mongodb://databaseAdmin:azoE4FwvDRVycH83CO@mongodb-56u-rs0-0.mongodb-56u-rs0.ns-1.svc.cluster.local:27017",
		},
		{
			name: "non-sharded psmdb, 3 node",
			db: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "psmdb-try",
					Namespace: "ns-2",
				},
				Spec:   everestv1alpha1.DatabaseClusterSpec{Engine: everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePSMDB}},
				Status: everestv1alpha1.DatabaseClusterStatus{Hostname: "mongodb-56u-rs0-0.mongodb-56u-rs0.ns-2.svc.cluster.local:27017,mongodb-56u-rs0-1.mongodb-56u-rs0.ns-2.svc.cluster.local:27017,mongodb-56u-rs0-2.mongodb-56u-rs0.ns-2.svc.cluster.local:27017", Port: 27017},
			},
			user:     "databaseAdmin",
			password: "azoE4FwvDRVycH83CO",
			expected: "mongodb://databaseAdmin:azoE4FwvDRVycH83CO@mongodb-56u-rs0-0.mongodb-56u-rs0.ns-2.svc.cluster.local:27017,mongodb-56u-rs0-1.mongodb-56u-rs0.ns-2.svc.cluster.local:27017,mongodb-56u-rs0-2.mongodb-56u-rs0.ns-2.svc.cluster.local:27017",
		},
		{
			name: "sharded psmdb, 3 node, external access enabled",
			db: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "psmdb-try",
					Namespace: "ns-4",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePSMDB},
					Sharding: &everestv1alpha1.Sharding{Enabled: true},
					Proxy:    everestv1alpha1.Proxy{Expose: everestv1alpha1.Expose{Type: "external"}},
				},
				Status: everestv1alpha1.DatabaseClusterStatus{Hostname: "34.34.163.11", Port: 27017},
			},
			user:     "databaseAdmin",
			password: "azoE4FwvDRVycH83CO",
			expected: "mongodb://databaseAdmin:azoE4FwvDRVycH83CO@34.34.163.11:27017",
		},
		{
			name: "sharded psmdb, external access disabled",
			db: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "psmdb-try",
					Namespace: "ns-5",
				},
				Spec: everestv1alpha1.DatabaseClusterSpec{
					Engine:   everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePSMDB},
					Sharding: &everestv1alpha1.Sharding{Enabled: true},
				},
				Status: everestv1alpha1.DatabaseClusterStatus{Hostname: "mongodb-56u-mongos.ns-5.svc.cluster.local", Port: 27017},
			},
			user:     "databaseAdmin",
			password: "azoE4FwvDRVycH83CO",
			expected: "mongodb://databaseAdmin:azoE4FwvDRVycH83CO@mongodb-56u-mongos.ns-5.svc.cluster.local:27017",
		},
		{
			name: "pg",
			db: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "psmdb-try",
					Namespace: "ns-6",
				},
				Spec:   everestv1alpha1.DatabaseClusterSpec{Engine: everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePostgresql}},
				Status: everestv1alpha1.DatabaseClusterStatus{Hostname: "postgresql-a5d-pgbouncer.ns-6.svc", Port: 5432},
			},
			user:     "postgres",
			password: "55aBDedMF;So|C?^3x|h.dDC",
			expected: "postgres://postgres:55aBDedMF%3BSo%7CC%3F%5E3x%7Ch.dDC@postgresql-a5d-pgbouncer.ns-6.svc:5432",
		},
		{
			name: "pxc external domain",
			db: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pxc-try",
					Namespace: "ns-7",
				},
				Spec:   everestv1alpha1.DatabaseClusterSpec{Engine: everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePXC}},
				Status: everestv1alpha1.DatabaseClusterStatus{Hostname: "mysql-29o-haproxy.everest.io", Port: 3306},
			},
			user:     "root",
			password: ",0#3PdCIc=9CS(do2",
			expected: "jdbc:mysql://root:%2C0%233PdCIc%3D9CS%28do2@mysql-29o-haproxy.everest.io:3306",
		},
		{
			name: "pxc local service",
			db: everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pxc-try",
					Namespace: "ns-8",
				},
				Spec:   everestv1alpha1.DatabaseClusterSpec{Engine: everestv1alpha1.Engine{Type: everestv1alpha1.DatabaseEnginePXC}},
				Status: everestv1alpha1.DatabaseClusterStatus{Hostname: "mysql-56o-haproxy.ns-8.svc", Port: 3306},
			},
			user:     "root",
			password: ",0#123AIc=9CS(do2",
			expected: "jdbc:mysql://root:%2C0%23123AIc%3D9CS%28do2@mysql-56o-haproxy.ns-8.svc:3306",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockClient := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(&tc.db)
			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient.Build())
			h := &k8sHandler{kubeConnector: k}
			url := h.connectionURL(context.Background(), &tc.db, tc.user, tc.password)
			require.Equal(t, tc.expected, *url)
		})
	}
}

func TestCreateDatabaseClusterSecret(t *testing.T) {
	t.Parallel()

	const (
		testNamespace = "test-namespace"
		testDBName    = "test-db"
	)

	testCases := []struct {
		name       string
		secret     *corev1.Secret
		verifyFunc func(t *testing.T, secret *corev1.Secret)
	}{
		{
			name: "create secret for PXC engine",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pxc-secret",
				},
				Data: map[string][]byte{
					"root": []byte("password"),
				},
			},
			verifyFunc: func(t *testing.T, secret *corev1.Secret) {
				require.Equal(t, testNamespace, secret.Namespace)
				require.Equal(t, "pxc-secret", secret.Name)
				require.Equal(t, testDBName, secret.Labels[common.DatabaseClusterNameLabel])
				require.Equal(t, []byte("password"), secret.Data["root"])
			},
		},
		{
			name: "create secret for PSMDB engine",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "psmdb-secret",
				},
				Data: map[string][]byte{
					"MONGODB_DATABASE_ADMIN_USER":     []byte("admin"),
					"MONGODB_DATABASE_ADMIN_PASSWORD": []byte("password"),
				},
			},
			verifyFunc: func(t *testing.T, secret *corev1.Secret) {
				require.Equal(t, testNamespace, secret.Namespace)
				require.Equal(t, "psmdb-secret", secret.Name)
				require.Equal(t, testDBName, secret.Labels[common.DatabaseClusterNameLabel])
				require.Equal(t, []byte("admin"), secret.Data["MONGODB_DATABASE_ADMIN_USER"])
				require.Equal(t, []byte("password"), secret.Data["MONGODB_DATABASE_ADMIN_PASSWORD"])
			},
		},
		{
			name: "create secret for PostgreSQL engine",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pg-secret",
				},
				Data: map[string][]byte{
					"password": []byte("password"),
				},
			},
			verifyFunc: func(t *testing.T, secret *corev1.Secret) {
				require.Equal(t, testNamespace, secret.Namespace)
				require.Equal(t, "pg-secret", secret.Name)
				require.Equal(t, testDBName, secret.Labels[common.DatabaseClusterNameLabel])
				require.Equal(t, []byte("password"), secret.Data["password"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create mock client
			mockClient := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				Build()

			// Create k8s handler with mock client
			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient)
			k8sH := New(zap.NewNop().Sugar(), k, "")

			// Call the function under test
			createdSecret, err := k8sH.CreateDatabaseClusterSecret(
				context.Background(),
				testNamespace,
				testDBName,
				tc.secret,
			)

			// Verify the result
			require.NoError(t, err)
			require.NotNil(t, createdSecret)
			tc.verifyFunc(t, createdSecret)
		})
	}
}
