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

package inventory

import (
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

const (
	// SummarySchemaVersion is the version of the Summary document. Fields are
	// only ever added, so it changes only on a breaking change.
	SummarySchemaVersion = 1

	// CustomProvider is the key under which Summary.InstancesByProvider
	// counts the instances of providers that are not in the official catalog.
	CustomProvider = "custom"
)

// Summary is what telemetry may know about one cluster: counts, sums and the
// names of official catalog extensions, never a name a user chose.
//
// It is also the document a future multicluster agent reports to the main
// installation, whatever the transport. Fields are only ever added: a field an
// older agent doesn't send decodes as zero.
type Summary struct {
	SchemaVersion int `json:"schemaVersion"`
	// ClusterID is the UID of the cluster's kube-system namespace. It is used
	// to count each cluster once and is never sent.
	ClusterID   string    `json:"clusterId"`
	CollectedAt time.Time `json:"collectedAt"`

	// ProviderInstallations counts Provider objects, official or not.
	ProviderInstallations int `json:"providerInstallations"`
	// Providers lists the official providers installed, sorted.
	Providers []string `json:"providers,omitempty"`
	// InstancesByProvider counts Instances per official provider; the others
	// are counted under CustomProvider.
	InstancesByProvider map[string]int `json:"instancesByProvider,omitempty"`
	// PluginInstallations counts Plugin objects, official or not.
	PluginInstallations int `json:"pluginInstallations"`
	// Plugins lists the official plugins installed, sorted.
	Plugins []string `json:"plugins,omitempty"`

	Instances               int `json:"instances"`
	InstancesReady          int `json:"instancesReady"`
	InstancesStalled        int `json:"instancesStalled"`
	NamespacesWithInstances int `json:"namespacesWithInstances"`

	// The resources the Instance specs request, see InstanceResources.
	CPURequestedCores     float64 `json:"cpuRequestedCores"`
	MemoryRequestedBytes  float64 `json:"memoryRequestedBytes"`
	StorageRequestedBytes float64 `json:"storageRequestedBytes"`

	BackupsSucceeded           int `json:"backupsSucceeded"`
	BackupsFailed              int `json:"backupsFailed"`
	Restores                   int `json:"restores"`
	InstancesWithBackupStorage int `json:"instancesWithBackupStorage"`
	InstancesWithPITR          int `json:"instancesWithPitr"`

	MonitoringClasses      int `json:"monitoringClasses"`
	MonitoringDestinations int `json:"monitoringDestinations"`
	MonitoringBindings     int `json:"monitoringBindings"`
	BackupClasses          int `json:"backupClasses"`
	BackupStorageClasses   int `json:"backupStorageClasses"`
	BackupStorages         int `json:"backupStorages"`
}

// Summarize reduces a Snapshot of the cluster clusterID to a Summary.
func Summarize(s *Snapshot, clusterID string, now time.Time) Summary {
	sum := Summary{
		SchemaVersion:          SummarySchemaVersion,
		ClusterID:              clusterID,
		CollectedAt:            now,
		ProviderInstallations:  len(s.Providers),
		PluginInstallations:    len(s.Plugins),
		Instances:              len(s.Instances),
		Restores:               len(s.Restores),
		MonitoringClasses:      len(s.MonitoringClasses),
		MonitoringDestinations: len(s.MonitoringDestinations),
		MonitoringBindings:     len(s.MonitoringBindings),
		BackupClasses:          len(s.BackupClasses),
		BackupStorageClasses:   len(s.BackupStorageClasses),
		BackupStorages:         len(s.BackupStorages),
	}

	providers := sets.New[string]()
	for i := range s.Providers {
		if officialProviders.Has(s.Providers[i].Name) {
			providers.Insert(s.Providers[i].Name)
		}
	}
	sum.Providers = sets.List(providers)

	plugins := sets.New[string]()
	for i := range s.Plugins {
		if officialPlugins.Has(s.Plugins[i].Name) {
			plugins.Insert(s.Plugins[i].Name)
		}
	}
	sum.Plugins = sets.List(plugins)

	summarizeInstances(&sum, s.Instances)

	for i := range s.Backups {
		switch s.Backups[i].Status.State {
		case backupv1alpha1.BackupStateSucceeded:
			sum.BackupsSucceeded++
		case backupv1alpha1.BackupStateFailed, backupv1alpha1.BackupStateError:
			sum.BackupsFailed++
		case backupv1alpha1.BackupStatePending, backupv1alpha1.BackupStateRunning, backupv1alpha1.BackupStateDeleting:
		}
	}
	return sum
}

func summarizeInstances(sum *Summary, instances []corev1alpha1.Instance) {
	namespaces := sets.New[string]()
	for i := range instances {
		inst := &instances[i]
		namespaces.Insert(inst.Namespace)

		provider := inst.Spec.ProviderRef.Name
		if !officialProviders.Has(provider) {
			provider = CustomProvider
		}
		if sum.InstancesByProvider == nil {
			sum.InstancesByProvider = make(map[string]int)
		}
		sum.InstancesByProvider[provider]++

		if ConditionIs(inst.Status.Conditions, corev1alpha1.ConditionReady, metav1.ConditionTrue) {
			sum.InstancesReady++
		}
		if ConditionIs(inst.Status.Conditions, corev1alpha1.ConditionStalled, metav1.ConditionTrue) {
			sum.InstancesStalled++
		}
		if inst.Spec.Backup != nil && len(inst.Spec.Backup.Storages) > 0 {
			sum.InstancesWithBackupStorage++
		}
		if hasPITR(inst) {
			sum.InstancesWithPITR++
		}
		for _, r := range InstanceResources(inst) {
			sum.CPURequestedCores += r.Requests[corev1.ResourceCPU]
			sum.MemoryRequestedBytes += r.Requests[corev1.ResourceMemory]
			sum.StorageRequestedBytes += r.Requests[corev1.ResourceStorage]
		}
	}
	sum.NamespacesWithInstances = namespaces.Len()
}

// hasPITR reports whether point-in-time recovery is enabled on any of the
// Instance's backup storages: the provider reports a PITR status only then.
func hasPITR(inst *corev1alpha1.Instance) bool {
	if inst.Status.Backup == nil {
		return false
	}
	for _, st := range inst.Status.Backup.Storages {
		if st.PITR != nil {
			return true
		}
	}
	return false
}

// Add adds the counts and sums of o to s and unions the name sets. The
// identity fields of s are left as they are.
func (s *Summary) Add(o Summary) {
	s.ProviderInstallations += o.ProviderInstallations
	s.Providers = union(s.Providers, o.Providers)
	for p, n := range o.InstancesByProvider {
		if s.InstancesByProvider == nil {
			s.InstancesByProvider = make(map[string]int)
		}
		s.InstancesByProvider[p] += n
	}
	s.PluginInstallations += o.PluginInstallations
	s.Plugins = union(s.Plugins, o.Plugins)

	s.Instances += o.Instances
	s.InstancesReady += o.InstancesReady
	s.InstancesStalled += o.InstancesStalled
	s.NamespacesWithInstances += o.NamespacesWithInstances

	s.CPURequestedCores += o.CPURequestedCores
	s.MemoryRequestedBytes += o.MemoryRequestedBytes
	s.StorageRequestedBytes += o.StorageRequestedBytes

	s.BackupsSucceeded += o.BackupsSucceeded
	s.BackupsFailed += o.BackupsFailed
	s.Restores += o.Restores
	s.InstancesWithBackupStorage += o.InstancesWithBackupStorage
	s.InstancesWithPITR += o.InstancesWithPITR

	s.MonitoringClasses += o.MonitoringClasses
	s.MonitoringDestinations += o.MonitoringDestinations
	s.MonitoringBindings += o.MonitoringBindings
	s.BackupClasses += o.BackupClasses
	s.BackupStorageClasses += o.BackupStorageClasses
	s.BackupStorages += o.BackupStorages
}

func union(a, b []string) []string {
	return sets.List(sets.New(a...).Insert(b...))
}

// Fleet is the sum of the summaries of every cluster an installation manages.
type Fleet struct {
	Summary

	// Clusters counts the clusters whose summary is in Summary.
	Clusters int
	// StaleClusters counts the clusters whose latest summary was too old to
	// be added.
	StaleClusters int
}

// Merge adds up the summaries of a fleet. A cluster reported more than once,
// for example through two sources, counts once with its newest summary; a
// summary collected more than maxAge before now is left out and counted in
// StaleClusters, so an unreachable cluster is not counted with old numbers.
func Merge(summaries []Summary, now time.Time, maxAge time.Duration) Fleet {
	newest := make(map[string]Summary)
	for _, s := range summaries {
		if cur, ok := newest[s.ClusterID]; !ok || s.CollectedAt.After(cur.CollectedAt) {
			newest[s.ClusterID] = s
		}
	}
	ids := make([]string, 0, len(newest))
	for id := range newest {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var f Fleet
	for _, id := range ids {
		s := newest[id]
		if now.Sub(s.CollectedAt) > maxAge {
			f.StaleClusters++
			continue
		}
		f.Clusters++
		f.Add(s)
	}
	return f
}
