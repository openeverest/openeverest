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
	"fmt"
	"strings"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

// Denied API groups — plugins must not touch these.
var deniedAPIGroups = map[string]bool{
	"everest.percona.com":          true,
	"rbac.authorization.k8s.io":    true,
	"admissionregistration.k8s.io": true,
	"apiextensions.k8s.io":         true,
	"certificates.k8s.io":          true,
}

// Denied core ("") resources — sensitive resources plugins must not access.
var deniedCoreResources = map[string]bool{
	"secrets":           true,
	"nodes":             true,
	"persistentvolumes": true,
	"namespaces":        true,
	"serviceaccounts":   true,
}

// validateKubePermissions checks that the declared kubePermissions do not
// violate the hard-coded denylist. Returns a list of human-readable violation
// messages. An empty slice means the rules are valid.
func validateKubePermissions(rules []pluginv1alpha1.KubePermissionRule) []string {
	var violations []string

	for i, rule := range rules {
		prefix := fmt.Sprintf("kubePermissions[%d]", i)

		// Reject wildcards.
		for _, g := range rule.APIGroups {
			if g == "*" {
				violations = append(violations, prefix+": wildcard apiGroup \"*\" is not allowed")
			}
		}
		for _, r := range rule.Resources {
			if r == "*" {
				violations = append(violations, prefix+": wildcard resource \"*\" is not allowed")
			}
		}
		for _, v := range rule.Verbs {
			if v == "*" {
				violations = append(violations, prefix+": wildcard verb \"*\" is not allowed")
			}
		}

		// Reject denied API groups.
		for _, g := range rule.APIGroups {
			if deniedAPIGroups[g] {
				violations = append(violations, fmt.Sprintf("%s: apiGroup %q is denied", prefix, g))
			}
		}

		// Reject denied core resources.
		for _, g := range rule.APIGroups {
			if g == "" { // core group
				for _, r := range rule.Resources {
					if deniedCoreResources[strings.ToLower(r)] {
						violations = append(violations, fmt.Sprintf("%s: core resource %q is denied", prefix, r))
					}
				}
			}
		}
	}

	return violations
}
