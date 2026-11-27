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

import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import {
  render,
  renderHook,
  act,
  fireEvent,
  screen,
  waitFor,
} from '@testing-library/react';
import React, { useContext } from 'react';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { ThemeContextProvider, everestThemeOptions } from '@percona/design';
import { AxiosHeaders } from 'axios';
import { api, addApiErrorInterceptor } from 'api/api';
import ProtectedRoute from 'components/protected-route/ProtectedRoute';
import Login from 'pages/login/Login';
import { resizeScreenSize } from 'utils/test';
import AuthContext from './auth.context';

// Mock oidc-react
const mockUserManager = {
  events: {
    addUserLoaded: vi.fn(),
    addAccessTokenExpiring: vi.fn(),
    removeAccessTokenExpiring: vi.fn(),
  },
  signinSilentCallback: vi.fn(),
  signinSilent: vi.fn(),
  clearStaleState: vi.fn(),
  removeUser: vi.fn(),
  getUser: vi.fn(),
};

const mockOidcAuth = {
  isLoading: false,
  signIn: vi.fn(),
  userManager: mockUserManager,
};

vi.mock('oidc-react', () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  useAuth: () => mockOidcAuth,
}));

// Mock api module
vi.mock('api/api', () => ({
  api: { post: vi.fn() },
  addApiErrorInterceptor: vi.fn(),
  removeApiErrorInterceptor: vi.fn(),
  addApiAuthInterceptor: vi.fn(),
  removeApiAuthInterceptor: vi.fn(),
}));

// Mock session-token
const mockRefreshSession = vi.fn();
const mockGetAccessToken = vi.fn();
const mockGetAccessTokenExpiry = vi.fn();

vi.mock('api/session-token', () => ({
  setAccessToken: vi.fn(),
  getAccessToken: (...args: unknown[]) => mockGetAccessToken(...args),
  getAccessTokenExpiry: (...args: unknown[]) =>
    mockGetAccessTokenExpiry(...args),
  clearAccessToken: vi.fn(),
  refreshSession: (...args: unknown[]) => mockRefreshSession(...args),
  setSsoTokenRefresher: vi.fn(),
}));

// Mock notistack
const mockEnqueueSnackbar = vi.fn();
vi.mock('notistack', () => ({
  enqueueSnackbar: (...args: unknown[]) => mockEnqueueSnackbar(...args),
}));

// Mock rbac utils
vi.mock('utils/rbac', () => ({
  initializeAuthorizerFetchLoop: vi.fn(),
  stopAuthorizerFetchLoop: vi.fn(),
}));

// Mock consts
vi.mock('consts', () => ({
  EVEREST_JWT_ISSUER: 'everest',
}));

// Mock jwt-decode
vi.mock('jwt-decode', () => ({
  jwtDecode: () => ({ iss: 'everest', sub: 'admin:', exp: 9999999999 }),
}));

const { default: Provider } = await import('./auth.provider');

const renderSessionRoutes = (initialEntry: string, isSsoEnabled = false) => {
  const router = createMemoryRouter(
    [
      { path: '/login', element: <Login /> },
      {
        element: <ProtectedRoute />,
        children: [{ path: '*', element: <div>Protected instance</div> }],
      },
    ],
    { initialEntries: [initialEntry] }
  );
  const { result } = renderHook(() => useContext(AuthContext), {
    wrapper: ({ children }) => (
      <ThemeContextProvider themeOptions={everestThemeOptions}>
        <Provider
          oidcConfig={
            isSsoEnabled
              ? { authority: 'https://idp', clientId: 'everest' }
              : undefined
          }
        >
          {children}
          <RouterProvider router={router} />
        </Provider>
      </ThemeContextProvider>
    ),
  });
  return { router, auth: result };
};

