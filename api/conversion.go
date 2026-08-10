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

package api

import (
	"github.com/AlekSi/pointer"
	v1 "k8s.io/api/storage/v1"

	"github.com/percona/everest-operator/api/everest/v1alpha1"
)

// FromCR populates the BackupStorage fields from the given BackupStorage custom resource.
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

// FromCR populates the MonitoringInstance fields from the given MonitoringConfig custom resource.
func (out *MonitoringInstance) FromCR(in *v1alpha1.MonitoringConfig) {
	out.Name = in.GetName()
	out.Namespace = in.GetNamespace()
	out.Url = in.Spec.PMM.URL
	out.AllowedNamespaces = &in.Spec.AllowedNamespaces
	out.VerifyTLS = in.Spec.VerifyTLS
	out.Type = MonitoringInstanceBaseWithNameType(in.Spec.Type)
}

// FromCR populates the StorageClass fields from the given StorageClass Kubernetes resource.
func (out *StorageClass) FromCR(in *v1.StorageClass) {
	meta := make(map[string]any)
	meta["name"] = in.GetName()
	meta["annotations"] = in.GetAnnotations()
	meta["labels"] = in.GetLabels()
	out.Metadata = &meta
	out.AllowVolumeExpansion = new(pointer.Get(in.AllowVolumeExpansion))
}
