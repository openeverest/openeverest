// Package validators provides validation utilities for OpenEverest
package validators

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	everestv1alpha1 "github.com/percona/everest/pkg/apis/v1alpha1"
	"github.com/percona/everest/pkg/translation"
)

// ValidateDatabaseCluster validates a DatabaseCluster
func ValidateDatabaseCluster(dc *everestv1alpha1.DatabaseCluster) field.ErrorList {
	var allErrs field.ErrorList

	if dc == nil || dc.Spec == nil {
		allErrs = append(allErrs, field.Required(field.NewPath("spec"), "spec is required"))
		return allErrs
	}

	// Validate TLS spec if present
	if dc.Spec.TLS != nil {
		// Apply defaulting if mode is empty
		dc.Spec.TLS.Default()

		allErrs = append(allErrs, dc.Spec.TLS.Validate(field.NewPath("spec").Child("tls"))...)
		allErrs = append(allErrs, translation.ValidateCustomSpecConflict(dc)...)
	}

	return allErrs
}
