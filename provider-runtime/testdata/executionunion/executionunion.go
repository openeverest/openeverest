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

// Package executionunion uses the execution union of a BackupClass in every
// way the invariant test of provider-runtime must report, and in ways it must
// not. The test analyses it with an allowlist of its own.
package executionunion

import (
	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

type (
	aliased  = backupv1alpha1.BackupClassSpec
	renamed  backupv1alpha1.BackupClassSpec
	embedded struct{ backupv1alpha1.BackupClassSpec }
)

type reconciler struct{ provider string }

// Claims is allowlisted for ExecutorProvider: it is the claim path.
func Claims(bc *backupv1alpha1.BackupClass, provider string) bool {
	return bc.Spec.ExecutorProvider() == provider
}

// Limits is allowlisted for ProviderManaged only, so its use of
// ExecutorProvider is reported.
func Limits(bc *backupv1alpha1.BackupClass, provider string) *backupv1alpha1.BackupClassLimits {
	if bc.Spec.ExecutorProvider() != provider || bc.Spec.ProviderManaged == nil {
		return nil
	}
	return bc.Spec.ProviderManaged.Limits
}

// Compatible decides compatibility from the discriminator.
func Compatible(bc *backupv1alpha1.BackupClass) bool {
	return bc.Spec.ExecutionMode == backupv1alpha1.BackupExecutionModeProviderManaged
}

func throughAlias(s *aliased) bool { return s.ProviderManaged != nil }

func throughRename(s *renamed) bool { return s.ExecutionMode != "" }

func throughEmbedding(e *embedded) string { return e.ExecutorProvider() }

func methodValue(s *backupv1alpha1.BackupClassSpec) func() string { return s.ExecutorProvider }

func throughIdentity(pm *backupv1alpha1.ProviderManagedSpec, provider string) bool {
	return pm.ProviderRef.Name == provider
}

func inClosure(bc *backupv1alpha1.BackupClass) func() bool {
	return func() bool { return bc.Spec.ProviderManaged != nil }
}

func (r *reconciler) owns(bc *backupv1alpha1.BackupClass) bool {
	return bc.Spec.ExecutorProvider() == r.provider
}

var methodExpression = (*backupv1alpha1.BackupClassSpec).ExecutorProvider

// Unrelated uses fields of the same names on other types, which say nothing
// about who executes a class.
func Unrelated(b *backupv1alpha1.Backup, in *corev1alpha1.Instance) (backupv1alpha1.BackupExecutionMode, string) {
	return b.Status.ExecutionMode, in.Spec.ProviderRef.Name
}
