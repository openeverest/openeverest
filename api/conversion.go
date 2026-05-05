package api

import (
	"github.com/AlekSi/pointer"
	v1 "k8s.io/api/storage/v1"

	"github.com/percona/everest-operator/api/everest/v1alpha1"
)

func (out *BackupStorage) FromCR(in *v1alpha1.BackupStorage) {
	out.Type = BackupStorageType(in.Spec.Type)
	out.Name = in.GetName()
	out.Namespace = in.GetNamespace()
	out.Description = &in.Spec.Description
	out.BucketName = in.Spec.Bucket
	out.Region = in.Spec.Region
	out.Url = &in.Spec.EndpointURL
	out.VerifyTLS = in.Spec.VerifyTLS
	out.ForcePathStyle = in.Spec.ForcePathStyle
}

func (out *MonitoringInstance) FromCR(in *v1alpha1.MonitoringConfig) {
	out.Name = in.GetName()
	out.Namespace = in.GetNamespace()
	out.Url = in.Spec.PMM.URL
	out.AllowedNamespaces = &in.Spec.AllowedNamespaces
	out.VerifyTLS = in.Spec.VerifyTLS
	out.Type = MonitoringInstanceBaseWithNameType(in.Spec.Type)
}

func (out *StorageClass) FromCR(in *v1.StorageClass) {
	meta := make(map[string]interface{})
	meta["name"] = in.GetName()
	meta["annotations"] = in.GetAnnotations()
	meta["labels"] = in.GetLabels()
	out.Metadata = &meta
	out.AllowVolumeExpansion = pointer.To(pointer.Get(in.AllowVolumeExpansion))
}