describe('AuthProvider — redirects after an initialized session', () => {
  const destination = '/databases/default/example/overview?tab=metrics#cpu';

  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    sessionStorage.clear();
    resizeScreenSize(1200);
    mockOidcAuth.isLoading = false;
    mockGetAccessToken.mockReturnValue(null);
    mockGetAccessTokenExpiry.mockReturnValue(null);
    mockRefreshSession.mockResolvedValue(null);
    vi.mocked(api.post).mockResolvedValue({ data: { access_token: 'token' } });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('returns to a protected route visited after logged-out bootstrap has finished', async () => {
    const { router, auth } = renderSessionRoutes('/login');
    await screen.findByTestId('login-button');
    await act(() => router.navigate(destination));
    expect(router.state.location.pathname).toBe('/login');
    await act(() =>
      auth.current.login('manual', { username: 'admin', password: 'secret' })
    );
    const { pathname, search, hash } = router.state.location;
    expect(pathname + search + hash).toBe(destination);
  });

  it('returns to the active instance after proactive refresh fails', async () => {
    vi.useFakeTimers();
    mockGetAccessToken.mockReturnValue('access-token');
    mockGetAccessTokenExpiry.mockReturnValue(Date.now() + 65_000);
    const { router, auth } = renderSessionRoutes(destination);
    await act(async () => {});
    expect(auth.current.authStatus).toBe('loggedIn');
    expect(auth.current.redirectRoute).toBeNull();
    mockGetAccessToken.mockReturnValue(null);
    await act(() => vi.advanceTimersByTimeAsync(5_000));
    expect(router.state.location.pathname).toBe('/login');
    await act(() =>
      auth.current.login('manual', { username: 'admin', password: 'secret' })
    );
    const { pathname, search, hash } = router.state.location;
    expect(pathname + search + hash).toBe(destination);
  });

  it('hands the active instance URL to SSO after silent renewal fails', async () => {
    localStorage.setItem('everestToken', 'saved-token');
    mockUserManager.getUser.mockResolvedValue({ id_token: 'idp-token' });
    mockUserManager.signinSilent.mockResolvedValue(null);
    const { router, auth } = renderSessionRoutes(destination, true);
    await waitFor(() => expect(auth.current.authStatus).toBe('loggedIn'));
    const expire =
      mockUserManager.events.addAccessTokenExpiring.mock.lastCall?.[0];
    if (typeof expire !== 'function') {
      throw new Error('The SSO expiry listener was not registered');
    }
    await act(async () => {
      expire();
    });
    expect(router.state.location.pathname).toBe('/login');
    await act(() => auth.current.login('sso'));
    expect(mockOidcAuth.signIn).toHaveBeenCalledExactlyOnceWith({
      state: destination,
    });
  });

  it('returns to the active instance after the API interceptor reports expired authentication', async () => {
    mockGetAccessToken.mockReturnValue('access-token');
    const { router, auth } = renderSessionRoutes(destination);
    await waitFor(() => expect(auth.current.authStatus).toBe('loggedIn'));
    const onSessionExpired = vi.mocked(addApiErrorInterceptor).mock
      .lastCall?.[0];
    expect(onSessionExpired).toBeTypeOf('function');
    if (typeof onSessionExpired !== 'function') {
      throw new Error('The API session expiry handler was not registered');
    }
    mockGetAccessToken.mockReturnValue(null);
    await act(() => onSessionExpired());
    expect(api.post).toHaveBeenCalledWith('/auth/revoke', {});
    expect(router.state.location.pathname).toBe('/login');
    await act(() =>
      auth.current.login('manual', { username: 'admin', password: 'secret' })
    );
    const { pathname, search, hash } = router.state.location;
    expect(pathname + search + hash).toBe(destination);
  });

  it('does not restore a destination when an in-flight renewal fails during explicit logout', async () => {
    localStorage.setItem('everestToken', 'saved-token');
    mockUserManager.getUser.mockResolvedValue({ id_token: 'idp-token' });
    let finishRenewal = () => {};
    const renewal = new Promise<void>((resolve) => {
      finishRenewal = resolve;
    });
    mockUserManager.signinSilent.mockImplementationOnce(async () => {
      await renewal;
      return null;
    });
    const { router, auth } = renderSessionRoutes(destination, true);
    await waitFor(() => expect(auth.current.authStatus).toBe('loggedIn'));
    const expire =
      mockUserManager.events.addAccessTokenExpiring.mock.lastCall?.[0];
    if (typeof expire !== 'function') {
      throw new Error('The SSO expiry listener was not registered');
    }
    await act(async () => {
      expire();
    });
    let finishRevocation = () => {};
    const revocation = new Promise<void>((resolve) => {
      finishRevocation = resolve;
    });
    vi.mocked(api.post).mockImplementationOnce(async () => {
      await revocation;
      return {
        data: {},
        status: 200,
        statusText: 'OK',
        headers: {},
        config: { headers: new AxiosHeaders() },
      };
    });
    await act(async () => {
      auth.current.logout();
    });
    expect(auth.current.authStatus).toBe('loggingOut');
    expect(screen.queryByText('Protected instance')).not.toBeInTheDocument();
    await act(async () => {
      finishRenewal();
    });
    expect(router.state.location.pathname).toBe('/login');
    expect(auth.current.redirectRoute).toBeNull();
    await act(async () => {
      finishRevocation();
    });
    await act(() =>
      auth.current.login('manual', { username: 'admin', password: 'secret' })
    );
    expect(router.state.location.pathname).toBe('/');
  });

  it.each([
    { mode: 'explicit', revisit: false },
    { mode: 'storage', revisit: false },
    { mode: 'channel', revisit: false },
    { mode: 'explicit', revisit: true },
    { mode: 'storage', revisit: true },
    { mode: 'channel', revisit: true },
  ])(
    'clears the active destination on $mode logout and captures only a fresh visit (revisit: $revisit)',
    async ({ mode, revisit }) => {
      const channels: EventTarget[] = [];
      if (mode === 'channel') {
        vi.stubGlobal(
          'BroadcastChannel',
          class extends EventTarget {
            constructor() {
              super();
              channels.push(this);
            }
            postMessage = vi.fn();
            close = vi.fn();
          }
        );
      }
      mockGetAccessToken.mockReturnValue('access-token');
      const { router, auth } = renderSessionRoutes(destination);
      await waitFor(() => expect(auth.current.authStatus).toBe('loggedIn'));
      mockGetAccessToken.mockReturnValue(null);
      if (mode === 'explicit') {
        await act(() => auth.current.logout());
      } else if (mode === 'storage') {
        await act(async () => {
          window.dispatchEvent(
            new StorageEvent('storage', {
              key: 'everest-auth-sync',
              newValue: JSON.stringify({
                type: 'logout',
                sender: 'another-tab',
              }),
            })
          );
        });
      } else {
        expect(channels).toHaveLength(1);
        await act(async () => {
          channels[0].dispatchEvent(
            new MessageEvent('message', {
              data: { type: 'logout', sender: 'another-tab' },
            })
          );
        });
      }
      expect(router.state.location.pathname).toBe('/login');
      expect(auth.current.redirectRoute).toBeNull();
      if (revisit) {
        await act(() => router.navigate(destination));
        expect(router.state.location.pathname).toBe('/login');
      }
      await act(() =>
        auth.current.login('manual', { username: 'admin', password: 'secret' })
      );
      const { pathname, search, hash } = router.state.location;
      expect(pathname + search + hash).toBe(revisit ? destination : '/');
    }
  );
});

