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

package metrics

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	utilruntime.Must(corev1alpha1.AddToScheme(s))
	utilruntime.Must(extensionsv1alpha1.AddToScheme(s))
	utilruntime.Must(backupv1alpha1.AddToScheme(s))
	utilruntime.Must(monitoringv1alpha1.AddToScheme(s))
	return s
}

func condition(t, status string) metav1.Condition {
	return metav1.Condition{Type: t, Status: metav1.ConditionStatus(status), Reason: "Test"}
}

// gather registers the collector on a pedantic registry, which fails on
// series that don't match their declared descriptor, and returns every sample
// as "name{label="value",...}" with labels sorted.
func gather(t *testing.T, c prometheus.Collector) map[string]float64 {
	t.Helper()
	reg := prometheus.NewPedanticRegistry()
	require.NoError(t, reg.Register(c))
	families, err := reg.Gather()
	require.NoError(t, err)
	out := map[string]float64{}
	for _, f := range families {
		for _, m := range f.GetMetric() {
			labels := make([]string, 0, len(m.GetLabel()))
			for _, l := range m.GetLabel() {
				labels = append(labels, fmt.Sprintf("%s=%q", l.GetName(), l.GetValue()))
			}
			sort.Strings(labels)
			out[f.GetName()+"{"+strings.Join(labels, ",")+"}"] = m.GetGauge().GetValue()
		}
	}
	return out
}

// series formats a sample the way gather does, from label name, value pairs.
func series(name string, kv ...string) string {
	labels := make([]string, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		labels = append(labels, fmt.Sprintf("%s=%q", kv[i], kv[i+1]))
	}
	sort.Strings(labels)
	return name + "{" + strings.Join(labels, ",") + "}"
}

func leader() bool { return true }

