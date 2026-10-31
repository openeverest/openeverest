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

package reconciler

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// judgedBackupStorages returns the verdict on every backup storage entry of
// an Instance with backups enabled, including the entries the provider
// rejected during Sync.
func (r *ProviderReconciler) judgedBackupStorages(c *controller.Context, in *v1alpha1.Instance) ([]controller.ResolvedBackupStorage, error) {
	if !r.isBackupProvider() || in.Spec.Backup == nil || !in.Spec.Backup.Enabled {
		return nil, nil
	}
	return c.BackupStorages()
}

// setBackupConfigured reports BackupConfigured: True when every storage entry
// is configured, otherwise False naming each entry that is not.
func setBackupConfigured(in *v1alpha1.Instance, storages []controller.ResolvedBackupStorage) {
	var failing []controller.ResolvedBackupStorage
	for _, s := range storages {
		if !s.Apply() {
			failing = append(failing, s)
		}
	}
	switch len(failing) {
	case 0:
		setCondition(in, v1alpha1.ConditionBackupConfigured, metav1.ConditionTrue,
			"Configured", "Backup configuration applied to engine", metav1.Now(),
		)
	case 1:
		setCondition(in, v1alpha1.ConditionBackupConfigured, metav1.ConditionFalse,
			failing[0].Reason, failing[0].Message, metav1.Now(),
		)
	default:
		messages := make([]string, 0, len(failing))
		for _, s := range failing {
			messages = append(messages, s.Reason+": "+s.Message)
		}
		setCondition(in, v1alpha1.ConditionBackupConfigured, metav1.ConditionFalse,
			v1alpha1.ReasonBackupStoragesUnusable, strings.Join(messages, "; "), metav1.Now(),
		)
	}
}

// setBackupStatus writes status.backup.storages: one entry per judged
// storage with its location, reason and message, and the PITR window the
// provider reported for it. An entry that is not configured keeps the
// location it was last configured with.
func setBackupStatus(in *v1alpha1.Instance, storages []controller.ResolvedBackupStorage, reported []v1alpha1.InstanceBackupStorageStatus) {
	previous := make(map[string]string)
	if in.Status.Backup != nil {
		for _, s := range in.Status.Backup.Storages {
			previous[s.Name] = s.Location
		}
	}
	pitr := make(map[string]*v1alpha1.InstanceBackupStoragePITRStatus, len(reported))
	for _, s := range reported {
		pitr[s.Name] = s.PITR
	}

	out := make([]v1alpha1.InstanceBackupStorageStatus, 0, max(len(storages), len(reported)))
	if len(storages) == 0 {
		out = append(out, reported...)
	}
	for _, s := range storages {
		st := v1alpha1.InstanceBackupStorageStatus{
			Name:     s.Name(),
			Location: previous[s.Name()],
			Reason:   s.Reason,
			Message:  s.Message,
			PITR:     pitr[s.Name()],
		}
		if s.Apply() {
			st.Location = s.Path
		}
		out = append(out, st)
	}
	if len(out) == 0 {
		in.Status.Backup = nil
		return
	}
	in.Status.Backup = &v1alpha1.InstanceBackupStatus{Storages: out}
}
