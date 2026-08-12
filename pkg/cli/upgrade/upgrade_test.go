// everest
// Copyright (C) 2023 Percona LLC
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

package upgrade

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	version "github.com/Percona-Lab/percona-version-service/versionpb"
	"github.com/blang/semver/v4"
	goversion "github.com/hashicorp/go-version"
	olmversion "github.com/operator-framework/api/pkg/lib/version"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	versionservice "github.com/openeverest/openeverest/v2/pkg/version_service"
)

func TestUpgrade_canUpgrade(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		everestVersion string
		versionMeta    *version.MetadataResponse
		wantErrIs      error
		wantUpgradeTo  string
	}{
		{
			name:           "upgrade by only one minor version",
			everestVersion: "0.6.0",
			versionMeta: &version.MetadataResponse{
				Versions: []*version.MetadataVersion{
					{Version: "0.5.0"},
					{Version: "0.6.0"},
					{Version: "0.7.0"},
					{Version: "0.8.0"},
				},
			},
			wantUpgradeTo: "0.7.0",
		},
		{
			name:           "upgrade by one version to the latest",
			everestVersion: "0.6.0",
			versionMeta: &version.MetadataResponse{
				Versions: []*version.MetadataVersion{
					{Version: "0.5.0"},
					{Version: "0.6.0"},
					{Version: "0.7.0"},
				},
			},
			wantUpgradeTo: "0.7.0",
		},
		{
			name:           "no update is available",
			everestVersion: "0.7.0",
			versionMeta: &version.MetadataResponse{
				Versions: []*version.MetadataVersion{
					{Version: "0.5.0"},
					{Version: "0.6.0"},
					{Version: "0.7.0"},
				},
			},
			wantErrIs: ErrNoUpdateAvailable,
		},
		{
			name:           "select latest patch version",
			everestVersion: "0.6.0",
			versionMeta: &version.MetadataResponse{
				Versions: []*version.MetadataVersion{
					{Version: "0.5.0"},
					{Version: "0.6.0"},
					{Version: "0.7.0"},
					{Version: "0.7.2"},
					{Version: "0.7.1"},
					{Version: "0.8.0"},
				},
			},
			wantUpgradeTo: "0.7.2",
		},
		{
			name:           "ignore invalid version",
			everestVersion: "0.6.0",
			versionMeta: &version.MetadataResponse{
				Versions: []*version.MetadataVersion{
					{Version: "0.5.0"},
					{Version: "0.6.0"},
					{Version: "invalid version"},
					{Version: "0.7.0"},
				},
			},
			wantUpgradeTo: "0.7.0",
		},
		{
			name:           "descending version order shall work",
			everestVersion: "0.6.0",
			versionMeta: &version.MetadataResponse{
				Versions: []*version.MetadataVersion{
					{Version: "0.8.0"},
					{Version: "0.7.0"},
					{Version: "0.6.0"},
					{Version: "0.5.0"},
				},
			},
			wantUpgradeTo: "0.7.0",
		},
		{
			name:           "upgrade to the next patch version if minor upgrade is not available",
			everestVersion: "0.8.0",
			versionMeta: &version.MetadataResponse{
				Versions: []*version.MetadataVersion{
					{Version: "0.8.1"},
					{Version: "0.8.0"},
					{Version: "0.7.0"},
					{Version: "0.6.0"},
					{Version: "0.5.0"},
				},
			},
			wantUpgradeTo: "0.8.1",
		},
		{
			name:           "upgrade to the next minor version even if a patch version is available",
			everestVersion: "0.8.0",
			versionMeta: &version.MetadataResponse{
				Versions: []*version.MetadataVersion{
					{Version: "0.9.2"},
					{Version: "0.9.0"},
					{Version: "0.8.1"},
					{Version: "0.8.0"},
					{Version: "0.7.0"},
					{Version: "0.6.0"},
					{Version: "0.5.0"},
				},
			},
			wantUpgradeTo: "0.9.2",
		},
		{
			name:           "upgrade to the next minor version even if a patch version is available - mixed order",
			everestVersion: "0.8.0",
			versionMeta: &version.MetadataResponse{
				Versions: []*version.MetadataVersion{
					{Version: "0.9.0"},
					{Version: "0.8.1"},
					{Version: "0.8.0"},
					{Version: "0.8.2"},
					{Version: "0.9.2"},
					{Version: "0.9.1"},
					{Version: "0.7.0"},
					{Version: "0.6.0"},
					{Version: "0.5.0"},
				},
			},
			wantUpgradeTo: "0.9.2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				b, err := json.Marshal(tt.versionMeta)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.Header().Add("content-type", "application/json")
				_, err = w.Write(b)
				assert.NoError(t, err)
			}))
			defer ts.Close()

			mockClient := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme())
			k := kubernetes.NewEmpty(zap.NewNop().Sugar(), "test-ns").WithKubernetesClient(mockClient.Build())

			u := &Upgrade{
				l: zap.NewNop().Sugar(),
				config: &Config{
					VersionMetadataURL: ts.URL,
				},
				kubeConnector:  k,
				versionService: versionservice.New(ts.URL),
			}
			everestVersion, err := goversion.NewVersion(tt.everestVersion)
			require.NoError(t, err)

			upgradeTo, err := u.canUpgrade(context.Background(), everestVersion)
			if err != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("error = %v, wantErrIs %v", err, tt.wantErrIs)
				return
			}

			if err != nil {
				return
			}

			assert.Equal(t, tt.wantUpgradeTo, upgradeTo.String())
		})
	}
}

