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

import { render, screen, waitFor } from '@testing-library/react';
import { AxiosAdapter, AxiosError, AxiosResponse } from 'axios';
import { ReactNode, useContext } from 'react';
import { api } from 'api/api';
import { initializeAuthorizerFetchLoop } from 'utils/rbac';
import AuthContext from './auth.context';
import AuthProvider from './auth.provider';

const { fakeOidc } = vi.hoisted(() => ({
  fakeOidc: {
    signIn: vi.fn(),
    userManager: {
      signinSilent: vi.fn(),
      getUser: vi.fn<() => Promise<object | null>>(async () => null),
      clearStaleState: vi.fn(async () => {}),
      removeUser: vi.fn(async () => {}),
      signinSilentCallback: vi.fn(async () => {}),
      events: { addAccessTokenExpiring: vi.fn() },
    },
  },
}));

vi.mock('oidc-react', () => ({
  AuthProvider: ({ children }: { children: ReactNode }) => children,
  useAuth: () => fakeOidc,
}));

vi.mock('utils/rbac', () => ({
  initializeAuthorizerFetchLoop: vi.fn(),
  stopAuthorizerFetchLoop: vi.fn(),
}));

const nowInSeconds = Math.floor(Date.now() / 1000);
const toJwt = (payload: object) => `e30.${btoa(JSON.stringify(payload))}.sig`;
const ssoClaims = {
  iss: 'everest',
  sub: 'idp-user',
  oidc_issuer: 'https://idp.example',
};
const tokens = {
  sso: toJwt({ ...ssoClaims, jti: 'current', exp: nowInSeconds + 600 }),
  expiredSso: toJwt({ ...ssoClaims, jti: 'expired', exp: nowInSeconds - 60 }),
  renewedSso: toJwt({ ...ssoClaims, jti: 'renewed', exp: nowInSeconds + 900 }),
  builtIn: toJwt({
    iss: 'everest',
    sub: 'admin:login',
    exp: nowInSeconds + 3600,
  }),
  // Stored as-is by sessions created before the Everest token exchange.
  rawIdp: toJwt({
    iss: 'https://idp.example',
    sub: 'idp-user',
    exp: nowInSeconds + 600,
  }),
  expiredRawIdp: toJwt({
    iss: 'https://idp.example',
    sub: 'idp-user',
    exp: nowInSeconds - 60,
  }),
};

const rejectedTokens = new Set<string>([tokens.expiredSso]);
const requests: Array<{ url?: string; token: string | null }> = [];
const navigations: string[] = [];
let beforeRejecting: (() => void) | null = null;

const fakeServer: AxiosAdapter = async (config) => {
  const header = config.headers['Authorization'];
  const token =
    typeof header === 'string' ? header.replace('Bearer ', '') : null;
  requests.push({ url: config.url, token });

  const respond = (status: number, data: unknown) => {
    const response: AxiosResponse = {
      data,
      status,
      statusText: String(status),
      headers: {},
      config,
    };
    if (status < 400) {
      return response;
    }
    throw new AxiosError(
      'Rejected',
      AxiosError.ERR_BAD_REQUEST,
      config,
      {},
      response
    );
  };

  if (config.url === '/session/sso') {
    return respond(200, { token: tokens.renewedSso });
  }
  if (!token || rejectedTokens.has(token)) {
    beforeRejecting?.();
    return respond(401, { message: 'invalid or expired jwt' });
  }
  return respond(200, {});
};

const AuthStatus = () => (
  <span data-testid="auth-status">{useContext(AuthContext).authStatus}</span>
);

const renderOnSsoInstance = (savedToken: string) => {
  localStorage.setItem('everestToken', savedToken);
  render(
    <AuthProvider
      oidcConfig={{ authority: 'https://idp.example', clientId: 'everest' }}
    >
      <AuthStatus />
    </AuthProvider>
  );
  return waitFor(() =>
    expect(screen.getByTestId('auth-status').textContent).toBe('loggedIn')
  );
};

describe('AuthProvider session renewal', () => {
  beforeEach(() => {
    localStorage.clear();
    rejectedTokens.clear();
    rejectedTokens.add(tokens.expiredSso);
    requests.length = 0;
    navigations.length = 0;
    beforeRejecting = null;
    fakeOidc.userManager.signinSilent.mockReset();
    fakeOidc.userManager.signinSilent.mockResolvedValue({
      access_token: 'idp-access-token',
    });
    api.defaults.adapter = fakeServer;
    vi.stubGlobal('location', {
      set href(url: string) {
        navigations.push(url);
      },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('logs in after renewing an expired SSO token on page load', async () => {
    await renderOnSsoInstance(tokens.expiredSso);

    expect(localStorage.getItem('everestToken')).toBe(tokens.renewedSso);
    expect(initializeAuthorizerFetchLoop).toHaveBeenLastCalledWith('idp-user');
  });

  it('renews an expired raw IdP token on page load', async () => {
    await renderOnSsoInstance(tokens.expiredRawIdp);

    expect(localStorage.getItem('everestToken')).toBe(tokens.renewedSso);
  });

  it('renews a raw IdP token session on 401 and replays the request', async () => {
    fakeOidc.userManager.getUser.mockResolvedValueOnce({});
    await renderOnSsoInstance(tokens.rawIdp);
    rejectedTokens.add(tokens.rawIdp);

    await api.get('/database-clusters');

    expect(fakeOidc.userManager.signinSilent).toHaveBeenCalledTimes(1);
    expect(requests[requests.length - 1]).toEqual({
      url: '/database-clusters',
      token: tokens.renewedSso,
    });
    expect(navigations).toEqual([]);
  });

  it('renews an SSO session on 401 and replays the request', async () => {
    await renderOnSsoInstance(tokens.sso);
    rejectedTokens.add(tokens.sso);

    await api.get('/database-clusters');

    expect(fakeOidc.userManager.signinSilent).toHaveBeenCalledTimes(1);
    expect(requests[requests.length - 1]).toEqual({
      url: '/database-clusters',
      token: tokens.renewedSso,
    });
    expect(navigations).toEqual([]);
  });

  it('logs a built-in session out on 401 instead of signing in through the IdP', async () => {
    await renderOnSsoInstance(tokens.builtIn);
    rejectedTokens.add(tokens.builtIn);

    await api.get('/database-clusters');

    expect(fakeOidc.userManager.signinSilent).not.toHaveBeenCalled();
    expect(navigations).toEqual(['/logout']);
  });

  it('does not revive a session that was logged out in another tab', async () => {
    await renderOnSsoInstance(tokens.sso);
    localStorage.removeItem('everestToken');

    await api.get('permissions');

    expect(fakeOidc.userManager.signinSilent).not.toHaveBeenCalled();
    expect(navigations).toEqual(['/logout']);
  });

  it('reuses a token another tab already renewed instead of renewing again', async () => {
    await renderOnSsoInstance(tokens.sso);
    rejectedTokens.add(tokens.sso);
    beforeRejecting = () =>
      localStorage.setItem('everestToken', tokens.renewedSso);

    await api.get('/database-clusters');

    expect(fakeOidc.userManager.signinSilent).not.toHaveBeenCalled();
    expect(requests[requests.length - 1]).toEqual({
      url: '/database-clusters',
      token: tokens.renewedSso,
    });
    expect(navigations).toEqual([]);
  });
});
