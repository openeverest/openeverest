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
	"strings"
	"testing"

	version "github.com/Percona-Lab/percona-version-service/versionpb"
	goversion "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	versionservice "github.com/openeverest/openeverest/v2/pkg/version_service"
)

func TestUpgrade_setVersionInfo(t *testing.T) {
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

			_, err = u.setVersionInfo(context.Background(), everestVersion)
			if err != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("error = %v, wantErrIs %v", err, tt.wantErrIs)
				return
			}

			if err != nil {
				return
			}

			assert.Equal(t, tt.wantUpgradeTo, u.upgradeToVersion)
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

func TestUpgrade_checkLegacyMonitoring(t *testing.T) {
	t.Parallel()

	monitoringConfigGVK := monitoringv1alpha1.GroupVersion.WithKind("MonitoringConfig")
	monitoringConfig := func(namespace, name string) *unstructured.Unstructured {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(monitoringConfigGVK)
		obj.SetNamespace(namespace)
		obj.SetName(name)
		return obj
	}
	instance := func(namespace, name, monitoringType string) *corev1alpha1.Instance {
		in := &corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
			Spec: corev1alpha1.InstanceSpec{Components: map[string]corev1alpha1.ComponentSpec{
				"engine": {Type: "mongod"},
			}},
		}
		if monitoringType != "" {
			in.Spec.Components["monitoring"] = corev1alpha1.ComponentSpec{Type: monitoringType}
		}
		return in
	}
	rbacPolicy := func(policy string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: "everest", Name: common.EverestRBACConfigMapName},
			Data:       map[string]string{"policy.csv": policy},
		}
	}

	tests := []struct {
		name string
		// monitoringConfigServed reports whether the cluster still serves the MonitoringConfig kind.
		monitoringConfigServed bool
		objects                []ctrlclient.Object
		wantInUse              []string
		wantCommands           []string
	}{
		{
			name: "MonitoringConfig no longer served and no monitoring component",
			objects: []ctrlclient.Object{
				instance("db", "mongo", ""),
				rbacPolicy("g, admin, role:admin\np, role:ops, monitoring-destinations, *, */*/*\n"),
			},
		},
		{
			name:                   "MonitoringConfig served but none left",
			monitoringConfigServed: true,
			objects:                []ctrlclient.Object{instance("db", "mongo", "")},
		},
		{
			name: "monitoring component of another type",
			objects: []ctrlclient.Object{
				instance("db", "mariadb", "mysqld-exporter"),
				instance("db", "cassandra", "prometheus"),
			},
		},
		{
			name:                   "MonitoringConfig left over",
			monitoringConfigServed: true,
			objects: []ctrlclient.Object{
				monitoringConfig("everest-monitoring", "pmm"),
				instance("db", "mongo", ""),
			},
			wantInUse:    []string{"MonitoringConfig everest-monitoring/pmm"},
			wantCommands: []string{"kubectl delete monitoringconfig -n everest-monitoring pmm"},
		},
		{
			name: "Instance with a pmm monitoring component",
			objects: []ctrlclient.Object{
				instance("db", "mongo", "pmm"),
				instance("db", "pg", ""),
			},
			wantInUse: []string{`Instance db/mongo (component "monitoring" of type "pmm")`},
			wantCommands: []string{
				`kubectl patch instance -n db mongo --type=json -p '[{"op":"remove","path":"/spec/components/monitoring"}]'`,
			},
		},
		{
			name: "RBAC policy granting monitoring-configs",
			objects: []ctrlclient.Object{
				rbacPolicy("g, admin, role:admin\n" +
					"p, role:ops, monitoring-configs, *, */*/*\n" +
					"p, role:ops, monitoring-destinations, read, */*/*\n" +
					"# p, role:old, monitoring-configs, read, */*/*\n" +
					"  p,role:dev,monitoring-configs,read,dev/*/*\n"),
			},
			wantInUse: []string{
				`ConfigMap everest/everest-rbac policy line "p, role:ops, monitoring-configs, *, */*/*" (use "monitoring-destinations" instead)`,
				`ConfigMap everest/everest-rbac policy line "p,role:dev,monitoring-configs,read,dev/*/*" (use "monitoring-destinations" instead)`,
			},
			wantCommands: []string{
				`kubectl edit configmap -n everest everest-rbac  # in policy.csv, replace "monitoring-configs" with "monitoring-destinations"`,
			},
		},
		{
			name:                   "MonitoringConfigs, Instances with a pmm monitoring component and RBAC policy",
			monitoringConfigServed: true,
			objects: []ctrlclient.Object{
				monitoringConfig("everest-monitoring", "pmm"),
				monitoringConfig("db", "pmm-db"),
				instance("db", "mongo", "pmm"),
				instance("other", "mysql", "pmm"),
				instance("db", "pg", ""),
				rbacPolicy("p, role:ops, monitoring-configs, read, */*/*\n"),
			},
			wantInUse: []string{
				"MonitoringConfig everest-monitoring/pmm",
				"MonitoringConfig db/pmm-db",
				`Instance db/mongo (component "monitoring" of type "pmm")`,
				`Instance other/mysql (component "monitoring" of type "pmm")`,
				`ConfigMap everest/everest-rbac policy line "p, role:ops, monitoring-configs, read, */*/*" (use "monitoring-destinations" instead)`,
			},
			wantCommands: []string{
				`kubectl patch instance -n db mongo --type=json -p '[{"op":"remove","path":"/spec/components/monitoring"}]'`,
				`kubectl patch instance -n other mysql --type=json -p '[{"op":"remove","path":"/spec/components/monitoring"}]'`,
				"kubectl delete monitoringconfig -n everest-monitoring pmm",
				"kubectl delete monitoringconfig -n db pmm-db",
				`kubectl edit configmap -n everest everest-rbac  # in policy.csv, replace "monitoring-configs" with "monitoring-destinations"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme := kubernetes.CreateScheme()
			scheme.AddKnownTypeWithName(monitoringConfigGVK, &unstructured.Unstructured{})
			scheme.AddKnownTypeWithName(monitoringConfigGVK.GroupVersion().WithKind("MonitoringConfigList"), &unstructured.UnstructuredList{})
			mapper := meta.NewDefaultRESTMapper(nil)
			if tt.monitoringConfigServed {
				mapper.Add(monitoringConfigGVK, meta.RESTScopeNamespace)
			}
			client := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithRESTMapper(mapper).
				WithObjects(tt.objects...).
				Build()

			u := &Upgrade{
				l:             zap.NewNop().Sugar(),
				kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "everest").WithKubernetesClient(client),
			}

			err := u.checkLegacyMonitoring(context.Background())
			if len(tt.wantInUse) == 0 {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, ErrLegacyMonitoringInUse)
			var listed, commands []string
			for line := range strings.SplitSeq(err.Error(), "\n") {
				if item, ok := strings.CutPrefix(line, "  - "); ok {
					listed = append(listed, item)
				}
				if command, ok := strings.CutPrefix(line, "       "); ok {
					commands = append(commands, command)
				}
			}
			assert.ElementsMatch(t, tt.wantInUse, listed)
			assert.ElementsMatch(t, tt.wantCommands, commands)
			assert.Contains(t, err.Error(), "install a monitoring extension")
			assert.Contains(t, err.Error(), "spec.monitoring.destinations[]")
		})
	}
}

func TestCheckUpgradableFrom(t *testing.T) {
	t.Parallel()

	const (
		target    = "2.0.0-alpha.3"
		installed = "2.0.0-alpha.2"
		hint      = `To move to 2.0.0-alpha.3, uninstall OpenEverest with "everestctl uninstall", ` +
			"which deletes every instance and its data, then install 2.0.0-alpha.3"
	)

	tests := []struct {
		name        string
		annotations map[string]string
		wantErr     string
	}{
		{
			name:        "chart without annotations",
			annotations: nil,
			wantErr: "OpenEverest 2.0.0-alpha.3 requires a fresh install; upgrading from 2.0.0-alpha.2 is not supported: " +
				"its chart predates in-place upgrades (it has no openeverest.io/upgradable-from annotation), " +
				"and every release before them required a fresh install.\n" + hint,
		},
		{
			name: "annotation absent",
			annotations: map[string]string{
				contract.CoreVersionKey:    target,
				contract.VersionAnnotation: target,
			},
			wantErr: "OpenEverest 2.0.0-alpha.3 requires a fresh install; upgrading from 2.0.0-alpha.2 is not supported: " +
				"its chart predates in-place upgrades (it has no openeverest.io/upgradable-from annotation), " +
				"and every release before them required a fresh install.\n" + hint,
		},
		{
			name:        "annotation empty: fresh install only",
			annotations: map[string]string{contract.UpgradableFromAnnotation: ""},
			wantErr:     "OpenEverest 2.0.0-alpha.3 requires a fresh install; upgrading from 2.0.0-alpha.2 is not supported.\n" + hint,
		},
		{
			name:        "annotation lists the installed release",
			annotations: map[string]string{contract.UpgradableFromAnnotation: "2.0.0-alpha.2"},
		},
		{
			name:        "annotation lists the installed release among others",
			annotations: map[string]string{contract.UpgradableFromAnnotation: "2.0.0-alpha.1, 2.0.0-alpha.2,2.0.0-alpha.2.1"},
		},
		{
			name:        "annotation does not list the installed release",
			annotations: map[string]string{contract.UpgradableFromAnnotation: "2.0.0-alpha.1,2.0.0-alpha.2.1,2.0.0-alpha.20"},
			wantErr: "OpenEverest 2.0.0-alpha.3 requires a fresh install; upgrading from 2.0.0-alpha.2 is not supported " +
				"(it upgrades in place from 2.0.0-alpha.1, 2.0.0-alpha.2.1, 2.0.0-alpha.20 only).\n" + hint,
		},
		{
			name:        "annotation with only separators",
			annotations: map[string]string{contract.UpgradableFromAnnotation: " , "},
			wantErr:     "OpenEverest 2.0.0-alpha.3 requires a fresh install; upgrading from 2.0.0-alpha.2 is not supported.\n" + hint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := checkUpgradableFrom(target, installed, tt.annotations)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, ErrFreshInstallRequired)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

type fakeUpgradeChart struct {
	annotations     map[string]string
	deployedVersion string
	deployedErr     error
}

func (f fakeUpgradeChart) ChartMetadata() *chart.Metadata {
	return &chart.Metadata{Annotations: f.annotations}
}

func (f fakeUpgradeChart) DeployedChartVersion() (string, error) {
	return f.deployedVersion, f.deployedErr
}

func TestUpgrade_installedCoreVersion(t *testing.T) {
	t.Parallel()

	coreCRD := func(labels map[string]string) *apiextv1.CustomResourceDefinition {
		return &apiextv1.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: coreVersionCRD, Labels: labels}}
	}
	tests := []struct {
		name    string
		objects []ctrlclient.Object
		getErr  error
		release fakeUpgradeChart
		want    string
		isErr   func(error) bool
	}{
		{
			name:    "label on the core CRD",
			objects: []ctrlclient.Object{coreCRD(map[string]string{contract.CoreVersionKey: "2.0.0-alpha.2"})},
			release: fakeUpgradeChart{deployedVersion: "2.0.0-alpha.1"},
			want:    "2.0.0-alpha.2",
		},
		{
			name:    "label missing: Helm release chart version",
			objects: []ctrlclient.Object{coreCRD(map[string]string{"everest.percona.com/version": "2.0.0-alpha.1"})},
			release: fakeUpgradeChart{deployedVersion: "2.0.0-alpha.1"},
			want:    "2.0.0-alpha.1",
		},
		{
			name:    "label empty: Helm release chart version",
			objects: []ctrlclient.Object{coreCRD(map[string]string{contract.CoreVersionKey: ""})},
			release: fakeUpgradeChart{deployedVersion: "2.0.0-alpha.1"},
			want:    "2.0.0-alpha.1",
		},
		{
			name:    "core CRD missing: Helm release chart version",
			release: fakeUpgradeChart{deployedVersion: "1.17.1"},
			want:    "1.17.1",
		},
		{
			name:    "label missing and no deployed Helm release",
			objects: []ctrlclient.Object{coreCRD(nil)},
			release: fakeUpgradeChart{deployedErr: driver.NewErrNoDeployedReleases("everest-system")},
			isErr:   func(err error) bool { return errors.Is(err, driver.ErrNoDeployedReleases) },
		},
		{
			name:    "core CRD cannot be read",
			getErr:  k8serrors.NewForbidden(apiextv1.Resource("customresourcedefinitions"), coreVersionCRD, errors.New("no access")),
			release: fakeUpgradeChart{deployedVersion: "2.0.0-alpha.1"},
			isErr:   k8serrors.IsForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(tt.objects...)
			if tt.getErr != nil {
				builder = builder.WithInterceptorFuncs(interceptor.Funcs{
					Get: func(context.Context, ctrlclient.WithWatch, ctrlclient.ObjectKey, ctrlclient.Object, ...ctrlclient.GetOption) error {
						return tt.getErr
					},
				})
			}
			u := &Upgrade{
				l:             zap.NewNop().Sugar(),
				kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "everest-system").WithKubernetesClient(builder.Build()),
			}

			got, err := u.installedCoreVersion(context.Background(), tt.release)
			if tt.isErr != nil {
				require.Error(t, err)
				assert.True(t, tt.isErr(err), "unexpected error: %v", err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpgrade_checkInPlaceUpgrade(t *testing.T) {
	t.Parallel()

	// Each case gets its own CRD: the fake client writes the
	// resourceVersion of the objects it is built with, and the cases run
	// in parallel.
	coreCRD := func() *apiextv1.CustomResourceDefinition {
		return &apiextv1.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{
			Name:   coreVersionCRD,
			Labels: map[string]string{contract.CoreVersionKey: "2.0.0-alpha.2"},
		}}
	}

	tests := []struct {
		name    string
		objects []ctrlclient.Object
		chart   fakeUpgradeChart
		wantErr string
	}{
		{
			name:    "target upgrades from the release on the core CRDs",
			objects: []ctrlclient.Object{coreCRD()},
			chart: fakeUpgradeChart{
				annotations:     map[string]string{contract.UpgradableFromAnnotation: "2.0.0-alpha.2"},
				deployedVersion: "2.0.0-alpha.1",
			},
		},
		{
			name:    "target upgrades from the deployed chart version only",
			objects: []ctrlclient.Object{coreCRD()},
			chart: fakeUpgradeChart{
				annotations:     map[string]string{contract.UpgradableFromAnnotation: "2.0.0-alpha.1"},
				deployedVersion: "2.0.0-alpha.1",
			},
			wantErr: "upgrading from 2.0.0-alpha.2 is not supported (it upgrades in place from 2.0.0-alpha.1 only)",
		},
		{
			name: "CRDs without the label: deployed chart version",
			chart: fakeUpgradeChart{
				annotations:     map[string]string{contract.UpgradableFromAnnotation: "2.0.0-alpha.2"},
				deployedVersion: "2.0.0-alpha.1",
			},
			wantErr: "upgrading from 2.0.0-alpha.1 is not supported (it upgrades in place from 2.0.0-alpha.2 only)",
		},
		{
			name:    "target chart predates the annotation",
			objects: []ctrlclient.Object{coreCRD()},
			chart:   fakeUpgradeChart{deployedVersion: "2.0.0-alpha.2"},
			wantErr: "upgrading from 2.0.0-alpha.2 is not supported: its chart predates in-place upgrades",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(tt.objects...).
				Build()
			u := &Upgrade{
				l:                zap.NewNop().Sugar(),
				kubeConnector:    kubernetes.NewEmpty(zap.NewNop().Sugar(), "everest-system").WithKubernetesClient(client),
				upgradeToVersion: "2.0.0-alpha.3",
			}

			err := u.checkInPlaceUpgrade(context.Background(), tt.chart)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, ErrFreshInstallRequired)
			assert.Contains(t, err.Error(), "OpenEverest 2.0.0-alpha.3 requires a fresh install; "+tt.wantErr)
		})
	}
}
