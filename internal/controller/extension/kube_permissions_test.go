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

package extension

import (
	"testing"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

func TestValidateKubePermissions_Valid(t *testing.T) {
	rules := []pluginv1alpha1.KubePermissionRule{
		{
			APIGroups: []string{"apps"},
			Resources: []string{"deployments"},
			Verbs:     []string{"get", "list", "create", "update", "delete"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"services", "configmaps"},
			Verbs:     []string{"get", "list", "watch", "create"},
		},
	}

	violations := validateKubePermissions(rules)
	if len(violations) != 0 {
		t.Errorf("expected no violations, got: %v", violations)
	}
}

func TestValidateKubePermissions_WildcardAPIGroup(t *testing.T) {
	rules := []pluginv1alpha1.KubePermissionRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"deployments"},
			Verbs:     []string{"get"},
		},
	}

	violations := validateKubePermissions(rules)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
}

func TestValidateKubePermissions_WildcardResource(t *testing.T) {
	rules := []pluginv1alpha1.KubePermissionRule{
		{
			APIGroups: []string{"apps"},
			Resources: []string{"*"},
			Verbs:     []string{"get"},
		},
	}

	violations := validateKubePermissions(rules)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
}

func TestValidateKubePermissions_WildcardVerb(t *testing.T) {
	rules := []pluginv1alpha1.KubePermissionRule{
		{
			APIGroups: []string{"apps"},
			Resources: []string{"deployments"},
			Verbs:     []string{"*"},
		},
	}

	violations := validateKubePermissions(rules)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
}

func TestValidateKubePermissions_DeniedAPIGroup(t *testing.T) {
	rules := []pluginv1alpha1.KubePermissionRule{
		{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"roles"},
			Verbs:     []string{"create"},
		},
	}

	violations := validateKubePermissions(rules)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
}

func TestValidateKubePermissions_DeniedCoreResource(t *testing.T) {
	rules := []pluginv1alpha1.KubePermissionRule{
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get"},
		},
	}

	violations := validateKubePermissions(rules)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
}

func TestValidateKubePermissions_MultipleViolations(t *testing.T) {
	rules := []pluginv1alpha1.KubePermissionRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{"everest.percona.com"},
			Resources: []string{"databaseclusters"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"nodes", "secrets"},
			Verbs:     []string{"list"},
		},
	}

	violations := validateKubePermissions(rules)
	// Rule 0: wildcard apiGroup + wildcard resource + wildcard verb = 3 violations
	// Rule 1: denied apiGroup = 1 violation
	// Rule 2: denied core resources (nodes + secrets) = 2 violations
	// Total = 6
	if len(violations) != 6 {
		t.Errorf("expected 6 violations, got %d: %v", len(violations), violations)
	}
}

func TestValidateKubePermissions_EmptyRules(t *testing.T) {
	violations := validateKubePermissions(nil)
	if len(violations) != 0 {
		t.Errorf("expected no violations for nil input, got: %v", violations)
	}

	violations = validateKubePermissions([]pluginv1alpha1.KubePermissionRule{})
	if len(violations) != 0 {
		t.Errorf("expected no violations for empty input, got: %v", violations)
	}
}
