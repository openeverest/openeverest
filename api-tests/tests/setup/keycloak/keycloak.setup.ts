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

import {test as setup, expect} from '@playwright/test';
import {
  waitForKeycloak,
  configureKeycloak,
  keycloakLogin,
  grantSSOUserAdmin,
  KEYCLOAK_IN_CLUSTER_URL,
} from '@tests/utils/keycloak';
import {
  KEYCLOAK_URL,
  API_SSO_TOKEN,
} from '@root/constants';
import {CliHelper} from '@helpers/cliHelper';

// OIDC config YAML written directly to everest-settings ConfigMap.
// Using the in-cluster Keycloak service URL so the Everest backend pod can reach it.
const buildOIDCConfigYAML = (clientId: string) =>
  `issuerUrl: ${KEYCLOAK_IN_CLUSTER_URL}/realms/everest\nclientId: ${clientId}\nscopes:\n    - openid\n    - profile\n    - email`;

setup.describe.serial('Keycloak SSO setup', () => {
  const cli = new CliHelper();

  setup('Wait for Keycloak readiness', async ({request}) => {
    await waitForKeycloak(request, KEYCLOAK_URL);
  });

  setup('Configure Keycloak realm, client, and user', async ({request}) => {
    const everestURL = process.env.EVEREST_URL || 'http://localhost:8080';
    const config = await configureKeycloak(request, KEYCLOAK_URL, everestURL);

    // Store config in env for subsequent steps
    process.env.KEYCLOAK_CLIENT_ID = config.clientId;
    process.env.KEYCLOAK_SSO_USER = config.username;
    process.env.KEYCLOAK_SSO_PASSWORD = config.password;
  });

  setup('Configure Everest OIDC settings via ConfigMap', async () => {
    const clientId = process.env.KEYCLOAK_CLIENT_ID;
    expect(clientId, 'KEYCLOAK_CLIENT_ID must be set').toBeTruthy();

    // Patch the ConfigMap directly — no network validation, works without port-forward.
    const oidcYAML = buildOIDCConfigYAML(clientId!);
    const patch = JSON.stringify({data: {'oidc.config': oidcYAML}});
    const patchResult = await cli.execute(
      `kubectl patch configmap everest-settings -n everest-system --type merge -p '${patch}'`,
    );
    expect(patchResult.code, `ConfigMap patch failed: ${patchResult.stderr}`).toEqual(0);

    // Restart Everest so it picks up the new OIDC config.
    const restartResult = await cli.execute(
      'kubectl rollout restart deployment/everest-server -n everest-system',
    );
    expect(restartResult.code, `Restart failed: ${restartResult.stderr}`).toEqual(0);

    // Wait for Everest to be ready again.
    const waitResult = await cli.execute(
      'kubectl rollout status deployment/everest-server -n everest-system --timeout=120s',
    );
    expect(waitResult.code, `Everest did not become ready: ${waitResult.stderr}`).toEqual(0);
  });

  setup('Login via Keycloak and obtain SSO token', async ({request}) => {
    const tokens = await keycloakLogin(request, KEYCLOAK_URL);

    // Store the access token for SSO test project
    process.env[API_SSO_TOKEN] = tokens.access_token;
  });

  setup('Grant SSO user admin RBAC role', async () => {
    // The Keycloak user's subject in the JWT is the username
    await grantSSOUserAdmin('sso-test-user');
  });
});
