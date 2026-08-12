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

package minio

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// This file hand-declares the subset of the MinIO Operator's Tenant CRD
// (group minio.min.io, version v1) that this provider needs to read and
// write. It intentionally does not import github.com/minio/operator: that
// module predates Go's module graph pruning (go 1.13), so importing even
// its pkg/apis subpackage drags in its full, unpruned dependency graph —
// the minio server itself, docker/cli, gopsutil, and more — none of which
// this provider needs. Field names, JSON tags, and the group/version/kind
// below are taken verbatim from the real CRD so the objects this provider
// creates are indistinguishable, on the wire, from ones built with the
// upstream types; fields this provider doesn't set or read are simply
// omitted, which is safe because they're all optional on the real CRD.

// tenantGroupVersion is the real MinIO Operator Tenant CRD's group/version.
// v2, not v1: verified directly against the operator installed in Phase 0
// (`kubectl get crd tenants.minio.min.io` only serves v2 — v1 existed in
// older operator releases but isn't what's deployed here).
var tenantGroupVersion = schema.GroupVersion{Group: "minio.min.io", Version: "v2"} //nolint:gochecknoglobals // schema.GroupVersion is a value type used as a constant; same pattern as resource.MustParse elsewhere in this repo

// AddToScheme registers the Tenant/TenantList types with a scheme, mirroring
// github.com/minio/operator/pkg/apis/minio.min.io/v1.AddToScheme.
func AddToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(tenantGroupVersion, &Tenant{}, &TenantList{})
	metav1.AddToGroupVersion(scheme, tenantGroupVersion)
	return nil
}

// Tenant is the subset of the MinIO Operator's Tenant resource this
// provider uses.
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Spec   TenantSpec   `json:"spec"`
	Status TenantStatus `json:"status,omitzero"`
}

// TenantSpec is the subset of the real TenantSpec this provider sets.
type TenantSpec struct {
	Pools           []Pool                       `json:"pools"`
	Image           string                       `json:"image,omitempty"`
	RequestAutoCert *bool                        `json:"requestAutoCert,omitempty"`
	Configuration   *corev1.LocalObjectReference `json:"configuration,omitempty"`
	Buckets         []Bucket                     `json:"buckets,omitempty"`
}

// Bucket is the subset of the real Bucket (Tenant.spec.buckets[]) this
// provider sets: just a name, auto-created by the operator at reconcile
// time. Used for the Phase 3 backup-bridge bucket.
type Bucket struct {
	Name string `json:"name,omitempty"`
}

// Pool is the subset of the real Pool this provider sets.
type Pool struct {
	Name                string                        `json:"name,omitempty"`
	Servers             int32                         `json:"servers"`
	VolumesPerServer    int32                         `json:"volumesPerServer"`
	VolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"volumeClaimTemplate"`
}

// TenantStatus is the subset of the real TenantStatus this provider reads.
type TenantStatus struct {
	CurrentState      string `json:"currentState"`
	AvailableReplicas int32  `json:"availableReplicas"`
}

// TenantList is the list type paired with Tenant for scheme registration.
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`

	Items []Tenant `json:"items"`
}

// DeepCopyObject implements runtime.Object.
func (t *Tenant) DeepCopyObject() runtime.Object { //nolint:ireturn // mandated by the runtime.Object interface
	if t == nil {
		return nil
	}
	out := new(Tenant)
	out.TypeMeta = t.TypeMeta
	t.DeepCopyInto(&out.ObjectMeta)
	out.Status = t.Status
	out.Spec.Image = t.Spec.Image
	if t.Spec.RequestAutoCert != nil {
		v := *t.Spec.RequestAutoCert
		out.Spec.RequestAutoCert = &v
	}
	if t.Spec.Configuration != nil {
		v := *t.Spec.Configuration
		out.Spec.Configuration = &v
	}
	if t.Spec.Buckets != nil {
		out.Spec.Buckets = make([]Bucket, len(t.Spec.Buckets))
		copy(out.Spec.Buckets, t.Spec.Buckets)
	}
	if t.Spec.Pools != nil {
		out.Spec.Pools = make([]Pool, len(t.Spec.Pools))
		for i, p := range t.Spec.Pools {
			out.Spec.Pools[i] = Pool{
				Name:             p.Name,
				Servers:          p.Servers,
				VolumesPerServer: p.VolumesPerServer,
			}
			if p.VolumeClaimTemplate != nil {
				out.Spec.Pools[i].VolumeClaimTemplate = p.VolumeClaimTemplate.DeepCopy()
			}
		}
	}
	return out
}

// DeepCopyObject implements runtime.Object.
func (l *TenantList) DeepCopyObject() runtime.Object { //nolint:ireturn // mandated by the runtime.Object interface
	if l == nil {
		return nil
	}
	out := new(TenantList)
	out.TypeMeta = l.TypeMeta
	l.DeepCopyInto(&out.ListMeta)
	if l.Items != nil {
		out.Items = make([]Tenant, len(l.Items))
		for i, t := range l.Items {
			if copied, ok := t.DeepCopyObject().(*Tenant); ok {
				out.Items[i] = *copied
			}
		}
	}
	return out
}
