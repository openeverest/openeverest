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

import {APIRequestContext, expect} from '@playwright/test';

const KEYCLOAK_ADMIN_USER = 'admin';
const KEYCLOAK_ADMIN_PASSWORD = 'admin';
const KEYCLOAK_REALM = 'everest';
const KEYCLOAK_CLIENT_ID = 'everest-app';
const KEYCLOAK_TEST_USER = 'sso-test-user';
const KEYCLOAK_TEST_PASSWORD = 'sso-test-password';
const KEYCLOAK_SCOPES = ['openid', 'profile', 'email'];

// In-cluster Keycloak service URL — used as the OIDC issuer configured in Everest,
// so the Everest backend pod can reach Keycloak for JWKS validation.
export const KEYCLOAK_IN_CLUSTER_URL =
  'http://keycloak.everest-system.svc.cluster.local:8080';

export interface KeycloakConfig {
  issuerURL: string;
  clientId: string;
  username: string;
  password: string;
  userId: string;
  baseURL: string;
  realm: string;
}

/**
 * Get an admin access token from Keycloak master realm.
 */
const getAdminToken = async (
  request: APIRequestContext,
  keycloakBaseURL: string,
): Promise<string> => {
  const resp = await request.post(
    `${keycloakBaseURL}/realms/master/protocol/openid-connect/token`,
    {
      headers: {'Content-Type': 'application/x-www-form-urlencoded'},
      form: {
        grant_type: 'password',
        client_id: 'admin-cli',
        username: KEYCLOAK_ADMIN_USER,
        password: KEYCLOAK_ADMIN_PASSWORD,
      },
    },
  );
  expect(resp.ok(), `Failed to get Keycloak admin token: ${resp.status()}`).toBeTruthy();
  const body = await resp.json();
  return body.access_token as string;
};

/**
 * Create the 'everest' realm in Keycloak.
 */
const createRealm = async (
  request: APIRequestContext,
  keycloakBaseURL: string,
  adminToken: string,
): Promise<void> => {
  const resp = await request.post(
    `${keycloakBaseURL}/admin/realms`,
    {
      headers: {Authorization: `Bearer ${adminToken}`},
      data: {
        realm: KEYCLOAK_REALM,
        enabled: true,
      },
    },
  );
  // 409 = realm already exists, which is fine
  expect(
    resp.ok() || resp.status() === 409,
    `Failed to create Keycloak realm: ${resp.status()}`,
  ).toBeTruthy();

  // Disable the VERIFY_PROFILE required action at realm level.
  // Keycloak 26 enables this by default, which blocks ROPC (password grant) logins
  // with "Account is not fully set up".
  const vpResp = await request.put(
    `${keycloakBaseURL}/admin/realms/${KEYCLOAK_REALM}/authentication/required-actions/VERIFY_PROFILE`,
    {
      headers: {Authorization: `Bearer ${adminToken}`},
      data: {
        alias: 'VERIFY_PROFILE',
        name: 'Verify Profile',
        providerId: 'VERIFY_PROFILE',
        enabled: false,
        defaultAction: false,
        priority: 90,
      },
    },
  );
  expect(
    vpResp.ok(),
    `Failed to disable VERIFY_PROFILE: ${vpResp.status()}`,
  ).toBeTruthy();
};

/**
 * Create a public OIDC client in the realm.
 */
const createClient = async (
  request: APIRequestContext,
  keycloakBaseURL: string,
  adminToken: string,
  everestURL: string,
): Promise<void> => {
  const resp = await request.post(
    `${keycloakBaseURL}/admin/realms/${KEYCLOAK_REALM}/clients`,
    {
      headers: {Authorization: `Bearer ${adminToken}`},
      data: {
        clientId: KEYCLOAK_CLIENT_ID,
        enabled: true,
        publicClient: true,
        directAccessGrantsEnabled: true,
        standardFlowEnabled: true,
        redirectUris: [`${everestURL}/*`],
        webOrigins: [everestURL],
        protocol: 'openid-connect',
      },
    },
  );
  expect(
    resp.ok() || resp.status() === 409,
    `Failed to create Keycloak client: ${resp.status()}`,
  ).toBeTruthy();
};

/**
 * Create a test user in the realm and return their ID.
 */