describe('AuthProvider — post-login navigation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    sessionStorage.clear();
    resizeScreenSize(1200);
    mockGetAccessToken.mockReturnValue(null);
    mockGetAccessTokenExpiry.mockReturnValue(null);
    mockRefreshSession.mockResolvedValue(null);
    mockOidcAuth.isLoading = false;
    vi.mocked(api.post).mockResolvedValue({ data: { access_token: 'token' } });
  });

  it.each([
    {
      initialEntry: '/databases/default/example/overview?tab=metrics#cpu',
      retry: false,
    },
    {
      initialEntry: '/databases/default/example/overview?tab=metrics#cpu',
      retry: true,
    },
    { initialEntry: '/login', retry: false },
  ])(
    'returns to $initialEntry after credential login (retry: $retry)',
    async ({ initialEntry, retry }) => {
      const destination = initialEntry === '/login' ? '/' : initialEntry;
      const router = createMemoryRouter(
        [
          { path: '/login', element: <Login /> },
          {
            element: <ProtectedRoute />,
            children: [{ path: '*', element: <div>Protected instance</div> }],
          },
        ],
        { initialEntries: [initialEntry] }
      );
      render(
        <ThemeContextProvider themeOptions={everestThemeOptions}>
          <Provider>
            <RouterProvider router={router} />
          </Provider>
        </ThemeContextProvider>
      );

      await screen.findByTestId('login-button');
      expect(router.state.location.pathname).toBe('/login');
      expect(screen.queryByText('Protected instance')).not.toBeInTheDocument();
      fireEvent.change(screen.getByTestId('text-input-username'), {
        target: { value: 'admin' },
      });
      fireEvent.change(screen.getByTestId('text-input-password'), {
        target: { value: 'secret' },
      });
      if (retry) {
        vi.mocked(api.post).mockRejectedValueOnce(new Error('Login failed'));
        fireEvent.click(screen.getByTestId('login-button'));
        await waitFor(() => expect(api.post).toHaveBeenCalledTimes(1));
        await waitFor(() =>
          expect(screen.getByTestId('login-button')).toBeEnabled()
        );
        expect(router.state.location.pathname).toBe('/login');
      }
      fireEvent.click(screen.getByTestId('login-button'));

      await waitFor(() => {
        const { pathname, search, hash } = router.state.location;
        expect(pathname + search + hash).toBe(destination);
      });
      expect(screen.getByText('Protected instance')).toBeInTheDocument();
      await act(() => router.navigate('/login'));
      await waitFor(() => expect(router.state.location.pathname).toBe('/'));
    }
  );

  it.each(['explicit', 'synchronized'])(
    'clears the saved destination on %s logout',
    async (mode) => {
      const { result } = renderHook(() => useContext(AuthContext), {
        wrapper: Provider,
      });
      await waitFor(() => expect(result.current.authStatus).toBe('loggedOut'));
      act(() => result.current.setRedirectRoute('/settings/namespaces'));
      expect(result.current.redirectRoute).toBe('/settings/namespaces');
      if (mode === 'explicit') {
        await act(() => result.current.logout());
        expect(api.post).toHaveBeenCalledWith('/auth/revoke', {});
      } else {
        await act(() =>
          window.dispatchEvent(
            new StorageEvent('storage', {
              key: 'everest-auth-sync',
              newValue: JSON.stringify({
                type: 'logout',
                sender: 'another-tab',
              }),
            })
          )
        );
      }
      expect(result.current.redirectRoute).toBeNull();
    }
  );

  it('hands the full destination to OIDC state for the external sign-in round trip', async () => {
    const { result } = renderHook(() => useContext(AuthContext), {
      wrapper: Provider,
    });
    await waitFor(() => expect(result.current.authStatus).toBe('loggedOut'));
    const destination = '/settings/namespaces?cluster=default#namespaces';
    act(() => result.current.setRedirectRoute(destination));
    await act(() => result.current.login('sso'));
    expect(mockOidcAuth.signIn).toHaveBeenCalledExactlyOnceWith({
      state: destination,
    });
  });

  it('waits for OIDC callback processing before bootstrapping the session', async () => {
    mockOidcAuth.isLoading = true;
    const { result, rerender } = renderHook(() => useContext(AuthContext), {
      wrapper: ({ children }) => (
        <Provider
          oidcConfig={{ authority: 'https://idp', clientId: 'everest' }}
        >
          {children}
        </Provider>
      ),
    });
    expect(result.current.authStatus).toBe('unknown');
    expect(mockRefreshSession).not.toHaveBeenCalled();
    expect(mockUserManager.clearStaleState).not.toHaveBeenCalled();
    mockOidcAuth.isLoading = false;
    rerender();
    await waitFor(() => expect(result.current.authStatus).toBe('loggedOut'));
    expect(mockRefreshSession).toHaveBeenCalledOnce();
  });

  it('preserves the destination while revoking a broken SSO session during bootstrap', async () => {
    localStorage.setItem('everestToken', 'saved-token');
    mockUserManager.getUser.mockRejectedValueOnce(
      new Error('Session unavailable')
    );
    const { result } = renderHook(() => useContext(AuthContext), {
      wrapper: ({ children }) => (
        <Provider
          oidcConfig={{ authority: 'https://idp', clientId: 'everest' }}
        >
          {children}
        </Provider>
      ),
    });
    act(() => result.current.setRedirectRoute('/settings/namespaces'));
    await waitFor(() => expect(result.current.authStatus).toBe('loggedOut'));
    expect(result.current.redirectRoute).toBe('/settings/namespaces');
    expect(api.post).toHaveBeenCalledWith('/auth/revoke', {});
    expect(localStorage.getItem('everestToken')).toBeNull();
  });
});