func TestCollector(t *testing.T) {
	t.Parallel()

	firstBackup := metav1.NewTime(time.Unix(1_700_000_000, 0))
	lastBackup := metav1.NewTime(time.Unix(1_700_086_400, 0))
	ns := "team-a"
	instanceRef := &common.ObjectRef{Name: "db1"}

	objects := []client.Object{
		&extensionsv1alpha1.InstalledExtension{
			ObjectMeta: metav1.ObjectMeta{Name: "percona-server-mongodb"},
			Spec:       extensionsv1alpha1.InstalledExtensionSpec{Type: extensionsv1alpha1.InstalledExtensionTypeProvider, Version: "1.2.0"},
			Status:     extensionsv1alpha1.InstalledExtensionStatus{Phase: extensionsv1alpha1.InstalledExtensionPhaseInstalled},
		},
		&corev1alpha1.Provider{
			ObjectMeta: metav1.ObjectMeta{Name: "percona-server-mongodb"},
			Status: corev1alpha1.ProviderStatus{
				ContractVersion: "2.0.0",
				Conditions:      []metav1.Condition{condition("Ready", "True")},
			},
		},
		&monitoringv1alpha1.MonitoringClass{
			ObjectMeta: metav1.ObjectMeta{Name: "prometheus"},
			Spec:       monitoringv1alpha1.MonitoringClassSpec{ControllerName: "openeverest.io/prometheus"},
			Status: monitoringv1alpha1.MonitoringClassStatus{
				Conditions: []metav1.Condition{condition(monitoringv1alpha1.MonitoringClassConditionAccepted, "True")},
			},
		},
		&backupv1alpha1.BackupClass{
			ObjectMeta: metav1.ObjectMeta{Name: "pbm"},
			Spec: backupv1alpha1.BackupClassSpec{
				ProviderManaged: &backupv1alpha1.ProviderManagedSpec{ProviderRef: common.ObjectRef{Name: "percona-server-mongodb"}},
			},
		},
		&backupv1alpha1.BackupStorageClass{ObjectMeta: metav1.ObjectMeta{Name: "s3"}},
		&backupv1alpha1.BackupStorage{
			ObjectMeta: metav1.ObjectMeta{Name: "store", Namespace: ns},
			Spec:       backupv1alpha1.BackupStorageSpec{BackupStorageClassRef: common.ObjectRef{Name: "s3"}},
		},
		&monitoringv1alpha1.MonitoringDestination{
			ObjectMeta: metav1.ObjectMeta{Name: "dest", Namespace: ns},
			Spec:       monitoringv1alpha1.MonitoringDestinationSpec{ClassRef: common.ObjectRef{Name: "prometheus"}},
			Status: monitoringv1alpha1.MonitoringDestinationStatus{
				Conditions: []metav1.Condition{condition(monitoringv1alpha1.MonitoringDestinationConditionReady, "False")},
			},
		},
		&corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: ns},
			Spec: corev1alpha1.InstanceSpec{
				ProviderRef: common.ObjectRef{Name: "percona-server-mongodb"},
				Version:     "8.0",
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Replicas: ptr.To[int32](3),
						Resources: &corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
							Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
						},
						Storage: &corev1alpha1.Storage{Size: resource.MustParse("10Gi")},
					},
				},
			},
			Status: corev1alpha1.InstanceStatus{
				Phase:   corev1alpha1.InstancePhaseProvisioned,
				Version: "8.0.12",
				Conditions: []metav1.Condition{
					condition(corev1alpha1.ConditionReady, "True"),
					condition(corev1alpha1.ConditionStalled, "False"),
					// Not a summary condition: not exported.
					condition(corev1alpha1.ConditionSynced, "True"),
				},
				Components: []corev1alpha1.ComponentStatus{
					{Name: "engine", DesiredReplicas: ptr.To[int32](3), Replicas: ptr.To[int32](3), ReadyReplicas: ptr.To[int32](2)},
				},
				Backup: &corev1alpha1.InstanceBackupStatus{Storages: []corev1alpha1.InstanceBackupStorageStatus{
					{Name: "store", PITR: &corev1alpha1.InstanceBackupStoragePITRStatus{State: corev1alpha1.PITRStateAvailable}},
				}},
			},
		},
		&backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "b1", Namespace: ns},
			Spec: backupv1alpha1.BackupSpec{
				Origin:     backupv1alpha1.BackupOrigin{Type: backupv1alpha1.BackupOriginTypeInstance, InstanceRef: instanceRef},
				StorageRef: common.ObjectRef{Name: "store"},
			},
			Status: backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded, CompletedAt: &firstBackup},
		},
		&backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "b2", Namespace: ns},
			Spec: backupv1alpha1.BackupSpec{
				Origin:     backupv1alpha1.BackupOrigin{Type: backupv1alpha1.BackupOriginTypeInstance, InstanceRef: instanceRef},
				StorageRef: common.ObjectRef{Name: "store"},
			},
			Status: backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded, CompletedAt: &lastBackup},
		},
		&backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "b3", Namespace: ns},
			Spec: backupv1alpha1.BackupSpec{
				Origin:     backupv1alpha1.BackupOrigin{Type: backupv1alpha1.BackupOriginTypeInstance, InstanceRef: instanceRef},
				StorageRef: common.ObjectRef{Name: "store"},
			},
			Status: backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateFailed},
		},
		&backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "imported", Namespace: ns},
			Spec: backupv1alpha1.BackupSpec{
				Origin:     backupv1alpha1.BackupOrigin{Type: backupv1alpha1.BackupOriginTypeExternal},
				StorageRef: common.ObjectRef{Name: "store"},
			},
			Status: backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded, CompletedAt: &lastBackup},
		},
		&backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: ns},
			Spec:       backupv1alpha1.RestoreSpec{InstanceRef: common.ObjectRef{Name: "db1"}},
		},
	}
	reader := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(objects...).Build()

	got := gather(t, NewCollector(reader, leader, logr.Discard()))

	// db1 labels a series of team-a/db1 with the labels given as name, value pairs.
	db1 := func(name string, kv ...string) string {
		return series(name, append([]string{labelNamespace, "team-a", labelInstance, "db1"}, kv...)...)
	}
	want := map[string]float64{
		series("openeverest_extension_info", "name", "percona-server-mongodb", "type", "provider", "version", "1.2.0"):               1,
		series("openeverest_extension_phase", "name", "percona-server-mongodb", "phase", "Installed", "type", "provider"):            1,
		series("openeverest_extension_phase", "name", "percona-server-mongodb", "phase", "Failed", "type", "provider"):               0,
		series("openeverest_provider_info", "contract_version", "2.0.0", labelProvider, "percona-server-mongodb"):                    1,
		series("openeverest_provider_condition", "condition", "Ready", labelProvider, "percona-server-mongodb", "status", "true"):    1,
		series("openeverest_provider_condition", "condition", "Ready", labelProvider, "percona-server-mongodb", "status", "false"):   0,
		series("openeverest_class_info", "controller", "openeverest.io/prometheus", "kind", "MonitoringClass", "name", "prometheus"): 1,
		series("openeverest_class_accepted", "kind", "MonitoringClass", "name", "prometheus", "status", "true"):                      1,
		series("openeverest_class_info", "controller", "percona-server-mongodb", "kind", "BackupClass", "name", "pbm"):               1,
		series("openeverest_class_info", "controller", "", "kind", "BackupStorageClass", "name", "s3"):                               1,
		series("openeverest_backup_storage_info", "class", "s3", "name", "store", labelNamespace, "team-a"):                          1,
		series("openeverest_monitoring_destination_info", "class", "prometheus", "name", "dest", labelNamespace, "team-a"):           1,
		series("openeverest_monitoring_destination_ready", "name", "dest", labelNamespace, "team-a", "status", "false"):              1,
		db1("openeverest_instance_info", labelProvider, "percona-server-mongodb", "version", "8.0.12"):                               1,
		db1("openeverest_instance_phase", "phase", "Provisioned"):                                                                    1,
		db1("openeverest_instance_phase", "phase", "Pending"):                                                                        0,
		db1("openeverest_instance_condition", "condition", "Ready", "status", "true"):                                                1,
		db1("openeverest_instance_condition", "condition", "Stalled", "status", "false"):                                             1,
		db1("openeverest_instance_condition", "condition", "Stalled", "status", "true"):                                              0,
		db1("openeverest_instance_component_replicas", "component", "engine", "kind", "desired"):                                     3,
		db1("openeverest_instance_component_replicas", "component", "engine", "kind", "ready"):                                       2,
		db1("openeverest_instance_resource_requests", "component", "engine", "resource", "cpu"):                                      1.5,
		db1("openeverest_instance_resource_requests", "component", "engine", "resource", "memory"):                                   3 * 1024 * 1024 * 1024,
		db1("openeverest_instance_resource_requests", "component", "engine", "resource", "storage"):                                  30 * 1024 * 1024 * 1024,
		db1("openeverest_instance_resource_limits", "component", "engine", "resource", "cpu"):                                        3,
		db1("openeverest_instance_pitr_available", "storage", "store"):                                                               1,
		db1("openeverest_instance_last_successful_backup_timestamp_seconds", "storage", "store"):                                     float64(lastBackup.Unix()),
		db1("openeverest_backups", "state", "Succeeded", "storage", "store"):                                                         2,
		db1("openeverest_backups", "state", "Failed", "storage", "store"):                                                            1,
		series("openeverest_backups", labelInstance, "", labelNamespace, "team-a", "state", "Succeeded", "storage", "store"):         1,
		db1("openeverest_restores", "state", "Unknown"):                                                                              1,
	}
	for s, value := range want {
		if assert.Contains(t, got, s) {
			assert.InDelta(t, value, got[s], 0.001, s)
		}
	}
	assert.NotContains(t, got, db1("openeverest_instance_resource_limits", "component", "engine", "resource", "memory"),
		"a resource the spec doesn't set is not exported")
	for s := range got {
		assert.NotContains(t, s, `condition="Synced"`, "only summary conditions are exported")
	}
}

