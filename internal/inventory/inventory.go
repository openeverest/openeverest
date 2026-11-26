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

// Package inventory reads what OpenEverest manages in one Kubernetes cluster.
//
// A Snapshot holds every object, names included, and feeds the /metrics
// endpoint. A Summary holds only what telemetry may send, and the summaries of
// several clusters add up into one report. The package depends on a
// client.Reader only, so the same code runs in the controller, the API server
// or a future multicluster agent.
package inventory

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// Snapshot is everything OpenEverest manages in one cluster at one point in
// time. Readers must treat the objects as read-only: they may be shared with
// an informer cache.
type Snapshot struct {
	InstalledExtensions    []extensionsv1alpha1.InstalledExtension
	Plugins                []extensionsv1alpha1.Plugin
	Providers              []corev1alpha1.Provider
	Instances              []corev1alpha1.Instance
	Backups                []backupv1alpha1.Backup
	Restores               []backupv1alpha1.Restore
	BackupClasses          []backupv1alpha1.BackupClass
	BackupStorageClasses   []backupv1alpha1.BackupStorageClass
	BackupStorages         []backupv1alpha1.BackupStorage
	MonitoringClasses      []monitoringv1alpha1.MonitoringClass
	MonitoringDestinations []monitoringv1alpha1.MonitoringDestination
	MonitoringBindings     []monitoringv1alpha1.MonitoringBinding
}

// Kinds returns one object of every kind Collect lists. A caller can check
// that its cache holds them all before collecting.
func Kinds() []client.Object {
	return []client.Object{
		&extensionsv1alpha1.InstalledExtension{},
		&extensionsv1alpha1.Plugin{},
		&corev1alpha1.Provider{},
		&corev1alpha1.Instance{},
		&backupv1alpha1.Backup{},
		&backupv1alpha1.Restore{},
		&backupv1alpha1.BackupClass{},
		&backupv1alpha1.BackupStorageClass{},
		&backupv1alpha1.BackupStorage{},
		&monitoringv1alpha1.MonitoringClass{},
		&monitoringv1alpha1.MonitoringDestination{},
		&monitoringv1alpha1.MonitoringBinding{},
	}
}

// Collect lists every kind the inventory covers, across all namespaces.
// With a cache-backed reader it makes no API server requests and does not copy
// the cached objects.
func Collect(ctx context.Context, r client.Reader) (*Snapshot, error) {
	var (
		installedExtensions    extensionsv1alpha1.InstalledExtensionList
		plugins                extensionsv1alpha1.PluginList
		providers              corev1alpha1.ProviderList
		instances              corev1alpha1.InstanceList
		backups                backupv1alpha1.BackupList
		restores               backupv1alpha1.RestoreList
		backupClasses          backupv1alpha1.BackupClassList
		backupStorageClasses   backupv1alpha1.BackupStorageClassList
		backupStorages         backupv1alpha1.BackupStorageList
		monitoringClasses      monitoringv1alpha1.MonitoringClassList
		monitoringDestinations monitoringv1alpha1.MonitoringDestinationList
		monitoringBindings     monitoringv1alpha1.MonitoringBindingList
	)
	lists := []struct {
		kind string
		list client.ObjectList
	}{
		{"InstalledExtension", &installedExtensions},
		{"Plugin", &plugins},
		{"Provider", &providers},
		{"Instance", &instances},
		{"Backup", &backups},
		{"Restore", &restores},
		{"BackupClass", &backupClasses},
		{"BackupStorageClass", &backupStorageClasses},
		{"BackupStorage", &backupStorages},
		{"MonitoringClass", &monitoringClasses},
		{"MonitoringDestination", &monitoringDestinations},
		{"MonitoringBinding", &monitoringBindings},
	}
	for _, l := range lists {
		if err := r.List(ctx, l.list, client.UnsafeDisableDeepCopy); err != nil {
			return nil, fmt.Errorf("list %s: %w", l.kind, err)
		}
	}
	return &Snapshot{
		InstalledExtensions:    installedExtensions.Items,
		Plugins:                plugins.Items,
		Providers:              providers.Items,
		Instances:              instances.Items,
		Backups:                backups.Items,
		Restores:               restores.Items,
		BackupClasses:          backupClasses.Items,
		BackupStorageClasses:   backupStorageClasses.Items,
		BackupStorages:         backupStorages.Items,
		MonitoringClasses:      monitoringClasses.Items,
		MonitoringDestinations: monitoringDestinations.Items,
		MonitoringBindings:     monitoringBindings.Items,
	}, nil
}
