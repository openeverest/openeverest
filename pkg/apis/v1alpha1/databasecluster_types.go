// Package v1alpha1 contains API definitions for OpenEverest
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	everestapi "github.com/percona/everest/api"
)

type (
	DatabaseClusterSpecEngineType                                           = everestapi.DatabaseClusterSpecEngineType
	DatabaseClusterSpecDataSourcePitrType                                   = everestapi.DatabaseClusterSpecDataSourcePitrType
	DatabaseCluster_Spec_Engine_Resources_Cpu                               = everestapi.DatabaseCluster_Spec_Engine_Resources_Cpu
	DatabaseCluster_Spec_Engine_Resources_Limits_Cpu                        = everestapi.DatabaseCluster_Spec_Engine_Resources_Limits_Cpu
	DatabaseCluster_Spec_Engine_Resources_Limits_Memory                     = everestapi.DatabaseCluster_Spec_Engine_Resources_Limits_Memory
	DatabaseCluster_Spec_Engine_Resources_Memory                            = everestapi.DatabaseCluster_Spec_Engine_Resources_Memory
	DatabaseCluster_Spec_Engine_Resources_Requests_Cpu                      = everestapi.DatabaseCluster_Spec_Engine_Resources_Requests_Cpu
	DatabaseCluster_Spec_Engine_Resources_Requests_Memory                   = everestapi.DatabaseCluster_Spec_Engine_Resources_Requests_Memory
	DatabaseCluster_Spec_Engine_Storage_Size                                = everestapi.DatabaseCluster_Spec_Engine_Storage_Size
	DatabaseCluster_Spec_Monitoring_Resources_Limits_AdditionalProperties   = everestapi.DatabaseCluster_Spec_Monitoring_Resources_Limits_AdditionalProperties
	DatabaseCluster_Spec_Monitoring_Resources_Requests_AdditionalProperties = everestapi.DatabaseCluster_Spec_Monitoring_Resources_Requests_AdditionalProperties
	DatabaseCluster_Spec_Proxy_Resources_Cpu                                = everestapi.DatabaseCluster_Spec_Proxy_Resources_Cpu
	DatabaseCluster_Spec_Proxy_Resources_Limits_Cpu                         = everestapi.DatabaseCluster_Spec_Proxy_Resources_Limits_Cpu
	DatabaseCluster_Spec_Proxy_Resources_Limits_Memory                      = everestapi.DatabaseCluster_Spec_Proxy_Resources_Limits_Memory
	DatabaseCluster_Spec_Proxy_Resources_Memory                             = everestapi.DatabaseCluster_Spec_Proxy_Resources_Memory
	DatabaseCluster_Spec_Proxy_Resources_Requests_Cpu                       = everestapi.DatabaseCluster_Spec_Proxy_Resources_Requests_Cpu
	DatabaseCluster_Spec_Proxy_Resources_Requests_Memory                    = everestapi.DatabaseCluster_Spec_Proxy_Resources_Requests_Memory
	DatabaseCluster_Spec_Proxy_Storage_Size                                 = everestapi.DatabaseCluster_Spec_Proxy_Storage_Size
	DatabaseClusterSpecProxyExposeType                                      = everestapi.DatabaseClusterSpecProxyExposeType
	DatabaseClusterSpecProxyType                                            = everestapi.DatabaseClusterSpecProxyType
	DatabaseClusterStatusConditionsStatus                                   = everestapi.DatabaseClusterStatusConditionsStatus
)

const (
	DatabaseClusterSpecEngineTypePostgresql = everestapi.DatabaseClusterSpecEngineTypePostgresql
	DatabaseClusterSpecEngineTypePsmdb      = everestapi.DatabaseClusterSpecEngineTypePsmdb
	DatabaseClusterSpecEngineTypePxc        = everestapi.DatabaseClusterSpecEngineTypePxc
)

