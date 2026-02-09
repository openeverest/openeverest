// Package rbac provides the RBAC handler.
package rbac

import (
	"context"
	"fmt"

	"github.com/AlekSi/pointer"

	"github.com/openeverest/openeverest/v2/api"
)

func (h *rbacHandler) GetKubernetesClusterResources(ctx context.Context) (*api.KubernetesClusterResources, error) {
	return h.next.GetKubernetesClusterResources(ctx)
}

func (h *rbacHandler) GetKubernetesClusterInfo(ctx context.Context) (*api.KubernetesClusterInfo, error) {
	return h.next.GetKubernetesClusterInfo(ctx)
}

func (h *rbacHandler) GetSettings(ctx context.Context) (*api.Settings, error) {
	return h.next.GetSettings(ctx)
}

func (h *rbacHandler) GetUserPermissions(ctx context.Context) (*api.UserPermissions, error) {
	user, err := h.userGetter(ctx)
	if err != nil {
		return nil, err
	}

	// Let's use a map to deduplicate the permissions after resolving all roles
	permsMap := make(map[[4]string]struct{})

	// Get permissions for the user and the groups it belongs to
	for _, sub := range append([]string{user.Subject}, user.Groups...) {
		perms, err := h.enforcer.GetImplicitPermissionsForUser(sub)
		if err != nil {
			return nil, fmt.Errorf("failed to GetImplicitPermissionsForUser: %w", err)
		}

		// GetImplicitPermissionsForUser returns all policies assigned to the
		// user/group directly as well as the policies assigned to the roles
		// the user/group has. We need to resolve all roles for the user.
		for _, perm := range perms {
			if len(perm) != 4 {
				// This should never happen, but let's be safe
				return nil, fmt.Errorf("invalid permission")
			}

			// We don't want to expose the groups or roles in the permissions
			// so we replace them with the user
			permsMap[[4]string{user.Subject, perm[1], perm[2], perm[3]}] = struct{}{}
		}
	}

	// Convert the map back to a slice
	result := make([][]string, len(permsMap))
	i := 0
	for k := range permsMap {
		result[i] = []string(k[:])
		i++
	}

	nextRes, err := h.next.GetUserPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to GetUserPermissions: %w", err)
	}

	enabled := nextRes.Enabled
	if !enabled {
		result = nil
	}

	res := &api.UserPermissions{
		Permissions: pointer.To(result),
		Enabled:     enabled,
	}
	return res, nil
}
