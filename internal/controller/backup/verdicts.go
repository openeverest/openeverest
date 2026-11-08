// everest
// Copyright (C) 2023 Percona LLC
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

package backup

import (
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// maxVerdicts is the MaxItems of BackupClassStatus.providers and
// BackupStorageStatus.backupClasses.
const maxVerdicts = 64

// storageVerdict judges whether class can write to storage, whose Accepted
// condition is accepted. storageClass is the storage's BackupStorageClass, nil
// when it does not exist. The verdict records the objects it was computed
// from.
func storageVerdict(
	storage *backupv1alpha1.BackupStorage, accepted metav1.Condition,
	class *backupv1alpha1.BackupClass, storageClass *backupv1alpha1.BackupStorageClass,
) backupv1alpha1.BackupClassVerdict {
	v := backupv1alpha1.BackupClassVerdict{
		Name: class.Name,
		Observed: backupv1alpha1.BackupClassVerdictInputs{
			StorageGeneration: storage.Generation,
			ClassUID:          class.UID,
			ClassGeneration:   class.Generation,
		},
	}
	if storageClass != nil {
		v.Observed.StorageClassUID = storageClass.UID
		v.Observed.StorageClassGeneration = storageClass.Generation
	}
	if accepted.Status != metav1.ConditionTrue {
		v.Compatibility = backupv1alpha1.CompatibilityPending
		v.Reason = backupv1alpha1.ReasonStorageNotAccepted
		v.Message = fmt.Sprintf("BackupStorage %q is not Accepted (%s)", storage.Name, accepted.Reason)
		return v
	}
	c := class.CanWriteTo(storage)
	if !c.Compatible {
		v.Compatibility = backupv1alpha1.CompatibilityIncompatible
		v.Reason, v.Message = c.Reason, c.Message
		return v
	}
	v.Compatibility = backupv1alpha1.CompatibilityCompatible
	return v
}

// storageVerdicts judges storage against every class that is Accepted at its
// current generation, in name order, at most maxVerdicts of them.
func storageVerdicts(
	storage *backupv1alpha1.BackupStorage, accepted metav1.Condition,
	classes []backupv1alpha1.BackupClass, storageClass *backupv1alpha1.BackupStorageClass,
) []backupv1alpha1.BackupClassVerdict {
	out := make([]backupv1alpha1.BackupClassVerdict, 0, len(classes))
	for i := range classes {
		if classes[i].IsAccepted() {
			out = append(out, storageVerdict(storage, accepted, &classes[i], storageClass))
		}
	}
	slices.SortFunc(out, func(a, b backupv1alpha1.BackupClassVerdict) int { return strings.Compare(a.Name, b.Name) })
	if len(out) > maxVerdicts {
		out = out[:maxVerdicts]
	}
	return out
}

// terminalClaimReason reports whether an Accepted=False condition with reason
// can only be cleared by a change of the class.
func terminalClaimReason(reason string) bool {
	return reason == backupv1alpha1.ReasonUnknownClass || reason == backupv1alpha1.ReasonSpecDrift
}

// providerCompatibilities judges class for the Provider that executes it:
// one entry for a ProviderManaged class, none for a mode without an executor
// Provider. providerFound reports whether that Provider exists.
func providerCompatibilities(class *backupv1alpha1.BackupClass, providerFound bool) []backupv1alpha1.BackupClassProviderCompatibility {
	name := class.Spec.ExecutorProvider()
	if name == "" {
		return []backupv1alpha1.BackupClassProviderCompatibility{}
	}
	e := backupv1alpha1.BackupClassProviderCompatibility{Name: name, ClassGeneration: class.Generation}
	cond := meta.FindStatusCondition(class.Status.Conditions, backupv1alpha1.BackupClassConditionAccepted)
	current := cond != nil && cond.ObservedGeneration == class.Generation
	switch {
	case !providerFound:
		e.Compatibility = backupv1alpha1.CompatibilityPending
		e.Reason = backupv1alpha1.ReasonProviderNotFound
		e.Message = fmt.Sprintf("Provider %q does not exist", name)
	case current && cond.Status == metav1.ConditionTrue:
		e.Compatibility = backupv1alpha1.CompatibilityCompatible
	case current && cond.Status == metav1.ConditionFalse && terminalClaimReason(cond.Reason):
		e.Compatibility = backupv1alpha1.CompatibilityIncompatible
		e.Reason = cond.Reason
		e.Message = cond.Message
	default:
		e.Compatibility = backupv1alpha1.CompatibilityPending
		e.Reason = backupv1alpha1.ReasonClassNotAccepted
		e.Message = fmt.Sprintf("Provider %q has not accepted BackupClass %q at generation %d", name, class.Name, class.Generation)
		if cond != nil && cond.Reason != "" {
			e.Message += fmt.Sprintf(" (%s: %s)", cond.Reason, cond.Message)
		}
	}
	return []backupv1alpha1.BackupClassProviderCompatibility{e}
}
