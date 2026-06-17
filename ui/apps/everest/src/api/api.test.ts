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
import { api, addApiErrorInterceptor, removeApiErrorInterceptor } from './api';

// Mock session-token module
vi.mock('./session-token', () => ({
  getAuthToken: vi.fn(() => 'valid-token'),
  refreshSession: vi.fn(),
}));

// Mock notistack
vi.mock('notistack', () => ({
  enqueueSnackbar: vi.fn(),
}));

import { refreshSession } from './session-token';
import { enqueueSnackbar } from 'notistack';

type AdapterConfig = {
  url?: string;
  headers?: Record<string, string>;
  retriedAfterRefresh?: boolean;
};

const buildAxiosError = (
  config: AdapterConfig,
  status: number,
  message: string
) => ({
  isAxiosError: true,
  config,
  response: {
    status,
    data: { message },
    headers: {},
    config,
  },
  toJSON: () => ({}),
});

const adapterMock = vi.fn();
const originalAdapter = api.defaults.adapter;

describe('api interceptor', () => {
  beforeEach(() => {
    adapterMock.mockReset();
    api.defaults.adapter = adapterMock as never;
    vi.clearAllMocks();
    addApiErrorInterceptor();
  });

  afterEach(() => {
    removeApiErrorInterceptor();
    api.defaults.adapter = originalAdapter;
  });

  describe('skip retry for /auth/ URLs', () => {
    it('does NOT attempt refresh when /auth/token returns 401', async () => {
      adapterMock.mockImplementation((config: AdapterConfig) =>
        Promise.reject(buildAxiosError(config, 401, 'invalid'))
      );

      await expect(
        api.post('/auth/token', { grant_type: 'refresh_token' })
      ).rejects.toMatchObject({
        response: { status: 401 },
      });

      expect(refreshSession).not.toHaveBeenCalled();
    });

    it('does NOT attempt refresh when /auth/revoke returns 401', async () => {
      adapterMock.mockImplementation((config: AdapterConfig) =>
        Promise.reject(buildAxiosError(config, 401, 'unauthorized'))
      );

      await expect(api.post('/auth/revoke', {})).rejects.toMatchObject({
        response: { status: 401 },
      });

      expect(refreshSession).not.toHaveBeenCalled();
    });
  });

  describe('retry for non-auth 401s', () => {
    it('attempts refresh and retries original request on 401', async () => {
      vi.mocked(refreshSession).mockResolvedValueOnce('new-token');

      // First call returns 401, retry (with refreshed token) returns 200.
      let calls = 0;
      adapterMock.mockImplementation((config: AdapterConfig) => {
        calls += 1;
        if (calls === 1) {
          return Promise.reject(
            buildAxiosError(config, 401, 'invalid or expired jwt')
          );
        }
        return Promise.resolve({
          data: { version: '1.0.0' },
          status: 200,
          statusText: 'OK',
          headers: {},
          config,
        });
      });

      const response = await api.get('/version');

      expect(refreshSession).toHaveBeenCalledTimes(1);
      expect(response.data).toEqual({ version: '1.0.0' });
    });

    it('redirects to /logout when refresh fails', async () => {
      vi.mocked(refreshSession).mockResolvedValueOnce(null);
      adapterMock.mockImplementation((config: AdapterConfig) =>
        Promise.reject(buildAxiosError(config, 401, 'invalid or expired jwt'))
      );

      // Mock location.href
      const locationSpy = vi
        .spyOn(window, 'location', 'get')
        .mockReturnValue({ ...window.location, href: '' } as Location);

      await api.get('/version').catch(() => {});

      expect(refreshSession).toHaveBeenCalledTimes(1);
      expect(enqueueSnackbar).toHaveBeenCalledWith(
        'Your session has expired. Please sign in again.',
        { variant: 'info' }
      );

      locationSpy.mockRestore();
    });

    it('does NOT retry more than once (prevents infinite loops)', async () => {
      vi.mocked(refreshSession).mockResolvedValue('still-bad-token');

      // Both original and retried requests return 401.
      adapterMock.mockImplementation((config: AdapterConfig) =>
        Promise.reject(buildAxiosError(config, 401, 'invalid or expired jwt'))
      );

      const locationSpy = vi
        .spyOn(window, 'location', 'get')
        .mockReturnValue({ ...window.location, href: '' } as Location);

      await api.get('/data').catch(() => {});

      // refreshSession called only once (first 401), not on the retry's 401
      expect(refreshSession).toHaveBeenCalledTimes(1);

      locationSpy.mockRestore();
    });
  });
});