const createUser = async (
  request: APIRequestContext,
  keycloakBaseURL: string,
  adminToken: string,
): Promise<string> => {
  const createResp = await request.post(
    `${keycloakBaseURL}/admin/realms/${KEYCLOAK_REALM}/users`,
    {
      headers: {Authorization: `Bearer ${adminToken}`},
      data: {
        username: KEYCLOAK_TEST_USER,
        enabled: true,
        emailVerified: true,
        email: `${KEYCLOAK_TEST_USER}@example.com`,
        requiredActions: [],
      },
    },
  );
  expect(
    createResp.ok() || createResp.status() === 409,
    `Failed to create Keycloak user: ${createResp.status()}`,
  ).toBeTruthy();

  // Fetch the user to get their ID
  const searchResp = await request.get(
    `${keycloakBaseURL}/admin/realms/${KEYCLOAK_REALM}/users?username=${KEYCLOAK_TEST_USER}&exact=true`,
    {headers: {Authorization: `Bearer ${adminToken}`}},
  );
  expect(searchResp.ok(), `Failed to search Keycloak user: ${searchResp.status()}`).toBeTruthy();
  const users = await searchResp.json();
  expect(users.length, 'Expected to find the created user').toBeGreaterThan(0);
  const userId = users[0].id as string;

  // Set password via dedicated endpoint (more reliable than credentials in create payload)
  const pwResp = await request.put(
    `${keycloakBaseURL}/admin/realms/${KEYCLOAK_REALM}/users/${userId}/reset-password`,
    {
      headers: {Authorization: `Bearer ${adminToken}`},
      data: {
        type: 'password',
        value: KEYCLOAK_TEST_PASSWORD,
        temporary: false,
      },
    },
  );
  expect(pwResp.ok(), `Failed to set Keycloak user password: ${pwResp.status()}`).toBeTruthy();

  // Explicitly clear required actions (Keycloak 26 adds VERIFY_PROFILE by default)
  const updateResp = await request.put(
    `${keycloakBaseURL}/admin/realms/${KEYCLOAK_REALM}/users/${userId}`,
    {
      headers: {Authorization: `Bearer ${adminToken}`},
      data: {requiredActions: []},
    },
  );
  expect(
    updateResp.ok(),
    `Failed to clear required actions: ${updateResp.status()}`,
  ).toBeTruthy();

  return userId;
};

/**
 * Wait for Keycloak to be ready by polling the master realm OIDC discovery endpoint.
 */
export const waitForKeycloak = async (
  request: APIRequestContext,
  keycloakBaseURL: string,
  timeoutMs = 120_000,
): Promise<void> => {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const resp = await request.get(
        `${keycloakBaseURL}/realms/master/.well-known/openid-configuration`,
      );
      if (resp.ok()) return;
    } catch {
      // Keycloak not ready yet
    }
    await new Promise((r) => setTimeout(r, 2000));
  }
  throw new Error(`Keycloak did not become ready within ${timeoutMs}ms`);
};

/**
 * Configure Keycloak with a realm, client, and test user.
 * Returns the configuration needed for Everest OIDC setup and SSO tests.
 */
export const configureKeycloak = async (
  request: APIRequestContext,
  keycloakBaseURL: string,
  everestURL: string,
): Promise<KeycloakConfig> => {
  const adminToken = await getAdminToken(request, keycloakBaseURL);
  await createRealm(request, keycloakBaseURL, adminToken);
  await createClient(request, keycloakBaseURL, adminToken, everestURL);
  const userId = await createUser(request, keycloakBaseURL, adminToken);

  return {
    issuerURL: `${keycloakBaseURL}/realms/${KEYCLOAK_REALM}`,
    clientId: KEYCLOAK_CLIENT_ID,
    username: KEYCLOAK_TEST_USER,
    password: KEYCLOAK_TEST_PASSWORD,
    userId,
    baseURL: keycloakBaseURL,
    realm: KEYCLOAK_REALM,
  };
};

/**
 * Perform OIDC Resource Owner Password Credentials (direct access grant) login
 * against Keycloak and return the access token.
 *
 * This uses Keycloak's "direct access grants" (ROPC) which is enabled on the client
 * to allow programmatic login without browser interaction.
 */
export const keycloakLogin = async (
  request: APIRequestContext,
  keycloakBaseURL: string,
  username?: string,
  password?: string,
): Promise<{access_token: string; id_token: string; refresh_token: string}> => {
  const resp = await request.post(
    `${keycloakBaseURL}/realms/${KEYCLOAK_REALM}/protocol/openid-connect/token`,
    {
      headers: {'Content-Type': 'application/x-www-form-urlencoded'},
      form: {
        grant_type: 'password',
        client_id: KEYCLOAK_CLIENT_ID,
        username: username ?? KEYCLOAK_TEST_USER,
        password: password ?? KEYCLOAK_TEST_PASSWORD,
        scope: KEYCLOAK_SCOPES.join(' '),
      },
    },
  );
  const body = await resp.text();
  expect(resp.ok(), `Keycloak login failed: HTTP ${resp.status()} — ${body}`).toBeTruthy();
  return JSON.parse(body);
};

/**
 * Add the SSO test user to the Everest admin RBAC role.
 * This patches the everest-rbac ConfigMap to grant the SSO user admin permissions,
 * similar to how the CI user is granted admin in the standard test setup.
 */
export const grantSSOUserAdmin = async (
  ssoSubject: string,
): Promise<void> => {
  const {execSync} = await import('child_process');
  // Patch the configmap to add the SSO user to the admin role
  const patchCmd = `kubectl get configmap everest-rbac -n everest-system -o json | jq '.data["policy.csv"] += "\\ng, ${ssoSubject}, role:admin"' | jq '{data: { "policy.csv": .data["policy.csv"] } }' | xargs -0 kubectl patch configmap everest-rbac -n everest-system --type merge -p`;
  // Simpler approach: just append to the policy
  execSync(
    `kubectl patch configmap everest-rbac -n everest-system --type merge -p "$(kubectl get configmap everest-rbac -n everest-system -o json | jq '{data: { "policy.csv": (.data["policy.csv"] + "\\ng, ${ssoSubject}, role:admin") } }')"`,
    {stdio: 'pipe'},
  );
};
