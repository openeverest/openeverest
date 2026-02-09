package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func (h *rbacHandler) ListNamespaces(ctx context.Context) ([]string, error) {
	list, err := h.next.ListNamespaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ListNamespaces: %w", err)
	}
	result := make([]string, 0, len(list))
	for _, ns := range list {
		if err := h.enforce(ctx, rbac.ResourceNamespaces, rbac.ActionRead, ns); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce error: %w", err)
		}
		result = append(result, ns)
	}
	return result, nil
}
