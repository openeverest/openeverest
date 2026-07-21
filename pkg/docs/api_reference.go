// Package docs provides API documentation for OpenEverest
package docs

import (
	everestv1alpha1 "github.com/percona/everest/pkg/apis/v1alpha1"
)

// TLSSpec documentation
// +k8s:openapi-gen=true
type TLSSpec = everestv1alpha1.TLSSpec

// DatabaseCluster documentation
// +k8s:openapi-gen=true
type DatabaseCluster = everestv1alpha1.DatabaseCluster