func TestUpgrade_ValidateVersionToUpgrade(t *testing.T) {
	t.Parallel()

	tests := []struct {
		currentEverestVersion string
		targetEverestVersion  string
		wantErrIs             error
	}{
		{
			currentEverestVersion: "0.6.0",
			targetEverestVersion:  "0.5.0",
			wantErrIs:             ErrDowngradeNotAllowed,
		},
		{
			currentEverestVersion: "0.6.1",
			targetEverestVersion:  "0.6.0",
			wantErrIs:             ErrDowngradeNotAllowed,
		},
		{
			currentEverestVersion: "0.6.0",
			targetEverestVersion:  "0.6.0",
			wantErrIs:             ErrNoUpdateAvailable,
		},
		{
			currentEverestVersion: "0.6.0",
			targetEverestVersion:  "0.8.0",
			wantErrIs:             ErrCannotUpgradeByMoreThanOneMinorVersion,
		},
		{
			currentEverestVersion: "0.6.0",
			targetEverestVersion:  "0.8.1",
			wantErrIs:             ErrCannotUpgradeByMoreThanOneMinorVersion,
		},
		{
			currentEverestVersion: "0.6.0",
			targetEverestVersion:  "0.7.0",
			wantErrIs:             nil,
		},
		{
			currentEverestVersion: "0.6.0",
			targetEverestVersion:  "0.6.1",
			wantErrIs:             nil,
		},
		{
			currentEverestVersion: "0.6.0",
			targetEverestVersion:  "0.7.1",
			wantErrIs:             nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.currentEverestVersion+" to "+tt.targetEverestVersion, func(t *testing.T) {
			t.Parallel()
			currentVer := goversion.Must(goversion.NewVersion(tt.currentEverestVersion))
			targetVer := goversion.Must(goversion.NewVersion(tt.targetEverestVersion))
			err := validateVersionToUpgrade(currentVer, targetVer)
			if tt.wantErrIs != nil {
				assert.ErrorIs(t, err, tt.wantErrIs)
			} else if err != nil {
				t.Errorf("error = %v", err)
			}
		})
	}
}

func TestUpgrade_checkOperatorRequirements(t *testing.T) {
	t.Parallel()

	// GetDBNamespaces filters out the system namespace, so the DB namespace must differ from it.
	const dbNamespace = "db-ns"

	pxcConstraint, err := goversion.NewConstraint(">= 1.10.0")
	require.NoError(t, err)
	pgConstraint, err := goversion.NewConstraint(">= 2.4.0")
	require.NoError(t, err)
	psmdbConstraint, err := goversion.NewConstraint(">= 1.15.0")
	require.NoError(t, err)

	supVer := &common.SupportedVersion{
		PXCOperator:   pxcConstraint,
		PGOperator:    pgConstraint,
		PSMBDOperator: psmdbConstraint,
	}

	tests := []struct {
		name          string
		sub           *olmv1alpha1.Subscription
		csv           *olmv1alpha1.ClusterServiceVersion
		expectedError string
	}{
		{
			name: "PostgreSQL operator fails requirement",
			sub: &olmv1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Name:      common.PostgreSQLOperatorName,
					Namespace: dbNamespace,
				},
				Status: olmv1alpha1.SubscriptionStatus{
					InstalledCSV: "pg-operator.v2.2.0",
				},
			},
			csv: &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pg-operator.v2.2.0",
					Namespace: dbNamespace,
				},
				Spec: olmv1alpha1.ClusterServiceVersionSpec{
					Version: olmversion.OperatorVersion{
						Version: semver.MustParse("2.2.0"),
					},
				},
			},
			expectedError: `percona-postgresql-operator version "2.2.0" does not meet minimum requirements of ">= 2.4.0"`,
		},
		{
			name: "MongoDB operator fails requirement",
			sub: &olmv1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Name:      common.MongoDBOperatorName,
					Namespace: dbNamespace,
				},
				Status: olmv1alpha1.SubscriptionStatus{
					InstalledCSV: "psmdb-operator.v1.12.0",
				},
			},
			csv: &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "psmdb-operator.v1.12.0",
					Namespace: dbNamespace,
				},
				Spec: olmv1alpha1.ClusterServiceVersionSpec{
					Version: olmversion.OperatorVersion{
						Version: semver.MustParse("1.12.0"),
					},
				},
			},
			expectedError: `percona-server-mongodb-operator version "1.12.0" does not meet minimum requirements of ">= 1.15.0"`,
		},
		{
			name: "PXC operator fails requirement",
			sub: &olmv1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Name:      common.MySQLOperatorName,
					Namespace: dbNamespace,
				},
				Status: olmv1alpha1.SubscriptionStatus{
					InstalledCSV: "pxc-operator.v1.9.0",
				},
			},
			csv: &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pxc-operator.v1.9.0",
					Namespace: dbNamespace,
				},
				Spec: olmv1alpha1.ClusterServiceVersionSpec{
					Version: olmversion.OperatorVersion{
						Version: semver.MustParse("1.9.0"),
					},
				},
			},
			expectedError: `percona-xtradb-cluster-operator version "1.9.0" does not meet minimum requirements of ">= 1.10.0"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   dbNamespace,
					Labels: map[string]string{common.KubernetesManagedByLabel: common.Everest},
				},
			}

			client := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(ns, tt.sub, tt.csv).
				Build()

			k := kubernetes.NewEmpty(zap.NewNop().Sugar(), "everest").WithKubernetesClient(client)
			u := &Upgrade{
				l:             zap.NewNop().Sugar(),
				kubeConnector: k,
			}

			err := u.checkOperatorRequirements(context.Background(), supVer)
			require.Error(t, err)
			assert.Equal(t, tt.expectedError, err.Error())
		})
	}
}
