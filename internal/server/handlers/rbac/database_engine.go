package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlekSi/pointer"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/pkg/rbac"
)

func (h *rbacHandler) ListDatabaseEngines(ctx context.Context, namespace string) (*everestv1alpha1.DatabaseEngineList, error) {
	list, err := h.next.ListDatabaseEngines(ctx, namespace)
	if err != nil {
		return nil, err
	}
	filtered := []everestv1alpha1.DatabaseEngine{}
	for _, dbengine := range list.Items {
		if err := h.enforce(ctx, rbac.ResourceDatabaseEngines, rbac.ActionRead, rbac.ObjectName(namespace, dbengine.GetName())); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, dbengine)
	}
	list.Items = filtered
	return list, nil
}

func (h *rbacHandler) GetDatabaseEngine(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseEngine, error) {
	if err := h.enforce(ctx, rbac.ResourceDatabaseEngines, rbac.ActionRead, rbac.ObjectName(namespace, name)); err != nil {
		return nil, err
	}
	return h.next.GetDatabaseEngine(ctx, namespace, name)
}

func (h *rbacHandler) UpdateDatabaseEngine(ctx context.Context, req *everestv1alpha1.DatabaseEngine) (*everestv1alpha1.DatabaseEngine, error) {
	if err := h.enforce(ctx, rbac.ResourceDatabaseEngines, rbac.ActionUpdate, rbac.ObjectName(req.GetNamespace(), req.GetName())); err != nil {
		return nil, err
	}
	return h.next.UpdateDatabaseEngine(ctx, req)
}

func (h *rbacHandler) GetUpgradePlan(ctx context.Context, namespace string) (*api.UpgradePlan, error) {
	// Need access to all DatabaseClusters to get the upgrade plan
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusters, rbac.ActionRead, rbac.ObjectName(namespace, "")); err != nil {
		return nil, err
	}
	result, err := h.next.GetUpgradePlan(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("GetUpgradePlan failed: %w", err)
	}
	for _, upg := range pointer.Get(result.Upgrades) {
		if err := h.enforce(ctx, rbac.ResourceDatabaseEngines, rbac.ActionRead, rbac.ObjectName(namespace, *upg.Name)); errors.Is(err, ErrInsufficientPermissions) {
			// We cannot show this plan, the user does not have permission to one or more engines.
			result = &api.UpgradePlan{}
			break
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
	}
	return result, nil
}

func (h *rbacHandler) ApproveUpgradePlan(ctx context.Context, namespace string) error {
	plan, err := h.GetUpgradePlan(ctx, namespace)
	if err != nil {
		return err
	}
	// Ensure we can update all these engines.
	for _, upg := range pointer.Get(plan.Upgrades) {
		if err := h.enforce(ctx, rbac.ResourceDatabaseEngines, rbac.ActionUpdate, rbac.ObjectName(namespace, *upg.Name)); err != nil {
			return err
		}
	}
	return h.next.ApproveUpgradePlan(ctx, namespace)
}
