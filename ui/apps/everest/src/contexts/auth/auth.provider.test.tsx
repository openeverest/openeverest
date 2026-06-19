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
import { render, act } from '@testing-library/react';
import React from 'react';

// Mock oidc-react
const mockUserManager = {
  events: {
    addUserLoaded: vi.fn(),
    addAccessTokenExpiring: vi.fn(),
  },
  signinSilentCallback: vi.fn(),
  signinSilent: vi.fn(),
  clearStaleState: vi.fn(),
  removeUser: vi.fn(),
  getUser: vi.fn(),
};

vi.mock('oidc-react', () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  useAuth: () => ({
    signIn: vi.fn(),
    userManager: mockUserManager,
  }),
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
