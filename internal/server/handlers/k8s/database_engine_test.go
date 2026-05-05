package k8s

import (
	"context"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	versionservice "github.com/percona/everest/pkg/version_service"
)

func TestGetUpgradePreflightChecks(t *testing.T) {
	t.Parallel()
	crVersion := "1.1.1"
	operatorVersion := "1.2.1"
	targetVersion := "1.3.0"

	// Setup mock
	mockDBVersions := []string{"0.5.0", "0.6.0", "0.7.0", "0.8.0"}
	versionService := versionservice.MockInterface{}
	versionService.On(
		"GetSupportedEngineVersions",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(mockDBVersions, nil)

	ctx := context.Background()
	t.Run("upgrade unavailable", func(t *testing.T) {
		t.Parallel()
		dbs := []everestv1alpha1.DatabaseCluster{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseClusterSpec{
				Engine: everestv1alpha1.Engine{
					Version: "0.4.0",
				},
			},
		}}
		engine := everestv1alpha1.DatabaseEngine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-engine",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseEngineSpec{
				Type: everestv1alpha1.DatabaseEnginePXC,
			},
			Status: everestv1alpha1.DatabaseEngineStatus{
				OperatorVersion: operatorVersion,
			},
		}

		_, err := getUpgradePreflightChecksResult(ctx, dbs, upgradePreflightCheckArgs{
			targetVersion:  targetVersion,
			versionService: &versionService,
			engine:         &engine,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, errDBEngineUpgradeUnavailable)
	})

	t.Run("pending DB Engine upgrade", func(t *testing.T) {
		t.Parallel()
		dbs := []everestv1alpha1.DatabaseCluster{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseClusterSpec{
				Engine: everestv1alpha1.Engine{
					Version: "0.4.0",
				},
			},
		}}
		engine := everestv1alpha1.DatabaseEngine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-engine",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseEngineSpec{
				Type: everestv1alpha1.DatabaseEnginePXC,
			},
			Status: everestv1alpha1.DatabaseEngineStatus{
				OperatorVersion: operatorVersion,
				PendingOperatorUpgrades: []everestv1alpha1.OperatorUpgrade{
					{TargetVersion: targetVersion},
				},
			},
		}

		result, err := getUpgradePreflightChecksResult(ctx, dbs, upgradePreflightCheckArgs{
			targetVersion:  targetVersion,
			versionService: &versionService,
			engine:         &engine,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, operatorVersion, result.currentVersion)
		assert.Len(t, result.databases, 1)
		dbResult := (result.databases)[0]
		assert.Equal(t, "test-db", pointer.Get(dbResult.Name))
		assert.Equal(t, api.UpgradeEngine, pointer.Get(dbResult.PendingTask))
		assert.Equal(t, "Upgrade DB version to 0.5.0 or higher", pointer.Get(dbResult.Message))
	})

	t.Run("pending minor version upgrade", func(t *testing.T) {
		t.Parallel()

		operatorVersion := "2.5.0"
		targetVersion := "2.6.0"

		// Setup mock
		mockDBVersions := []string{"13.16", "14.12", "15.8", "16.4"}
		versionService := versionservice.MockInterface{}
		versionService.On(
			"GetSupportedEngineVersions",
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(mockDBVersions, nil)

		dbs := []everestv1alpha1.DatabaseCluster{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseClusterSpec{
				Engine: everestv1alpha1.Engine{
					Version: "16.3",
				},
			},
			Status: everestv1alpha1.DatabaseClusterStatus{
				Status: everestv1alpha1.AppStateReady,
			},
		}}
		engine := everestv1alpha1.DatabaseEngine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-engine",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseEngineSpec{
				Type: everestv1alpha1.DatabaseEnginePXC,
			},
			Status: everestv1alpha1.DatabaseEngineStatus{
				OperatorVersion: operatorVersion,
				PendingOperatorUpgrades: []everestv1alpha1.OperatorUpgrade{
					{TargetVersion: targetVersion},
				},
			},
		}

		result, err := getUpgradePreflightChecksResult(ctx, dbs, upgradePreflightCheckArgs{
			targetVersion:  targetVersion,
			versionService: &versionService,
			engine:         &engine,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, operatorVersion, result.currentVersion)
		assert.Len(t, result.databases, 1)
		dbResult := (result.databases)[0]
		assert.Equal(t, "test-db", pointer.Get(dbResult.Name))
		assert.Equal(t, api.UpgradeEngine, pointer.Get(dbResult.PendingTask))
		assert.Equal(t, "Upgrade DB version to 16.4 or higher", pointer.Get(dbResult.Message))
	})

	t.Run("pending major version upgrade", func(t *testing.T) {
		t.Parallel()

		operatorVersion := "2.5.0"
		targetVersion := "2.6.0"

		// Setup mock
		mockDBVersions := []string{"13.16", "14.12", "15.8", "16.4"}
		versionService := versionservice.MockInterface{}
		versionService.On(
			"GetSupportedEngineVersions",
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(mockDBVersions, nil)

		dbs := []everestv1alpha1.DatabaseCluster{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseClusterSpec{
				Engine: everestv1alpha1.Engine{
					Version: "12.19",
				},
			},
			Status: everestv1alpha1.DatabaseClusterStatus{
				Status: everestv1alpha1.AppStateReady,
			},
		}}
		engine := everestv1alpha1.DatabaseEngine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-engine",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseEngineSpec{
				Type: everestv1alpha1.DatabaseEnginePXC,
			},
			Status: everestv1alpha1.DatabaseEngineStatus{
				OperatorVersion: operatorVersion,
				PendingOperatorUpgrades: []everestv1alpha1.OperatorUpgrade{
					{TargetVersion: targetVersion},
				},
			},
		}

		result, err := getUpgradePreflightChecksResult(ctx, dbs, upgradePreflightCheckArgs{
			targetVersion:  targetVersion,
			versionService: &versionService,
			engine:         &engine,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, operatorVersion, result.currentVersion)
		assert.Len(t, result.databases, 1)
		dbResult := (result.databases)[0]
		assert.Equal(t, "test-db", pointer.Get(dbResult.Name))
		assert.Equal(t, api.UpgradeEngine, pointer.Get(dbResult.PendingTask))
		assert.Equal(t, "Upgrade DB version to 13.16 or higher", pointer.Get(dbResult.Message))
	})

	t.Run("pending CRVersion update", func(t *testing.T) {
		t.Parallel()
		dbs := []everestv1alpha1.DatabaseCluster{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseClusterSpec{
				Engine: everestv1alpha1.Engine{
					Version: "0.5.0",
				},
			},
			Status: everestv1alpha1.DatabaseClusterStatus{
				CRVersion:            crVersion,
				RecommendedCRVersion: pointer.To(operatorVersion),
			},
		}}

		engine := everestv1alpha1.DatabaseEngine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-engine",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseEngineSpec{
				Type: everestv1alpha1.DatabaseEnginePXC,
			},
			Status: everestv1alpha1.DatabaseEngineStatus{
				OperatorVersion: operatorVersion,
				PendingOperatorUpgrades: []everestv1alpha1.OperatorUpgrade{
					{TargetVersion: targetVersion},
				},
			},
		}

		result, err := getUpgradePreflightChecksResult(ctx, dbs, upgradePreflightCheckArgs{
			targetVersion:  targetVersion,
			versionService: &versionService,
			engine:         &engine,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, operatorVersion, result.currentVersion)
		assert.Len(t, result.databases, 1)
		dbResult := (result.databases)[0]
		assert.Equal(t, "test-db", pointer.Get(dbResult.Name))
		assert.Equal(t, api.Restart, pointer.Get(dbResult.PendingTask))
		assert.Equal(t, "Update CRVersion to "+operatorVersion, pointer.Get(dbResult.Message))
	})

	t.Run("not ready", func(t *testing.T) {
		t.Parallel()
		dbs := []everestv1alpha1.DatabaseCluster{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseClusterSpec{
				Engine: everestv1alpha1.Engine{
					Version: "0.5.0",
				},
			},
			Status: everestv1alpha1.DatabaseClusterStatus{
				CRVersion: operatorVersion,
				Status:    everestv1alpha1.AppStateError,
			},
		}}

		engine := everestv1alpha1.DatabaseEngine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-engine",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseEngineSpec{
				Type: everestv1alpha1.DatabaseEnginePXC,
			},
			Status: everestv1alpha1.DatabaseEngineStatus{
				OperatorVersion: operatorVersion,
				PendingOperatorUpgrades: []everestv1alpha1.OperatorUpgrade{
					{TargetVersion: targetVersion},
				},
			},
		}

		result, err := getUpgradePreflightChecksResult(ctx, dbs, upgradePreflightCheckArgs{
			targetVersion:  targetVersion,
			versionService: &versionService,
			engine:         &engine,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, operatorVersion, result.currentVersion)
		assert.Len(t, result.databases, 1)
		dbResult := (result.databases)[0]
		assert.Equal(t, "test-db", pointer.Get(dbResult.Name))
		assert.Equal(t, dbResult.Message, pointer.ToString("Database is not ready"))
		assert.Equal(t, api.NotReady, pointer.Get(dbResult.PendingTask))
	})

	t.Run("ready for upgrade", func(t *testing.T) {
		t.Parallel()
		dbs := []everestv1alpha1.DatabaseCluster{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseClusterSpec{
				Engine: everestv1alpha1.Engine{
					Version: "0.5.0",
				},
			},
			Status: everestv1alpha1.DatabaseClusterStatus{
				CRVersion: operatorVersion,
				Status:    everestv1alpha1.AppStateReady,
			},
		}}

		engine := everestv1alpha1.DatabaseEngine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-engine",
				Namespace: "test-namespace",
			},
			Spec: everestv1alpha1.DatabaseEngineSpec{
				Type: everestv1alpha1.DatabaseEnginePXC,
			},
			Status: everestv1alpha1.DatabaseEngineStatus{
				OperatorVersion: operatorVersion,
				PendingOperatorUpgrades: []everestv1alpha1.OperatorUpgrade{
					{TargetVersion: targetVersion},
				},
			},
		}

		result, err := getUpgradePreflightChecksResult(ctx, dbs, upgradePreflightCheckArgs{
			targetVersion:  targetVersion,
			versionService: &versionService,
			engine:         &engine,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, operatorVersion, result.currentVersion)
		assert.Len(t, result.databases, 1)
		dbResult := (result.databases)[0]
		assert.Equal(t, "test-db", pointer.Get(dbResult.Name))
		assert.Equal(t, api.Ready, pointer.Get(dbResult.PendingTask))
	})
}
