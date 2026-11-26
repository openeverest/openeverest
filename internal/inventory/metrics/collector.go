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

// Package metrics exposes the OpenEverest inventory as Prometheus metrics.
//
// The collector builds every series at scrape time from an inventory
// Snapshot, the way kube-state-metrics does: no reconciler keeps gauges up to
// date, and the series of a deleted object disappear on the next scrape.
package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/internal/inventory"
	"github.com/openeverest/openeverest/v2/pkg/version"
)

const collectTimeout = 10 * time.Second

// Collector is a prometheus.Collector over the inventory of one cluster.
type Collector struct {
	reader   client.Reader
	isLeader func() bool
	log      logr.Logger
}

// NewCollector returns a Collector that reads the inventory from reader.
//
// Every controller replica shares the same cache, so only the replica for
// which isLeader returns true emits the inventory; the others emit only
// openeverest_build_info. Scraping every replica then yields one copy of each
// series.
func NewCollector(reader client.Reader, isLeader func() bool, log logr.Logger) *Collector {
	return &Collector{reader: reader, isLeader: isLeader, log: log}
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range allDescs {
		ch <- d
	}
}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	e := newEmitter(ch)
	e.gauge(buildInfo, 1, version.Version, contract.Version)
	if !c.isLeader() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), collectTimeout)
	defer cancel()
	snap, err := inventory.Collect(ctx, c.reader)
	if err != nil {
		c.log.Error(err, "Failed to collect the inventory for /metrics")
		return
	}
	emitSnapshot(e, snap)
}

func emitSnapshot(e *emitter, s *inventory.Snapshot) {
	emitExtensions(e, s.InstalledExtensions)
	emitProviders(e, s.Providers)
	emitClasses(e, s)
	emitBackupStorages(e, s.BackupStorages)
	emitMonitoringDestinations(e, s.MonitoringDestinations)
	emitInstances(e, s.Instances)
	emitBackups(e, s.Backups)
	emitRestores(e, s.Restores)
}

func emitExtensions(e *emitter, extensions []extensionsv1alpha1.InstalledExtension) {
	for i := range extensions {
		ie := &extensions[i]
		typ := string(ie.Spec.Type)
		e.gauge(extensionInfo, 1, typ, ie.Name, ie.Spec.Version)
		e.oneHot(extensionPhase, string(ie.Status.Phase), extensionPhases, typ, ie.Name)
	}
}

func emitProviders(e *emitter, providers []corev1alpha1.Provider) {
	for i := range providers {
		p := &providers[i]
		e.gauge(providerInfo, 1, p.Name, p.Status.ContractVersion)
		for _, cond := range p.Status.Conditions {
			e.conditionStatus(providerCondition, cond, p.Name, cond.Type)
		}
	}
}

func emitClasses(e *emitter, s *inventory.Snapshot) {
	const (
		kindMonitoringClass    = "MonitoringClass"
		kindBackupClass        = "BackupClass"
		kindBackupStorageClass = "BackupStorageClass"
	)
	for i := range s.MonitoringClasses {
		mc := &s.MonitoringClasses[i]
		e.gauge(classInfo, 1, kindMonitoringClass, mc.Name, mc.Spec.ControllerName)
		emitAccepted(e, kindMonitoringClass, mc.Name, mc.Status.Conditions, monitoringv1alpha1.MonitoringClassConditionAccepted)
	}
	for i := range s.BackupClasses {
		bc := &s.BackupClasses[i]
		controller := ""
		if bc.Spec.ProviderManaged != nil {
			controller = bc.Spec.ProviderManaged.ProviderRef.Name
		}
		e.gauge(classInfo, 1, kindBackupClass, bc.Name, controller)
		emitAccepted(e, kindBackupClass, bc.Name, bc.Status.Conditions, backupv1alpha1.BackupClassConditionAccepted)
	}
	for i := range s.BackupStorageClasses {
		e.gauge(classInfo, 1, kindBackupStorageClass, s.BackupStorageClasses[i].Name, "")
	}
}

func emitAccepted(e *emitter, kind, name string, conditions []metav1.Condition, conditionType string) {
	for _, cond := range conditions {
		if cond.Type == conditionType {
			e.conditionStatus(classAccepted, cond, kind, name)
			return
		}
	}
}

func emitBackupStorages(e *emitter, storages []backupv1alpha1.BackupStorage) {
	for i := range storages {
		bs := &storages[i]
		e.gauge(backupStorageInfo, 1, bs.Namespace, bs.Name, bs.Spec.BackupStorageClassRef.Name)
	}
}

func emitMonitoringDestinations(e *emitter, destinations []monitoringv1alpha1.MonitoringDestination) {
	for i := range destinations {
		md := &destinations[i]
		e.gauge(monitoringDestinationInfo, 1, md.Namespace, md.Name, md.Spec.ClassRef.Name)
		for _, cond := range md.Status.Conditions {
			if cond.Type == monitoringv1alpha1.MonitoringDestinationConditionReady {
				e.conditionStatus(monitoringDestinationReady, cond, md.Namespace, md.Name)
				break
			}
		}
	}
}