describe('AuthProvider — proactive refresh timer', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    mockGetAccessToken.mockReturnValue(null);
    mockGetAccessTokenExpiry.mockReturnValue(null);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function renderProvider() {
    return render(
      <Provider>
        <div data-testid="child" />
      </Provider>
    );
  }

  it('schedules refresh at (expiry - 60s), minimum 5s', async () => {
    // Simulate bootstrapSession resolving with an access token
    mockRefreshSession.mockResolvedValue('access-token');
    mockGetAccessToken.mockReturnValue('access-token');
    // Expiry in 120 seconds → delay should be 120 - 60 = 60s
    mockGetAccessTokenExpiry.mockReturnValue(Date.now() + 120 * 1000);

    renderProvider();

    // Wait for bootstrapSession to complete and trigger loggedIn status
    await act(async () => {
      await Promise.resolve();
    });

    // The timer should NOT have fired yet at 59s
    mockRefreshSession.mockClear();
    mockRefreshSession.mockResolvedValue('new-access-token');
    mockGetAccessTokenExpiry.mockReturnValue(Date.now() + 120 * 1000);

    await act(async () => {
      vi.advanceTimersByTime(59_000);
    });
    expect(mockRefreshSession).not.toHaveBeenCalled();

    // At 60s it should fire
    await act(async () => {
      vi.advanceTimersByTime(1_000);
    });
    expect(mockRefreshSession).toHaveBeenCalledTimes(1);
  });

  it('enforces minimum 5s delay when token is about to expire', async () => {
    // Token expiring in 10s → delay should be max(10-60, 5) = 5s
    mockRefreshSession.mockResolvedValue('access-token');
    mockGetAccessToken.mockReturnValue('access-token');
    mockGetAccessTokenExpiry.mockReturnValue(Date.now() + 10 * 1000);

    renderProvider();

    await act(async () => {
      await Promise.resolve();
    });

    mockRefreshSession.mockClear();
    mockRefreshSession.mockResolvedValue('new-token');
    mockGetAccessTokenExpiry.mockReturnValue(Date.now() + 900 * 1000);

    // Should NOT fire at 4s
    await act(async () => {
      vi.advanceTimersByTime(4_000);
    });
    expect(mockRefreshSession).not.toHaveBeenCalled();

    // Should fire at 5s
    await act(async () => {
      vi.advanceTimersByTime(1_000);
    });
    expect(mockRefreshSession).toHaveBeenCalledTimes(1);
  });

  it('calls logout when proactive refresh fails', async () => {
    mockRefreshSession.mockResolvedValue('access-token');
    mockGetAccessToken.mockReturnValue('access-token');
    mockGetAccessTokenExpiry.mockReturnValue(Date.now() + 65 * 1000);

    renderProvider();

    await act(async () => {
      await Promise.resolve();
    });

    // Proactive refresh will fail
    mockRefreshSession.mockClear();
    mockRefreshSession.mockResolvedValue(null);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5_000);
    });

    expect(mockRefreshSession).toHaveBeenCalledTimes(1);
    expect(mockEnqueueSnackbar).toHaveBeenCalledWith(
      'Your session has expired. Please sign in again.',
      { variant: 'info' }
    );
  });
});

