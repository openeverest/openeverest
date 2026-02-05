// Package handlers contains the interface and types for the Everest API handlers.
package handlers

//go:generate go tool mockery --name=Handler --case=snake --inpackage

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	enginefeaturesv1alpha1 "github.com/percona/everest-operator/api/enginefeatures.everest/v1alpha1"
	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
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
	DatabaseClusterHandler
	DatabaseClusterBackupHandler
	DatabaseClusterRestoreHandler
	DatabaseEngineHandler
	BackupStorageHandler
	MonitoringInstanceHandler
	PodSchedulingPolicyHandler
	LoadBalancerConfigHandler
	DataImporterHandler
	DataImportJobHandler
	EngineFeaturesHandler

	GetKubernetesClusterResources(ctx context.Context) (*api.KubernetesClusterResources, error)
	GetKubernetesClusterInfo(ctx context.Context) (*api.KubernetesClusterInfo, error)
	GetUserPermissions(ctx context.Context) (*api.UserPermissions, error)
	GetSettings(ctx context.Context) (*api.Settings, error)
}

type StreamFunc func(ctx context.Context, namespace, clusterName, componentName string, params api.GetDatabaseClusterComponentLogsParams) error

// DatabaseClusterHandler provides methods for handling operations on database clusters.
type DatabaseClusterHandler interface {
	CreateDatabaseCluster(ctx context.Context, req *everestv1alpha1.DatabaseCluster) (*everestv1alpha1.DatabaseCluster, error)
	UpdateDatabaseCluster(ctx context.Context, req *everestv1alpha1.DatabaseCluster) (*everestv1alpha1.DatabaseCluster, error)
	ListDatabaseClusters(ctx context.Context, namespace string) (*everestv1alpha1.DatabaseClusterList, error)
	DeleteDatabaseCluster(ctx context.Context, namespace, name string, delReq *api.DeleteDatabaseClusterParams) error
	GetDatabaseCluster(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseCluster, error)
	GetDatabaseClusterCredentials(ctx context.Context, namespace, name string) (*api.DatabaseClusterCredential, error)
	GetDatabaseClusterComponents(ctx context.Context, namespace, name string) ([]api.DatabaseClusterComponent, error)
	GetDatabaseClusterComponentLogs(ctx context.Context, namespace, clusterName, componentName string, params api.GetDatabaseClusterComponentLogsParams, stream StreamFunc) error
	GetDatabaseClusterPitr(ctx context.Context, namespace, name string) (*api.DatabaseClusterPitr, error)
	CreateDatabaseClusterSecret(ctx context.Context, namespace, dbName string, secret *corev1.Secret) (*corev1.Secret, error)
}

// NamespacesHandler provides methods for handling operations on namespaces.
type NamespacesHandler interface {
	ListNamespaces(ctx context.Context) ([]string, error)
}

// DatabaseClusterBackupHandler provides methods for handling operations on database cluster backups.
type DatabaseClusterBackupHandler interface {
	GetDatabaseClusterBackup(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseClusterBackup, error)
	ListDatabaseClusterBackups(ctx context.Context, namespace, clusterName string) (*everestv1alpha1.DatabaseClusterBackupList, error)
	CreateDatabaseClusterBackup(ctx context.Context, req *everestv1alpha1.DatabaseClusterBackup) (*everestv1alpha1.DatabaseClusterBackup, error)
	DeleteDatabaseClusterBackup(ctx context.Context, namespace, name string, req *api.DeleteDatabaseClusterBackupParams) error
}

// DatabaseClusterRestoreHandler provides methods for handling operations on database cluster restores.
type DatabaseClusterRestoreHandler interface {
	CreateDatabaseClusterRestore(ctx context.Context, req *everestv1alpha1.DatabaseClusterRestore) (*everestv1alpha1.DatabaseClusterRestore, error)
	UpdateDatabaseClusterRestore(ctx context.Context, req *everestv1alpha1.DatabaseClusterRestore) (*everestv1alpha1.DatabaseClusterRestore, error)
	GetDatabaseClusterRestore(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseClusterRestore, error)
	ListDatabaseClusterRestores(ctx context.Context, namespace, clusterName string) (*everestv1alpha1.DatabaseClusterRestoreList, error)
	DeleteDatabaseClusterRestore(ctx context.Context, namespace, name string) error
}

// DatabaseEngineHandler provides methods for handling operations on database engines.
type DatabaseEngineHandler interface {
	UpdateDatabaseEngine(ctx context.Context, req *everestv1alpha1.DatabaseEngine) (*everestv1alpha1.DatabaseEngine, error)
	ListDatabaseEngines(ctx context.Context, namespace string) (*everestv1alpha1.DatabaseEngineList, error)
	GetDatabaseEngine(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseEngine, error)
	GetUpgradePlan(ctx context.Context, namespace string) (*api.UpgradePlan, error)
	ApproveUpgradePlan(ctx context.Context, namespace string) error
}

