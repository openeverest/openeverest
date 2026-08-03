// everest
// Copyright (C) 2023 Percona LLC
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

package rbac

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"

	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
)

// ErrPolicySyntax is returned when a policy has a syntax error.
var errPolicySyntax = errors.New("policy syntax error")

func validatePolicy(enforcer *casbin.Enforcer) error {
	// check basic policy syntax.
	policy, err := enforcer.GetPolicy()
	if err != nil {
		return err
	}
	for _, policy := range policy {
		if err := validateTerms(policy); err != nil {
			return errors.Join(errPolicySyntax, err)
		}
	}

	// ensure that non-existent roles are not used.
	roles, err := enforcer.GetAllRoles()
	if err != nil {
		return err
	}
	if err := checkRoles(roles, policy); err != nil {
		return errors.Join(errPolicySyntax, err)
	}

	// ensure that non-existent resources are not used.
	if err := checkResourceNames(policy); err != nil {
		return errors.Join(errPolicySyntax, err)
	}
	return nil
}

// ValidatePolicy validates a policy from either Kubernetes or local file.
func ValidatePolicy(
	ctx context.Context,
	k kubernetes.KubernetesConnector,
	filepath string,
) error {
	enforcer, err := newKubeOrFileEnforcer(ctx, k, filepath)
	if err != nil {
		return errors.Join(errPolicySyntax, err)
	}
	return validatePolicy(enforcer)
}

func checkResourceNames(policies [][]string) error {
	resourcePathMap, _, err := buildPathResourceMap("")
	if err != nil {
		return fmt.Errorf("failed to get resource path map: %w", err)
	}
	knownResources := make(map[string]struct{})
	for _, resource := range resourcePathMap {
		knownResources[resource] = struct{}{}
	}
	for _, policy := range policies {
		resourceName := policy[1]
		if resourceName == "*" {
			continue
		}
		if _, ok := knownResources[resourceName]; !ok {
			return fmt.Errorf("unknown resource name '%s'", resourceName)
		}
	}
	return nil
}

func checkRoles(roles []string, policies [][]string) error {
	for _, policy := range policies {
		roleName := policy[0]
		if !strings.HasPrefix(roleName, common.EverestRBACRolePrefix) {
			continue
		}
		if roleName == common.EverestAdminRole {
			// Its fine to not assign the admin role to any user.
			continue
		}
		if !slices.Contains(roles, roleName) {
			return fmt.Errorf("role '%s' does not exist", roleName)
		}
	}
	return nil
}

func validateTerms(terms []string) error {
	if len(terms) != 4 {
		return fmt.Errorf("expected 4 policy terms [sub, res, act, obj], got %d", len(terms))
	}

	subject, resource, action, object := terms[0], terms[1], terms[2], terms[3]

	if err := validateSubject(subject); err != nil {
		return fmt.Errorf("invalid subject '%s': %w", subject, err)
	}

	if strings.TrimSpace(resource) == "" {
		return errors.New("empty resource type")
	}

	if !ValidateAction(action) {
		return fmt.Errorf("invalid action '%s'", action)
	}

	if err := validateObject(object); err != nil {
		return fmt.Errorf("invalid object pattern '%s': %w", object, err)
	}

	return nil
}

func validateSubject(subject string) error {
	if subject == "" {
		return errors.New("empty subject")
	}
	if strings.HasPrefix(subject, common.EverestRBACRolePrefix) {
		roleName := strings.TrimPrefix(subject, common.EverestRBACRolePrefix)
		if strings.TrimSpace(roleName) == "" {
			return errors.New("empty role name after prefix")
		}
	}
	for _, r := range subject {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return errors.New("contains whitespace or control characters")
		}
	}
	return nil
}

func validateObject(object string) error {
	if object == "" {
		return errors.New("empty object pattern")
	}
	if !doublestar.ValidatePattern(object) {
		return errors.New("malformed glob pattern")
	}
	segments := strings.Split(object, "/")
	if len(segments) > 2 {
		return errors.New("object pattern contains more than two segments")
	}
	return nil
}

//nolint:nonamedreturns
func newKubeOrFileEnforcer(
	ctx context.Context,
	kubeClient kubernetes.KubernetesConnector,
	filePath string,
) (e *casbin.Enforcer, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot create enforcer: %v", r)
			e = nil
		}
	}()
	if filePath != "" {
		return NewEnforcerFromFilePath(filePath)
	}
	return NewEnforcer(ctx, kubeClient, zap.NewNop().Sugar())
}
