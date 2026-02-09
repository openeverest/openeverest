// everest
// Copyright (C) 2023 Percona LLC
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

// Package oidc ...
package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openeverest/openeverest/v2/pkg/cli/steps"
	"github.com/openeverest/openeverest/v2/pkg/cli/tui"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// OIDC describes the command to configure OIDC settings.
type OIDC struct {
	config     Config
	kubeClient kubernetes.KubernetesConnector
	l          *zap.SugaredLogger
}

// ---- Config ----

// Config stores configuration for the OIDC command.
type Config struct {
	// KubeconfigPath is a path to a kubeconfig
	KubeconfigPath string
	// Pretty print the output.
	Pretty bool
	// IssuerURL OIDC issuer url.
	IssuerURL string
	// ClientID ID of the client OIDC app.
	ClientID string
	// Scopes requested scopes.
	Scopes []string
}

// PopulateIssuerURL function to fill the configuration with the required IssuerURL.
// This function shall be called only in cases when there is no other way to obtain value for IssuerURL.
// User will be asked to provide the IssuerURL in interactive mode.
// Provided by user url will be parsed, validated and stored in the IssuerURL property.
// Note: in case IssuerURL is not empty - it will be overwritten by user's input.
func (cfg *Config) PopulateIssuerURL(ctx context.Context) error {
	// ask user to provide issuer URL
	var err error
	if cfg.IssuerURL, err = tui.NewInput(ctx, "Provide issuer URL",
		tui.WithInputValidation(ValidateURL),
	).Run(); err != nil {
		return err
	}

	return nil
}

// PopulateClientID function to fill the configuration with the required ClientID.
// This function shall be called only in cases when there is no other way to obtain value for ClientID.
// User will be asked to provide the ClientID in interactive mode.
// Provided by user value will be parsed, validated and stored in the ClientID property.
// Note: in case ClientID is not empty - it will be overwritten by user's input.
func (cfg *Config) PopulateClientID(ctx context.Context) error {
	// ask user to provide client ID
	var err error
	if cfg.ClientID, err = tui.NewInput(ctx, "Provide Client ID",
		tui.WithInputValidation(ValidateClientID),
	).Run(); err != nil {
		return err
	}

	return nil
}

// ---- OIDC ----

// NewOIDC returns a new OIDC struct.
func NewOIDC(c Config, l *zap.SugaredLogger) (*OIDC, error) {
	cli := &OIDC{
		config: c,
		l:      l.With("component", "oidc"),
	}

	if c.Pretty {
		cli.l = zap.NewNop().Sugar()
	}

	k, err := cliutils.NewKubeConnector(cli.l, c.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	cli.kubeClient = k

	return cli, nil
}

// Run runs the command.
func (u *OIDC) Run(ctx context.Context) error {
	if err := ValidateURL(u.config.IssuerURL); err != nil {
		return err
	}

	if err := ValidateClientID(u.config.ClientID); err != nil {
		return err
	}

	if err := steps.RunStepsWithSpinner(ctx, u.l, u.getOIDCProviderConfigureSteps(), u.config.Pretty); err != nil {
		return err
	}
	u.l.Info("OIDC has been configured successfully")
	return nil
}

// getOIDCProviderConfigureSteps returns the steps to configure the OIDC provider.
func (u *OIDC) getOIDCProviderConfigureSteps() []steps.Step {
	var stepList []steps.Step
	stepList = append(stepList, steps.Step{
		Desc: "Checking connection to the OIDC provider",
		F: func(ctx context.Context) error {
			// Check if we can connect to the provider.
			if _, err := NewProviderConfig(ctx, u.config.IssuerURL); err != nil {
				if errors.Is(err, ErrUnexpectedSatusCode) {
					return fmt.Errorf("failed to connect with OIDC provider due to incorrect response: %s", err)
				}
				return fmt.Errorf("failed to connect with OIDC provider: %w", err)
			}
			return nil
		},
	})

	stepList = append(stepList, steps.Step{
		Desc: "Updating Everest settings",
		F: func(ctx context.Context) error {
			oidcCfg := common.OIDCConfig{
				IssuerURL: u.config.IssuerURL,
				ClientID:  u.config.ClientID,
				Scopes:    u.config.Scopes,
			}

			oidcRaw, err := oidcCfg.Raw()
			if err != nil {
				return err
			}
			return u.kubeClient.UpdateEverestSettings(ctx, common.EverestSettings{
				OIDCConfigRaw: oidcRaw,
			})
		},
	},
	)

	// Restart Everest to apply the changes.
	stepList = append(stepList, steps.Step{
		Desc: "Restarting Everest",
		F: func(ctx context.Context) error {
			return u.kubeClient.RestartDeployment(ctx, types.NamespacedName{
				Namespace: common.SystemNamespace,
				Name:      common.PerconaEverestDeploymentName,
			})
		},
	},
	)

	return stepList
}

// ValidateURL checks if the provided URL is valid.
func ValidateURL(u string) error {
	if u == "" {
		return errors.New("URL is required")
	}

	parsedURL, err := url.ParseRequestURI(u)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme == "" {
		return errors.New("URL scheme is required")
	}

	if parsedURL.Host == "" {
		return errors.New("URL host is required")
	}

	return nil
}

// ValidateClientID checks if the provided client ID is valid.
func ValidateClientID(clientID string) error {
	if clientID == "" {
		return errors.New("client ID is required")
	}

	return nil
}

// ValidateScopes checks if the provided scopes are valid.
func ValidateScopes(scopes []string) error {
	if !slices.ContainsFunc(scopes, func(s string) bool {
		return s == "openid"
	}) {
		return errors.New("scopes must contain 'openid'")
	}
	return nil
}