// DatabaseClusterSpec defines the desired state of DatabaseCluster
// +kubebuilder:object:generate=true
// +kubebuilder:resource:scope=Namespaced,shortName=dc
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Type,path=.spec.engine.type,type=string,description=Database engine type
// +kubebuilder:printcolumn:name=Status,path=.status.status,type=string,description=Cluster status
// +kubebuilder:printcolumn:name=Age,path=.metadata.creationTimestamp,type=date
// +kubebuilder:default:{spec:{engine:{type:"postgresql",storage:{size:"1G"}}}}
type DatabaseClusterSpec struct {
	// AllowUnsafeConfiguration AllowUnsafeConfiguration field used to ensure that the user can create configurations unfit for production use.
	//
	// Deprecated: AllowUnsafeConfiguration will not be supported in the future releases.
	AllowUnsafeConfiguration *bool `json:"allowUnsafeConfiguration,omitempty" yaml:"allowUnsafeConfiguration,omitempty"`

	// Backup Backup is the backup specification
	Backup *struct {
		// Enabled Enabled is a flag to enable backups
		// Deprecated. Please use db.spec.backup.schedules[].enabled to control each schedule separately and db.spec.backup.pitr.enabled to control PITR.
		Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

		// Pitr PITR is the configuration of the point in time recovery
		Pitr *struct {
			// BackupStorageName BackupStorageName is the name of the BackupStorage where the PITR is enabled
			// The BackupStorage must be created in the same namespace as the DatabaseCluster.
			BackupStorageName *string `json:"backupStorageName,omitempty" yaml:"backupStorageName,omitempty"`

			// Enabled Enabled is a flag to enable PITR
			Enabled bool `json:"enabled" yaml:"enabled"`

			// UploadIntervalSec UploadIntervalSec number of seconds between the binlogs uploads
			UploadIntervalSec *int `json:"uploadIntervalSec,omitempty" yaml:"uploadIntervalSec,omitempty"`
		} `json:"pitr,omitempty" yaml:"pitr,omitempty"`

		// Schedules Schedules is a list of backup schedules
		Schedules *[]struct {
			// BackupStorageName BackupStorageName is the name of the BackupStorage CR that defines the
			// storage location.
			// The BackupStorage must be created in the same namespace as the DatabaseCluster.
			BackupStorageName string `json:"backupStorageName" yaml:"backupStorageName"`

			// Enabled Enabled is a flag to enable the schedule
			Enabled bool `json:"enabled" yaml:"enabled"`

			// Name Name is the name of the schedule
			Name string `json:"name" yaml:"name"`

			// RetentionCopies RetentionCopies is the number of backup copies to retain
			RetentionCopies *int32 `json:"retentionCopies,omitempty" yaml:"retentionCopies,omitempty"`

			// Schedule Schedule is the cron schedule
			Schedule string `json:"schedule" yaml:"schedule"`
		} `json:"schedules,omitempty" yaml:"schedules,omitempty"`
	} `json:"backup,omitempty" yaml:"backup,omitempty"`

	// DataSource DataSource defines a data source for bootstraping a new cluster
	DataSource *struct {
		// BackupSource BackupSource is the backup source to restore from
		BackupSource *struct {
			// BackupStorageName BackupStorageName is the name of the BackupStorage used for storing backups.
			// The BackupStorage must be created in the same namespace as the DatabaseCluster.
			BackupStorageName string `json:"backupStorageName" yaml:"backupStorageName"`

			// Path Path is the path to the backup file/directory.
			Path string `json:"path" yaml:"path"`
		} `json:"backupSource,omitempty" yaml:"backupSource,omitempty"`

		// DataImport DataImport allows importing data from an external backup source.
		DataImport *struct {
			// Config Config defines the configuration for the data import job.
			// These options are specific to the DataImporter being used and must conform to
			// the schema defined in the DataImporter's .spec.config.openAPIV3Schema.
			Config *map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`

			// DataImporterName DataImporterName is the data importer to use for the import.
			DataImporterName string `json:"dataImporterName" yaml:"dataImporterName"`

			// Source Source is the source of the data to import.
			Source struct {
				// Path Path is the path to the directory to import the data from.
				// This may be a path to a file or a directory, depending on the data importer.
				// Only absolute file paths are allowed. Leading and trailing '/' are optional.
				Path string `json:"path" yaml:"path"`

				// S3 S3 contains the S3 information for the data import.
				S3 *struct {
					// AccessKeyId AccessKeyID allows specifying the S3 access key ID inline.
					// It is provided as a write-only input field for convenience.
					// When this field is set, a webhook writes this value in the Secret specified by `credentialsSecretName`
					// and empties this field.
					// This field is not stored in the API.
					AccessKeyId *string `json:"accessKeyId,omitempty" yaml:"accessKeyId,omitempty"`

					// Bucket Bucket is the name of the S3 bucket.
					Bucket string `json:"bucket" yaml:"bucket"`

					// CredentialsSecretName CredentialsSecreName is the reference to the secret containing the S3 credentials.
					// The Secret must contain the keys `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.
					CredentialsSecretName string `json:"credentialsSecretName" yaml:"credentialsSecretName"`

					// EndpointURL EndpointURL is an endpoint URL of backup storage.
					EndpointURL string `json:"endpointURL" yaml:"endpointURL"`

					// ForcePathStyle ForcePathStyle is set to use path-style URLs.
					// If unspecified, the default value is false.
					ForcePathStyle *bool `json:"forcePathStyle,omitempty" yaml:"forcePathStyle,omitempty"`

					// Region Region is the region of the S3 bucket.
					Region string `json:"region" yaml:"region"`

					// SecretAccessKey SecretAccessKey allows specifying the S3 secret access key inline.
					// It is provided as a write-only input field for convenience.
					// When this field is set, a webhook writes this value in the Secret specified by `credentialsSecretName`
					// and empties this field.
					// This field is not stored in the API.
					SecretAccessKey *string `json:"secretAccessKey,omitempty" yaml:"secretAccessKey,omitempty"`

					// VerifyTLS VerifyTLS is set to ensure TLS/SSL verification.
					// If unspecified, the default value is true.
					VerifyTLS *bool `json:"verifyTLS,omitempty" yaml:"verifyTLS,omitempty"`
				} `json:"s3,omitempty" yaml:"s3,omitempty"`
			} `json:"source" yaml:"source"`
		} `json:"dataImport,omitempty" yaml:"dataImport,omitempty"`

		// DbClusterBackupName DBClusterBackupName is the name of the DB cluster backup to restore from
		DbClusterBackupName *string `json:"dbClusterBackupName,omitempty" yaml:"dbClusterBackupName,omitempty"`

		// Pitr PITR is the point-in-time recovery configuration
		Pitr *struct {
			// Date Date is the UTC date to recover to. The accepted format: "2006-01-02T15:04:05Z".
			Date *string `json:"date,omitempty" yaml:"date,omitempty"`

			// Type Type is the type of recovery.
			Type *DatabaseClusterSpecDataSourcePitrType `json:"type,omitempty" yaml:"type,omitempty"`
		} `json:"pitr,omitempty" yaml:"pitr,omitempty"`
	} `json:"dataSource,omitempty" yaml:"dataSource,omitempty"`

	// Engine Engine is the database engine specification
	Engine struct {
		// Config Config is the engine configuration
		Config *string `json:"config,omitempty" yaml:"config,omitempty"`

		// CrVersion CRVersion is the desired version of the CR to use with the
		// underlying operator.
		// If unspecified, everest-operator will use the same version as the operator.
		//
		// NOTE: Updating this property post installation may lead to a restart of the cluster.
		CrVersion *string `json:"crVersion,omitempty" yaml:"crVersion,omitempty"`

		// Replicas Replicas is the number of engine replicas
		Replicas *int32 `json:"replicas,omitempty" yaml:"replicas,omitempty"`

		// Resources Resources are the resource limits for each engine replica.
		// If not set, resource limits are not imposed
		Resources *struct {
			// Cpu CPU is the CPU resource requirements.
			// Deprecated: use limits.cpu instead.
			Cpu *DatabaseCluster_Spec_Engine_Resources_Cpu `json:"cpu,omitempty" yaml:"cpu,omitempty"`

			// Limits Limits are the resource limits applied to each replica.
			// If set, it takes precedence over the deprecated cpu and memory fields.
			Limits *struct {
				// Cpu CPU is the CPU resource requirements
				Cpu *DatabaseCluster_Spec_Engine_Resources_Limits_Cpu `json:"cpu,omitempty" yaml:"cpu,omitempty"`

				// Memory Memory is the memory resource requirements
				Memory *DatabaseCluster_Spec_Engine_Resources_Limits_Memory `json:"memory,omitempty" yaml:"memory,omitempty"`
			} `json:"limits,omitempty" yaml:"limits,omitempty"`

			// Memory Memory is the memory resource requirements.
			// Deprecated: use limits.memory instead.
			Memory *DatabaseCluster_Spec_Engine_Resources_Memory `json:"memory,omitempty" yaml:"memory,omitempty"`

			// Requests Requests are the resource requests applied to each replica.
			// If unset, the request behavior is engine-specific.
			Requests *struct {
				// Cpu CPU is the CPU resource requirements
				Cpu *DatabaseCluster_Spec_Engine_Resources_Requests_Cpu `json:"cpu,omitempty" yaml:"cpu,omitempty"`

				// Memory Memory is the memory resource requirements
				Memory *DatabaseCluster_Spec_Engine_Resources_Requests_Memory `json:"memory,omitempty" yaml:"memory,omitempty"`
			} `json:"requests,omitempty" yaml:"requests,omitempty"`
		} `json:"resources,omitempty" yaml:"resources,omitempty"`

		// Storage Storage is the engine storage configuration
		Storage struct {
			// Class Class is the storage class to use for the persistent volume claim
			Class *string `json:"class,omitempty" yaml:"class,omitempty"`

			// Size Size is the size of the persistent volume claim
			Size DatabaseCluster_Spec_Engine_Storage_Size `json:"size" yaml:"size"`
		} `json:"storage" yaml:"storage"`

		// Type Type is the engine type
		Type DatabaseClusterSpecEngineType `json:"type" yaml:"type"`

		// UserSecretsName UserSecretsName is the name of the secret containing the user secrets
		UserSecretsName *string `json:"userSecretsName,omitempty" yaml:"userSecretsName,omitempty"`

		// Version Version is the engine version
		Version *string `json:"version,omitempty" yaml:"version,omitempty"`
	} `json:"engine" yaml:"engine"`

	// EngineFeatures EngineFeatures represents configuration of additional features for the database engine.
	EngineFeatures *struct {
		// Psmdb PSMDB represents additional features for the PSMDB engine.
		Psmdb *struct {
			// SplitHorizonDnsConfigName SplitHorizonDNSConfigName is the name of a SplitHorizonDNSConfig CR.
			// The SplitHorizonDNSConfig must be created in the same namespace as the DatabaseCluster.
			SplitHorizonDnsConfigName *string `json:"splitHorizonDnsConfigName,omitempty" yaml:"splitHorizonDnsConfigName,omitempty"`
		} `json:"psmdb,omitempty" yaml:"psmdb,omitempty"`
	} `json:"engineFeatures,omitempty" yaml:"engineFeatures,omitempty"`

	// Monitoring Monitoring is the monitoring configuration
	Monitoring *struct {
		// MonitoringConfigName MonitoringConfigName is the name of a monitoringConfig CR.
		// The MonitoringConfig must be created in the same namespace as the DatabaseCluster.
		MonitoringConfigName *string `json:"monitoringConfigName,omitempty" yaml:"monitoringConfigName,omitempty"`

		// Resources Resources defines resource limitations for the monitoring.
		Resources *struct {
			// Claims Claims lists the names of resources, defined in spec.resourceClaims,
			// that are used by this container.
			//
			// This field depends on the
			// DynamicResourceAllocation feature gate.
			//
			// This field is immutable. It can only be set for containers.
			Claims *[]struct {
				// Name Name must match the name of one entry in pod.spec.resourceClaims of
				// the Pod where this field is used. It makes that resource available
				// inside a container.
				Name string `json:"name" yaml:"name"`

				// Request Request is the name chosen for a request in the referenced claim.
				// If empty, everything from the claim is made available, otherwise
				// only the result of this request.
				Request *string `json:"request,omitempty" yaml:"request,omitempty"`
			} `json:"claims,omitempty" yaml:"claims,omitempty"`

			// Limits Limits describes the maximum amount of compute resources allowed.
			// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
			Limits *map[string]DatabaseCluster_Spec_Monitoring_Resources_Limits_AdditionalProperties `json:"limits,omitempty" yaml:"limits,omitempty"`

			// Requests Requests describes the minimum amount of compute resources required.
			// If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
			// otherwise to an implementation-defined value. Requests cannot exceed Limits.
			// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
			Requests *map[string]DatabaseCluster_Spec_Monitoring_Resources_Requests_AdditionalProperties `json:"requests,omitempty" yaml:"requests,omitempty"`
		} `json:"resources,omitempty" yaml:"resources,omitempty"`
	} `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`

	// Paused Paused is a flag to stop the cluster
	Paused *bool `json:"paused,omitempty" yaml:"paused,omitempty"`

	// PodSchedulingPolicyName PodSchedulingPolicyName is the name of the PodSchedulingPolicy CR that defines rules for DB cluster pods allocation across the cluster.
	PodSchedulingPolicyName *string `json:"podSchedulingPolicyName,omitempty" yaml:"podSchedulingPolicyName,omitempty"`

	// Proxy Proxy is the proxy specification. If not set, an appropriate
	// proxy specification will be applied for the given engine. A
	// common use case for setting this field is to control the
	// external access to the database cluster.
	Proxy *struct {
		// Config Config is the proxy configuration
		Config *string `json:"config,omitempty" yaml:"config,omitempty"`

		// Expose Expose is the proxy expose configuration
		Expose *struct {
			// IpSourceRanges IPSourceRanges is the list of IP source ranges (CIDR notation)
			// to allow access from. If not set, there is no limitations
			IpSourceRanges *[]string `json:"ipSourceRanges,omitempty" yaml:"ipSourceRanges,omitempty"`

			// LoadBalancerConfigName LoadBalancerConfigName is the name of load balancer config if applied
			LoadBalancerConfigName *string `json:"loadBalancerConfigName,omitempty" yaml:"loadBalancerConfigName,omitempty"`

			// Type Type is the expose type, can be ClusterIP, LoadBalancer, NodePort. The types internal and external are deprecated.
			Type *DatabaseClusterSpecProxyExposeType `json:"type,omitempty" yaml:"type,omitempty"`
		} `json:"expose,omitempty" yaml:"expose,omitempty"`

		// Replicas Replicas is the number of proxy replicas
		Replicas *int32 `json:"replicas,omitempty" yaml:"replicas,omitempty"`

		// Resources Resources are the resource limits for each proxy replica.
		// If not set, resource limits are not imposed
		Resources *struct {
			// Cpu CPU is the CPU resource requirements.
			// Deprecated: use limits.cpu instead.
			Cpu *DatabaseCluster_Spec_Proxy_Resources_Cpu `json:"cpu,omitempty" yaml:"cpu,omitempty"`

			// Limits Limits are the resource limits applied to each replica.
			// If set, it takes precedence over the deprecated cpu and memory fields.
			Limits *struct {
				// Cpu CPU is the CPU resource requirements
				Cpu *DatabaseCluster_Spec_Proxy_Resources_Limits_Cpu `json:"cpu,omitempty" yaml:"cpu,omitempty"`

				// Memory Memory is the memory resource requirements
				Memory *DatabaseCluster_Spec_Proxy_Resources_Limits_Memory `json:"memory,omitempty" yaml:"memory,omitempty"`
			} `json:"limits,omitempty" yaml:"limits,omitempty"`

			// Memory Memory is the memory resource requirements.
			// Deprecated: use limits.memory instead.
			Memory *DatabaseCluster_Spec_Proxy_Resources_Memory `json:"memory,omitempty" yaml:"memory,omitempty"`

			// Requests Requests are the resource requests applied to each replica.
			// If unset, the request behavior is engine-specific.
			Requests *struct {
				// Cpu CPU is the CPU resource requirements
				Cpu *DatabaseCluster_Spec_Proxy_Resources_Requests_Cpu `json:"cpu,omitempty" yaml:"cpu,omitempty"`

				// Memory Memory is the memory resource requirements
				Memory *DatabaseCluster_Spec_Proxy_Resources_Requests_Memory `json:"memory,omitempty" yaml:"memory,omitempty"`
			} `json:"requests,omitempty" yaml:"requests,omitempty"`
		} `json:"resources,omitempty" yaml:"resources,omitempty"`

		// Storage Storage is the proxy storage configuration
		Storage *struct {
			// Class Class is the storage class to use for the persistent volume claim
			Class *string `json:"class,omitempty" yaml:"class,omitempty"`

			// Size Size is the size of the persistent volume claim
			Size DatabaseCluster_Spec_Proxy_Storage_Size `json:"size" yaml:"size"`
		} `json:"storage,omitempty" yaml:"storage,omitempty"`

		// Type Type is the proxy type
		Type *DatabaseClusterSpecProxyType `json:"type,omitempty" yaml:"type,omitempty"`
	} `json:"proxy,omitempty" yaml:"proxy,omitempty"`

	// Sharding Sharding is the sharding configuration. PSMDB-only
	Sharding *struct {
		// ConfigServer ConfigServer represents the sharding configuration server settings
		ConfigServer struct {
			// Replicas Replicas is the amount of configServers
			Replicas int32 `json:"replicas" yaml:"replicas"`
		} `json:"configServer" yaml:"configServer"`

		// Enabled Enabled defines if the sharding is enabled
		Enabled bool `json:"enabled" yaml:"enabled"`

		// Shards Shards defines the number of shards
		Shards int32 `json:"shards" yaml:"shards"`
	} `json:"sharding,omitempty" yaml:"sharding,omitempty"`

	// TLS TLS configuration for the database cluster
	// +optional
	TLS *TLSSpec `json:"tls,omitempty" yaml:"tls,omitempty"`
}

