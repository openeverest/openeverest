// everest
// Copyright (C) 2023 Percona LLC
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

// Mocks must be declared before imports
vi.mock('oidc-react', () => {
  const userManager = {
    clearStaleState: vi.fn().mockResolvedValue(undefined),
    removeUser: vi.fn().mockResolvedValue(undefined),
    signinSilent: vi.fn().mockResolvedValue(null),
    signinSilentCallback: vi.fn(),
    getUser: vi.fn().mockResolvedValue(null),
    events: {
      addUserLoaded: vi.fn(),
      addAccessTokenExpiring: vi.fn(),
    },
  };
  return {
    AuthProvider: ({ children }: { children: ReactNode }) => children,
    useAuth: () => ({ signIn: vi.fn(), userManager }),
  };
});

vi.mock('utils/rbac', () => ({
  initializeAuthorizerFetchLoop: vi.fn(),
  stopAuthorizerFetchLoop: vi.fn(),
}));

vi.mock('notistack', () => ({
  enqueueSnackbar: vi.fn(),
}));

import type { ReactNode } from 'react';
import { useContext } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import AuthProvider from './auth.provider';
import AuthContext from './auth.context';
import * as apiModule from '../../api/api';

const LogoutConsumer = () => {
  const { logout } = useContext(AuthContext);
  return <button onClick={logout}>logout</button>;
};

describe('AuthProvider logout behavior', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
    sessionStorage.clear();
  });

  it('runs cleanup even when api.delete fails', async () => {
    vi.spyOn(apiModule.api, 'delete').mockRejectedValueOnce(new Error('fail'));
    const removeErrorSpy = vi.spyOn(apiModule, 'removeApiErrorInterceptor');
    const removeAuthSpy = vi.spyOn(apiModule, 'removeApiAuthInterceptor');

    render(
      <AuthProvider>
        <LogoutConsumer />
      </AuthProvider>
    );

    localStorage.setItem('everestToken', 'test-token');
    sessionStorage.setItem('auth-data', 'some-value');

    fireEvent.click(screen.getByRole('button', { name: 'logout' }));

    await waitFor(() => {
      expect(localStorage.getItem('everestToken')).toBeNull();
    });
    expect(sessionStorage.length).toBe(0);
    expect(removeErrorSpy).toHaveBeenCalled();
    expect(removeAuthSpy).toHaveBeenCalled();
  });
});
