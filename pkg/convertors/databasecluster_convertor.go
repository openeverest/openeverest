// Package convertors provides conversion utilities for OpenEverest
package convertors

import (
	everestv1alpha1 "github.com/percona/everest/pkg/apis/v1alpha1"
)

// ConvertDatabaseCluster converts from v1alpha1 to v1alpha1
func ConvertDatabaseCluster(src *everestv1alpha1.DatabaseCluster) *everestv1alpha1.DatabaseCluster {
	if src == nil {
		return nil
	}

	dst := &everestv1alpha1.DatabaseCluster{
		TypeMeta:   src.TypeMeta,
		ObjectMeta: src.ObjectMeta,
		Spec:       src.Spec.DeepCopy(),
		Status:     src.Status.DeepCopy(),
	}

	if src.Spec != nil && src.Spec.TLS != nil {
		dst.Spec.TLS = &everestv1alpha1.TLSSpec{
			Mode:                src.Spec.TLS.Mode,
			CASecretRef:         src.Spec.TLS.CASecretRef,
			ServerCertSecretRef: src.Spec.TLS.ServerCertSecretRef,
			ClientCertSecretRef: src.Spec.TLS.ClientCertSecretRef,
		}
	}

	return dst
}

// ConvertDatabaseClusterBack converts back from v1alpha1 to v1alpha1
func ConvertDatabaseClusterBack(src *everestv1alpha1.DatabaseCluster) *everestv1alpha1.DatabaseCluster {
	return ConvertDatabaseCluster(src)
}
