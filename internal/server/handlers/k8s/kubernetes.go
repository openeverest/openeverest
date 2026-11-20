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

// Package k8s contains the Kubernetes handler.
package k8s

import (
	"context"
	"errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func (h *k8sHandler) GetUserPermissions(ctx context.Context) (*api.UserPermissions, error) {
	cm, err := h.kubeConnector.GetConfigMap(ctx, types.NamespacedName{Namespace: h.kubeConnector.Namespace(), Name: common.EverestRBACConfigMapName})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not get Everest RBAC ConfigMap"))
	}
	enabled := rbac.IsEnabled(cm)
	return &api.UserPermissions{
		Enabled: enabled,
	}, nil
}

func (h *k8sHandler) GetSettings(ctx context.Context) (*api.Settings, error) {
	settings, err := h.kubeConnector.GetEverestSettings(ctx)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	config, err := settings.OIDCConfig()
	if err != nil {
		return nil, errors.Join(err, errors.New("cannot parse OIDC raw config"))
	}
	return &api.Settings{
		OidcConfig: api.OIDCConfig{
			ClientId:  config.ClientID,
			IssuerURL: config.IssuerURL,
			Scopes:    config.Scopes,
		},
	}, nil
}
