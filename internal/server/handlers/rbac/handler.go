// Package rbac provides the RBAC handler.
package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ErrInsufficientPermissions is returned when the user does not have sufficient permissions to perform the operation.
var ErrInsufficientPermissions = errors.New("insufficient permissions for performing the operation")

type rbacHandler struct {
	enforcer   casbin.IEnforcer
	log        *zap.SugaredLogger
	next       handlers.Handler
	userGetter func(ctx context.Context) (rbac.User, error)
}

// New returns a new RBAC handler.
//
//nolint:ireturn
func New(
	ctx context.Context,
	log *zap.SugaredLogger,
	kubeConnector kubernetes.KubernetesConnector,
) (handlers.Handler, error) {
	enf, err := rbac.NewEnforcerWithRefresh(ctx, kubeConnector, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}
	l := log.With("handler", "rbac")
	return &rbacHandler{
		enforcer:   enf,
		log:        l,
		userGetter: rbac.GetUser,
	}, nil
}

// SetNext sets the next handler to call in the chain.
func (h *rbacHandler) SetNext(next handlers.Handler) {
	h.next = next
}

func (h *rbacHandler) enforce(
	ctx context.Context,
	resource,
	action,
	object string,
) error {
	user, err := h.userGetter(ctx)
	if err != nil {
		return err
	}

	// User is allowed to perform the operation if the user's subject or any
	// of its groups have the required permission.
	for _, sub := range append([]string{user.Subject}, user.Groups...) {
		ok, err := h.enforcer.Enforce(sub, resource, action, object)
		if err != nil {
			return fmt.Errorf("enforce error: %w", err)
		}
		if ok {
			return nil
		}
	}

	h.log.Warnf("Permission denied: [%s %s %s %s]", user.Subject, resource, action, object)
	return ErrInsufficientPermissions
}
