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

	// Hard denylist: plugin daemon tokens cannot write to spec-001 resources.
	if rbac.IsPluginWriteDenied(user.Subject, resource, action) {
		h.log.Warnf("Plugin write denied: [%s %s %s %s]", user.Subject, resource, action, object)
		return ErrInsufficientPermissions
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