// DatabaseClusterStatus defines the observed state of DatabaseCluster
// +kubebuilder:object:generate=true
type DatabaseClusterStatus struct {
	// ActiveStorage ActiveStorage is the storage used in cluster (psmdb only)
	ActiveStorage *string `json:"activeStorage,omitempty" yaml:"activeStorage,omitempty"`

	// Conditions Conditions contains the observed conditions of the DatabaseCluster.
	Conditions *[]struct {
		// LastTransitionTime lastTransitionTime is the last time the condition transitioned from one status to another.
		// This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.
		LastTransitionTime metav1.Time `json:"lastTransitionTime" yaml:"lastTransitionTime"`

		// Message message is a human readable message indicating details about the transition.
		// This may be an empty string.
		Message string `json:"message" yaml:"message"`

		// ObservedGeneration observedGeneration represents the .metadata.generation that the condition was set based upon.
		// For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
		// with respect to the current state of the instance.
		ObservedGeneration *int64 `json:"observedGeneration,omitempty" yaml:"observedGeneration,omitempty"`

		// Reason reason contains a programmatic identifier indicating the reason for the condition's last transition.
		// Producers of specific condition types may define expected values and meanings for this field,
		// and whether the values are considered a guaranteed API.
		// The value should be a CamelCase string.
		// This field may not be empty.
		Reason string `json:"reason" yaml:"reason"`

		// Status status of the condition, one of True, False, Unknown.
		Status DatabaseClusterStatusConditionsStatus `json:"status" yaml:"status"`

		// Type type of condition in CamelCase or in foo.example.com/CamelCase.
		Type string `json:"type" yaml:"type"`
	} `json:"conditions,omitempty" yaml:"conditions,omitempty"`

	// CrVersion CRVersion is the observed version of the CR used with the underlying operator.
	CrVersion *string `json:"crVersion,omitempty" yaml:"crVersion,omitempty"`

	// DataImportJobName DataImportJobName refers to the DataImportJob that is used to import data into the cluster.
	// This is set only when .spec.dataSource.dataImport is set.
	DataImportJobName *string `json:"dataImportJobName,omitempty" yaml:"dataImportJobName,omitempty"`

	// Details Details provides full status of the upstream cluster as a plain text.
	Details *string `json:"details,omitempty" yaml:"details,omitempty"`

	// EngineFeatures EngineFeaturesStatus represents additional features statuses for the database engine.
	EngineFeatures *struct {
		// Psmdb PSMDB represents additional features statuses for the PSMDB engine.
		Psmdb *struct {
			// SplitHorizon SplitHorizon status of SplitHorizon feature.
			SplitHorizon *struct {
				// Domains SplitHorizon status of SplitHorizon feature.
				Domains *[]struct {
					// Domain Domain is the SplitHorizon domain name.
					Domain *string `json:"domain,omitempty" yaml:"domain,omitempty"`

					// PrivateIP PrivateIP is the private IP address for the domain.
					PrivateIP *string `json:"privateIP,omitempty" yaml:"privateIP,omitempty"`

					// PublicIP PublicIP is the public IP address for the domain.
					PublicIP *string `json:"publicIP,omitempty" yaml:"publicIP,omitempty"`
				} `json:"domains,omitempty" yaml:"domains,omitempty"`

				// Host ConnectionURL is the connection URL using SplitHorizon domains.
				Host *string `json:"host,omitempty" yaml:"host,omitempty"`
			} `json:"splitHorizon,omitempty" yaml:"splitHorizon,omitempty"`
		} `json:"psmdb,omitempty" yaml:"psmdb,omitempty"`
	} `json:"engineFeatures,omitempty" yaml:"engineFeatures,omitempty"`

	// Hostname Hostname is the hostname where the cluster can be reached
	Hostname *string `json:"hostname,omitempty" yaml:"hostname,omitempty"`

	// Message Message is extra information about the cluster
	Message *string `json:"message,omitempty" yaml:"message,omitempty"`

	// ObservedGeneration ObservedGeneration is the most recent generation observed for this DatabaseCluster.
	ObservedGeneration *int64 `json:"observedGeneration,omitempty" yaml:"observedGeneration,omitempty"`

	// Port Port is the port where the cluster can be reached
	Port *int32 `json:"port,omitempty" yaml:"port,omitempty"`

	// Ready Ready is the number of ready pods
	Ready *int32 `json:"ready,omitempty" yaml:"ready,omitempty"`

	// RecommendedCRVersion RecommendedCRVersion indicates the target version that the underlying CR should be updated to.
	// When this field is set, it means the CR is running an outdated version and requires an update.
	// The following restrictions apply until the CR is updated to the recommended version:
	// - The operator cannot be upgraded
	// - The database engine version (.spec.engine.version) cannot be modified
	// This field is unset when the CR is already running at the latest recommended version.
	RecommendedCRVersion *string `json:"recommendedCRVersion,omitempty" yaml:"recommendedCRVersion,omitempty"`

	// Size Size is the total number of pods
	Size *int32 `json:"size,omitempty" yaml:"size,omitempty"`

	// Status Status is the status of the cluster
	Status *string `json:"status,omitempty" yaml:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dc
type DatabaseCluster struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   *DatabaseClusterSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status *DatabaseClusterStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DatabaseClusterList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items           []DatabaseCluster `json:"items" yaml:"items"`
}