describe('AuthProvider — SSO renewal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetAccessToken.mockReturnValue(null);
    mockRefreshSession.mockResolvedValue(null);
  });

  it('exchanges the IdP ID token, not the access token', async () => {
    const { api } = await import('api/api');
    const { setSsoTokenRefresher } = await import('api/session-token');
    vi.mocked(api.post).mockResolvedValue({
      data: { access_token: 'everest-jwt' },
    });
    mockUserManager.signinSilent.mockResolvedValue({
      access_token: 'idp-access-token',
      id_token: 'idp-id-token',
    });

    render(
      <Provider oidcConfig={{ authority: 'https://idp', clientId: 'everest' }}>
        <div />
      </Provider>
    );
    await act(async () => {
      await Promise.resolve();
    });

    const refresher = vi
      .mocked(setSsoTokenRefresher)
      .mock.calls.map(([fn]) => fn)
      .find((fn) => fn !== null);
    expect(refresher).toBeDefined();

    await expect(refresher?.()).resolves.toBe('everest-jwt');
    expect(api.post).toHaveBeenCalledWith(
      '/auth/token',
      expect.objectContaining({
        subject_token: 'idp-id-token',
        subject_token_type: 'urn:ietf:params:oauth:token-type:id_token',
      })
    );
  });
});
