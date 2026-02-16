// Package v1alpha1 ...
package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=bc
//+kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName"
//+kubebuilder:printcolumn:name="Description",type="string",JSONPath=".spec.description"
//+kubebuilder:printcolumn:name="SupportedProviders",type="string",JSONPath=".spec.supportedProviders"
//+kubebuilder:resource:scope=Cluster

// BackupClass defines a reusable strategy for backing up data from a DataStore.
type BackupClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupClassSpec   `json:"spec,omitempty"`
	Status BackupClassStatus `json:"status,omitempty"`
}

// ProviderNameList is a type alias for a list of provider names.
type ProviderNameList []string

// Has checks if the list contains the specified provider.
func (e ProviderNameList) Has(provider string) bool {
	return slices.Contains(e, provider)
}

// BackupClassSpec defines the specification of a BackupClass.
type BackupClassSpec struct {
	// DisplayName is a human-readable name for the backup tool.
	DisplayName string `json:"displayName,omitempty"`
	// Description is the description of the backup tool.
	Description string `json:"description,omitempty"`
	// SupportedProviders is the list of providers that the backup tool supports.
	SupportedProviders ProviderNameList `json:"supportedProviders,omitempty"`
	// Config contains additional configuration defined for the backup tool.
	Config BackupClassConfig `json:"config,omitempty"`
	// JobSpec is the specification of the backup job.
	// +optional
	JobSpec *BackupJobSpec `json:"jobSpec,omitempty"`
	// CleanupJobSpec is the specification of the cleanup job.
	// +optional
	CleanupJobSpec *BackupJobSpec `json:"cleanupJobSpec,omitempty"`
	// DataStoreConstraints defines compatibility requirements and prerequisites that must be satisfied
	// by a DataStore before this backup tool can be used with it. This allows the backup tool to
	// express specific requirements about the database configuration needed for successful backup operations,
	// such as required database fields, specific engine configurations, or other database properties.
	// When a DataStore references this backup tool, the operator will validate the DataStore
	// against these constraints before proceeding with the backup operation.
	// +optional
	DataStoreConstraints BackupClassDataStoreConstraints `json:"dataStoreConstraints,omitempty"`
	// Permissions defines the permissions required by the backup tool.
	// These permissions are used to generate a Role for the backup job.
	// +optional
	Permissions []rbacv1.PolicyRule `json:"permissions,omitempty"`
	// ClusterPermissions defines the cluster-wide permissions required by the backup tool.
	// These permissions are used to generate a ClusterRole for the backup job.
	// +optional
	ClusterPermissions []rbacv1.PolicyRule `json:"clusterPermissions,omitempty"`
}

// BackupClassConfig contains additional configuration defined for the backup tool.
type BackupClassConfig struct {
	// OpenAPIV3Schema is the OpenAPI v3 schema of the backup tool.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	OpenAPIV3Schema *apiextensionsv1.JSONSchemaProps `json:"openAPIV3Schema,omitempty"`
}

// ErrSchemaValidationFailure is returned when the parameters do not conform to the BackupClass schema defined in .spec.config.
var ErrSchemaValidationFailure = errors.New("schema validation failed")

// Validate the config for the backup tool.
func (cfg *BackupClassConfig) Validate(params *runtime.RawExtension) error {
	schema := cfg.OpenAPIV3Schema
	if schema == nil && params != nil {
		return ErrSchemaValidationFailure
	}
	if schema == nil && params == nil {
		return nil
	}

	// Additional properties are implicitly disallowed
	schema.AdditionalProperties = &apiextensionsv1.JSONSchemaPropsOrBool{
		Allows: false,
	}

	// Unmarshal the parameters into a generic map
	var paramsMap map[string]interface{}
	if err := json.Unmarshal(params.Raw, &paramsMap); err != nil {
		return fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	// Convert the OpenAPI v3 schema to a JSON schema validator
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal OpenAPI v3 schema: %w", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaJSON))
	paramsLoader := gojsonschema.NewGoLoader(paramsMap)

	// Validate the parameters against the schema
	result, err := gojsonschema.Validate(schemaLoader, paramsLoader)
	if err != nil {
		return fmt.Errorf("failed to validate parameters: %w", err)
	}

	if !result.Valid() {
		var validationErrors []string
		for _, err := range result.Errors() {
			validationErrors = append(validationErrors, err.String())
		}
		return errors.Join(ErrSchemaValidationFailure, fmt.Errorf("validation errors: %s", strings.Join(validationErrors, "; ")))
	}
	return nil
}

// BackupJobSpec defines the specification for the Kubernetes job.
type BackupJobSpec struct {
	// Image is the image of the backup tool.
	Image string `json:"image,omitempty"`
	// Command is the command to run the backup tool.
	// +optional
	Command []string `json:"command,omitempty"`
}

// BackupClassDataStoreConstraints defines compatibility requirements and prerequisites
// that must be satisfied by a DataStore before this backup tool can be used with it.
type BackupClassDataStoreConstraints struct {
	// RequiredFields contains a list of fields that must be set in the DataStore spec.
	// Each key is a JSON path expressions that points to a field in the DataStore spec.
	// For example, ".spec.engine.type" or ".spec.dataSource.dataImport.config.someField".
	// +optional
	RequiredFields []string `json:"requiredFields,omitempty"`
}

// BackupClassList contains a list of BackupClass.
// +kubebuilder:object:root=true
type BackupClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupClass `json:"items"`
}

// BackupClassStatus defines the status of the BackupClass.
type BackupClassStatus struct{}

func init() {
	SchemeBuilder.Register(&BackupClass{}, &BackupClassList{})
}
