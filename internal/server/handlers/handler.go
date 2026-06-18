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

// Package handlers contains the interface and types for the Everest API handlers.
package handlers

//go:generate go tool mockery --name=Handler --case=snake --inpackage

import (
	"context"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// Handler provides an abstraction for the core business logic of the Everest API.
// Each implementation of a handler is responsible for handling a specific set of operations (e.g, request validation, RBAC, KubeAPI, etc.).
// Handlers may be chained together using the SetNext() method to form a chain of responsibility.
// Each Handler implementation is individually responsible for calling the next handler in the chain.
//
//nolint:interfacebloat
type Handler interface {
	// SetNext sets the next handler to call in the chain.
	SetNext(h Handler)

	NamespacesHandler
	BackupStorageHandler
	ProviderHandler
	InstanceHandler
	InstancePresetHandler
	ClusterHandler
	BackupClassHandler
	BackupHandler
	RestoreHandler
	InstanceBackupHandler
	MonitoringConfigHandler
	InstanceRestoreHandler

	GetKubernetesClusterResources(ctx context.Context) (*api.KubernetesClusterResources, error)
	GetKubernetesClusterInfo(ctx context.Context) (*api.KubernetesClusterInfo, error)
	GetUserPermissions(ctx context.Context) (*api.UserPermissions, error)
	GetSettings(ctx context.Context) (*api.Settings, error)
}

// NamespacesHandler provides methods for handling operations on namespaces.
type NamespacesHandler interface {
	ListNamespaces(ctx context.Context, cluster string) ([]string, error)
}

// BackupStorageHandler provides methods for handling operations on backup storages.
type BackupStorageHandler interface {
	CreateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error)
	UpdateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error)
	PatchBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error)
	ListBackupStorages(ctx context.Context, cluster, namespace string) (*backupv1alpha1.BackupStorageList, error)
	GetBackupStorage(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.BackupStorage, error)
	DeleteBackupStorage(ctx context.Context, cluster, namespace, name string) error
}

// ProviderHandler provides methods for handling operations on providers.
type ProviderHandler interface {
	ListProviders(ctx context.Context, cluster string) (*corev1alpha1.ProviderList, error)
	GetProvider(ctx context.Context, cluster, name string) (*corev1alpha1.Provider, error)
}

// InstanceHandler provides methods for handling operations on instances.
type InstanceHandler interface {
	ListInstances(ctx context.Context, cluster, namespace string) (*corev1alpha1.InstanceList, error)
	GetInstance(ctx context.Context, cluster, namespace, name string) (*corev1alpha1.Instance, error)
	CreateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error)
	UpdateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error)
	DeleteInstance(ctx context.Context, cluster, namespace, name string, params *api.DeleteInstanceParams) error
	GetInstanceConnection(ctx context.Context, cluster, namespace, name string) (*api.InstanceConnectionDetails, error)
}

// InstancePresetHandler provides methods for handling operations on instance presets.
type InstancePresetHandler interface {
	ListInstancePresets(ctx context.Context, cluster string, provider string) (*corev1alpha1.InstancePresetList, error)
	GetInstancePreset(ctx context.Context, cluster, name string) (*corev1alpha1.InstancePreset, error)
	ResolveInstancePreset(ctx context.Context, cluster, name, namespace string) (*corev1alpha1.InstancePreset, error)
}

// ClusterHandler provides methods for handling operations on clusters.
type ClusterHandler interface {
	ListClusters(ctx context.Context) (*api.ClusterList, error)
	GetCluster(ctx context.Context, name string) (*api.Cluster, error)
}

// BackupClassHandler provides methods for handling operations on backup classes.
type BackupClassHandler interface {
	ListBackupClasses(ctx context.Context, cluster string) (*backupv1alpha1.BackupClassList, error)
	GetBackupClass(ctx context.Context, cluster, name string) (*backupv1alpha1.BackupClass, error)
}

// BackupHandler provides methods for handling operations on backups.
type BackupHandler interface {
	GetBackup(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.Backup, error)
	CreateBackup(ctx context.Context, cluster string, backup *backupv1alpha1.Backup) (*backupv1alpha1.Backup, error)
	DeleteBackup(ctx context.Context, cluster, namespace, name string, params *api.DeleteBackupParams) error
}

// RestoreHandler provides methods for handling operations on restores.
type RestoreHandler interface {
	GetRestore(ctx context.Context, namespace, name string) (*backupv1alpha1.Restore, error)
	CreateRestore(ctx context.Context, restore *backupv1alpha1.Restore) (*backupv1alpha1.Restore, error)
	DeleteRestore(ctx context.Context, namespace, name string) error
}

// InstanceBackupHandler provides methods for handling operations on instance backups.
type InstanceBackupHandler interface {
	ListInstanceBackups(ctx context.Context, cluster, namespace, instance string) (*backupv1alpha1.BackupList, error)
}

// MonitoringConfigHandler provides methods for handling operations on monitoring configs.
type MonitoringConfigHandler interface {
	CreateMonitoringConfig(ctx context.Context, cluster, namespace string, req *api.MonitoringConfigCreateParams) (*monitoringv1alpha1.MonitoringConfig, error)
	UpdateMonitoringConfig(ctx context.Context, cluster, namespace, name string, req *api.MonitoringConfigUpdateParams) (*monitoringv1alpha1.MonitoringConfig, error)
	ListMonitoringConfigs(ctx context.Context, cluster, namespaces string) (*monitoringv1alpha1.MonitoringConfigList, error)
	GetMonitoringConfig(ctx context.Context, cluster, namespace, name string) (*monitoringv1alpha1.MonitoringConfig, error)
	DeleteMonitoringConfig(ctx context.Context, cluster, namespace, name string) error
}

// InstanceRestoreHandler provides methods for handling operations on instance restores.
type InstanceRestoreHandler interface {
	ListInstanceRestores(ctx context.Context, cluster, namespace, instanceName string) (*backupv1alpha1.RestoreList, error)
}