func emitInstances(e *emitter, instances []corev1alpha1.Instance) {
	for i := range instances {
		inst := &instances[i]
		ns, name := inst.Namespace, inst.Name
		e.gauge(instanceInfo, 1, ns, name, inst.Spec.ProviderRef.Name, inst.Spec.Version)
		e.oneHot(instancePhase, string(inst.Status.Phase), instancePhases, ns, name)
		for _, cond := range inst.Status.Conditions {
			if instanceConditions[cond.Type] {
				e.conditionStatus(instanceCondition, cond, ns, name, cond.Type)
			}
		}
		emitComponentReplicas(e, inst)
		emitInstanceResources(e, inst)
		emitPITR(e, inst)
	}
}

func emitComponentReplicas(e *emitter, inst *corev1alpha1.Instance) {
	for _, c := range inst.Status.Components {
		for kind, replicas := range map[string]*int32{
			"desired": c.DesiredReplicas,
			"current": c.Replicas,
			"ready":   c.ReadyReplicas,
		} {
			if replicas != nil {
				e.gauge(instanceComponentReplicas, float64(*replicas), inst.Namespace, inst.Name, c.Name, kind)
			}
		}
	}
}

func emitInstanceResources(e *emitter, inst *corev1alpha1.Instance) {
	for _, r := range inventory.InstanceResources(inst) {
		for res, v := range r.Requests {
			e.gauge(instanceResourceRequests, v, inst.Namespace, inst.Name, r.Component, string(res))
		}
		for res, v := range r.Limits {
			e.gauge(instanceResourceLimits, v, inst.Namespace, inst.Name, r.Component, string(res))
		}
	}
}

func emitPITR(e *emitter, inst *corev1alpha1.Instance) {
	if inst.Status.Backup == nil {
		return
	}
	for _, st := range inst.Status.Backup.Storages {
		if st.PITR == nil {
			continue
		}
		available := 0.0
		if st.PITR.State == corev1alpha1.PITRStateAvailable {
			available = 1
		}
		e.gauge(instancePITRAvailable, available, inst.Namespace, inst.Name, st.Name)
	}
}

type backupKey struct{ namespace, instance, storage, state string }

type storageKey struct{ namespace, instance, storage string }

func emitBackups(e *emitter, backups []backupv1alpha1.Backup) {
	counts := map[backupKey]int{}
	lastSuccess := map[storageKey]time.Time{}
	for i := range backups {
		b := &backups[i]
		instance := ""
		if b.Spec.Origin.InstanceRef != nil {
			instance = b.Spec.Origin.InstanceRef.Name
		}
		storage := b.Spec.StorageRef.Name
		counts[backupKey{b.Namespace, instance, storage, stateOrUnknown(string(b.Status.State))}]++
		if b.Status.State != backupv1alpha1.BackupStateSucceeded || instance == "" {
			continue
		}
		completed := b.CreationTimestamp.Time
		if b.Status.CompletedAt != nil {
			completed = b.Status.CompletedAt.Time
		}
		k := storageKey{b.Namespace, instance, storage}
		if completed.After(lastSuccess[k]) {
			lastSuccess[k] = completed
		}
	}
	for k, n := range counts {
		e.gauge(backupsCount, float64(n), k.namespace, k.instance, k.storage, k.state)
	}
	for k, t := range lastSuccess {
		e.gauge(instanceLastSuccessfulBackup, float64(t.Unix()), k.namespace, k.instance, k.storage)
	}
}

type restoreKey struct{ namespace, instance, state string }

func emitRestores(e *emitter, restores []backupv1alpha1.Restore) {
	counts := map[restoreKey]int{}
	for i := range restores {
		r := &restores[i]
		counts[restoreKey{r.Namespace, r.Spec.InstanceRef.Name, stateOrUnknown(string(r.Status.State))}]++
	}
	for k, n := range counts {
		e.gauge(restoresCount, float64(n), k.namespace, k.instance, k.state)
	}
}

// stateOrUnknown keeps objects without a reported state from producing an
// empty label value, which Prometheus treats as a missing label.
func stateOrUnknown(state string) string {
	if state == "" {
		return "Unknown"
	}
	return state
}

// emitter sends const metrics and drops a series whose labels were already
// sent. A duplicate series fails the whole scrape, and an object with, say,
// a repeated condition type must not take /metrics down.
type emitter struct {
	ch   chan<- prometheus.Metric
	seen map[string]struct{}
}

func newEmitter(ch chan<- prometheus.Metric) *emitter {
	return &emitter{ch: ch, seen: map[string]struct{}{}}
}

func (e *emitter) gauge(desc *prometheus.Desc, value float64, labels ...string) {
	key := desc.String() + "\x00" + strings.Join(labels, "\x00")
	if _, ok := e.seen[key]; ok {
		return
	}
	e.seen[key] = struct{}{}
	e.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, labels...)
}

// oneHot emits one series per possible value, 1 for the current one.
func (e *emitter) oneHot(desc *prometheus.Desc, current string, values []string, labels ...string) {
	for _, v := range values {
		value := 0.0
		if v == current {
			value = 1
		}
		e.gauge(desc, value, append(labels, v)...)
	}
}

func (e *emitter) conditionStatus(desc *prometheus.Desc, cond metav1.Condition, labels ...string) {
	e.oneHot(desc, strings.ToLower(string(cond.Status)), conditionStatuses, labels...)
}