// BackupStorageHandler provides methods for handling operations on backup storages.
type BackupStorageHandler interface {
	CreateBackupStorage(ctx context.Context, namespace string, req *api.CreateBackupStorageParams) (*everestv1alpha1.BackupStorage, error)
	UpdateBackupStorage(ctx context.Context, name, namespace string, req *api.UpdateBackupStorageParams) (*everestv1alpha1.BackupStorage, error)
	ListBackupStorages(ctx context.Context, namespace string) (*everestv1alpha1.BackupStorageList, error)
	GetBackupStorage(ctx context.Context, namespace, name string) (*everestv1alpha1.BackupStorage, error)
	DeleteBackupStorage(ctx context.Context, namespace, name string) error
}

// MonitoringInstanceHandler provides methods for handling operations on monitoring instances.
type MonitoringInstanceHandler interface {
	CreateMonitoringInstance(ctx context.Context, namespace string, req *api.CreateMonitoringInstanceJSONRequestBody) (*everestv1alpha1.MonitoringConfig, error)
	UpdateMonitoringInstance(ctx context.Context, namespace, name string, req *api.UpdateMonitoringInstanceJSONRequestBody) (*everestv1alpha1.MonitoringConfig, error)
	ListMonitoringInstances(ctx context.Context, namespaces string) (*everestv1alpha1.MonitoringConfigList, error)
	GetMonitoringInstance(ctx context.Context, namespace, name string) (*everestv1alpha1.MonitoringConfig, error)
	DeleteMonitoringInstance(ctx context.Context, namespace, name string) error
}

// PodSchedulingPolicyHandler provides methods for handling operations on pod scheduling policies.
type PodSchedulingPolicyHandler interface {
	CreatePodSchedulingPolicy(ctx context.Context, psp *everestv1alpha1.PodSchedulingPolicy) (*everestv1alpha1.PodSchedulingPolicy, error)
	UpdatePodSchedulingPolicy(ctx context.Context, psp *everestv1alpha1.PodSchedulingPolicy) (*everestv1alpha1.PodSchedulingPolicy, error)
	ListPodSchedulingPolicies(ctx context.Context, params *api.ListPodSchedulingPolicyParams) (*everestv1alpha1.PodSchedulingPolicyList, error)
	DeletePodSchedulingPolicy(ctx context.Context, name string) error
	GetPodSchedulingPolicy(ctx context.Context, name string) (*everestv1alpha1.PodSchedulingPolicy, error)
}

// LoadBalancerConfigHandler provides methods for handling operations on load balancer configs.
type LoadBalancerConfigHandler interface {
	CreateLoadBalancerConfig(ctx context.Context, psp *everestv1alpha1.LoadBalancerConfig) (*everestv1alpha1.LoadBalancerConfig, error)
	UpdateLoadBalancerConfig(ctx context.Context, psp *everestv1alpha1.LoadBalancerConfig) (*everestv1alpha1.LoadBalancerConfig, error)
	ListLoadBalancerConfigs(ctx context.Context) (*everestv1alpha1.LoadBalancerConfigList, error)
	DeleteLoadBalancerConfig(ctx context.Context, name string) error
	GetLoadBalancerConfig(ctx context.Context, name string) (*everestv1alpha1.LoadBalancerConfig, error)
}

// DataImporterHandler provides methods for handling operations on data importers.
type DataImporterHandler interface {
	ListDataImporters(ctx context.Context, supportedEngines ...string) (*everestv1alpha1.DataImporterList, error)
}

// DataImportJobHandler provides methods for handling operations on data import jobs.
type DataImportJobHandler interface {
	ListDataImportJobs(ctx context.Context, namespace, dbName string) (*everestv1alpha1.DataImportJobList, error)
}

//  ------ Engine Features interfaces ------

// EngineFeaturesHandler provides methods for handling operations on engine features.
type EngineFeaturesHandler interface {
	// PMDB engine features.

	SplitHorizonDNSConfigHandler
}

// SplitHorizonDNSConfigHandler provides methods for handling operations on Split-Horizon DNS Config.
type SplitHorizonDNSConfigHandler interface {
	CreateSplitHorizonDNSConfig(ctx context.Context, shdc *enginefeaturesv1alpha1.SplitHorizonDNSConfig) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error)
	UpdateSplitHorizonDNSConfig(ctx context.Context, namespace, name string, req *api.SplitHorizonDNSConfigUpdateParams) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error)
	ListSplitHorizonDNSConfigs(ctx context.Context, namespace string) (*enginefeaturesv1alpha1.SplitHorizonDNSConfigList, error)
	DeleteSplitHorizonDNSConfig(ctx context.Context, namespace, name string) error
	GetSplitHorizonDNSConfig(ctx context.Context, namespace, name string) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error)
}
