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

import { render, waitFor } from '@testing-library/react';
import { AuthProviderProps, User } from 'oidc-react';
import { api } from 'api/api';
import { resizeScreenSize } from 'utils/test';
import App from './App';

let oidcConfig: AuthProviderProps | undefined;

vi.mock('contexts/auth', () => ({
  AuthProvider: ({ oidcConfig: config }: { oidcConfig: AuthProviderProps }) => {
    oidcConfig = config;
    return null;
  },
}));

vi.mock('router', () => ({ default: {} }));
vi.mock('api/everestConfigs', () => ({
  getEverestConfigs: vi.fn().mockResolvedValue({}),
}));
vi.mock('api/api', () => ({ api: { post: vi.fn() } }));

describe('App SSO callback navigation', () => {
  beforeEach(() => {
    oidcConfig = undefined;
    vi.clearAllMocks();
    localStorage.clear();
    resizeScreenSize(1200);
    vi.mocked(api.post).mockResolvedValue({
      data: { access_token: 'everest-jwt' },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it.each([
    {
      state: '/settings/namespaces?cluster=default#namespaces',
      expected: '/settings/namespaces?cluster=default#namespaces',
    },
    { state: undefined, expected: '/' },
    { state: '/\\evil.example', expected: '/' },
    { state: '/%6cogin?again=true', expected: '/' },
    { state: { redirect: 'https://evil.example' }, expected: '/' },
  ])(
    'uses a safe destination from OIDC state: $state',
    async ({ state, expected }) => {
      render(<App />);
      await waitFor(() => expect(oidcConfig?.onSignIn).toBeDefined());
      const location = {
        href: '/login-callback',
        origin: window.location.origin,
      };
      vi.stubGlobal('window', { location });
      const user = new User({
        access_token: 'idp-access-token',
        id_token: 'idp-id-token',
        token_type: 'Bearer',
        profile: {
          sub: 'admin',
          iss: 'https://idp',
          aud: 'everest',
          exp: 9999999999,
          iat: 1,
        },
        userState: state,
      });

      await oidcConfig?.onSignIn?.(user);

      expect(api.post).toHaveBeenCalledExactlyOnceWith('/auth/token', {
        grant_type: 'urn:ietf:params:oauth:grant-type:token-exchange',
        subject_token: 'idp-id-token',
        subject_token_type: 'urn:ietf:params:oauth:token-type:id_token',
      });
      expect(localStorage.getItem('everestToken')).toBe('everest-jwt');
      expect(location.href).toBe(expected);
      vi.unstubAllGlobals();
    }
  );
});
