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

import {expect, test} from '@playwright/test';
import {keycloakLogin} from '@tests/utils/keycloak';
import {KEYCLOAK_URL, API_SSO_TOKEN} from '@root/constants';

test.describe('SSO with Keycloak', () => {
  test('OIDC settings are configured', async ({request}) => {
    const resp = await request.get('/v1/settings');
    expect(resp.ok()).toBeTruthy();

    const settings = await resp.json();
    expect(settings.oidcConfig).toBeDefined();
    expect(settings.oidcConfig.issuerURL).toContain('/realms/everest');
    expect(settings.oidcConfig.clientId).toBeTruthy();
  });

  test('SSO access token is accepted as Bearer token', async ({request}) => {
    // Get a fresh token from Keycloak
    const tokens = await keycloakLogin(request, KEYCLOAK_URL);

    const resp = await request.get('/v1/version', {
      headers: {
        Authorization: `Bearer ${tokens.access_token}`,
      },
    });
    expect(resp.status(), `Expected 200 but got ${resp.status()}`).toEqual(200);
  });

  test('SSO token can access authenticated endpoints', async ({request}) => {
    // Use the project-level Authorization header (set from API_SSO_TOKEN in
    // playwright.config.ts).  No need for a fresh keycloakLogin — this test
    // verifies a protected endpoint works, not that keycloakLogin works.
    const resp = await request.get('/v1/namespaces');
    expect(resp.ok(), `Namespaces request failed: ${resp.status()}`).toBeTruthy();
  });

  test('logout invalidates SSO session', async ({request}) => {
    const tokens = await keycloakLogin(request, KEYCLOAK_URL);
    const authHeader = {Authorization: `Bearer ${tokens.access_token}`};

    // Verify the token works
    const beforeLogout = await request.get('/v1/version', {headers: authHeader});
    expect(beforeLogout.status()).toEqual(200);

    // Logout (add token to blocklist)
    const logoutResp = await request.delete('/v1/session', {headers: authHeader});
    expect(logoutResp.status()).toEqual(204);

    // Token should now be rejected
    const afterLogout = await request.get('/v1/version', {headers: authHeader});
    expect(afterLogout.status()).toEqual(401);
  });

  test('invalid SSO token is rejected', async ({request}) => {
    const resp = await request.get('/v1/version', {
      headers: {
        Authorization: 'Bearer invalid-sso-token-12345',
      },
    });
    expect(resp.status()).toEqual(401);
  });

  test('expired or tampered JWT is rejected', async ({request}) => {
    // Use the token already obtained during setup — no need for a fresh Keycloak login.
    const token = process.env['API_SSO_TOKEN'];
    expect(token, 'API_SSO_TOKEN must be set by setup').toBeTruthy();

    const parts = token!.split('.');
    expect(parts.length, 'Expected a JWT with 3 parts').toEqual(3);

    const tampered = parts[0] + '.' + parts[1] + 'tampered.' + parts[2];
    const resp = await request.get('/v1/version', {
      headers: {Authorization: `Bearer ${tampered}`},
    });
    expect(resp.status()).toEqual(401);
  });
});