func TestCollectorFollowsDeletions(t *testing.T) {
	t.Parallel()

	inst := &corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "team-a"}}
	reader := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(inst).Build()
	c := NewCollector(reader, leader, logr.Discard())
	info := series("openeverest_instance_info", labelNamespace, "team-a", labelInstance, "db1", labelProvider, "", "version", "")

	require.Contains(t, gather(t, c), info)
	require.NoError(t, reader.Delete(context.Background(), inst))
	assert.NotContains(t, gather(t, c), info)
}

func TestCollectorOnlyLeaderEmitsInventory(t *testing.T) {
	t.Parallel()

	inst := &corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "team-a"}}
	reader := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(inst).Build()

	got := gather(t, NewCollector(reader, func() bool { return false }, logr.Discard()))

	require.Len(t, got, 1)
	for series := range got {
		assert.True(t, strings.HasPrefix(series, "openeverest_build_info{"), series)
	}
}

func TestCollectorSurvivesRepeatedConditions(t *testing.T) {
	t.Parallel()

	inst := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "team-a"},
		Status: corev1alpha1.InstanceStatus{Conditions: []metav1.Condition{
			condition(corev1alpha1.ConditionReady, "True"),
			condition(corev1alpha1.ConditionReady, "False"),
		}},
	}
	reader := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(inst).Build()

	got := gather(t, NewCollector(reader, leader, logr.Discard()))

	assert.InDelta(t, 1, got[series("openeverest_instance_condition",
		labelNamespace, "team-a", labelInstance, "db1", "condition", "Ready", "status", "true")], 0)
}
