package rbac

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
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
	knownResources := make(map[string]struct{})
	for _, resource := range AllResources {
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
	pattern := `^[/*-_:a-zA-Z0-9]+$`
	compiled := regexp.MustCompile(pattern)
	for _, term := range terms {
		if !compiled.MatchString(term) {
			return fmt.Errorf("invalid policy term '%s'", term)
		}
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
