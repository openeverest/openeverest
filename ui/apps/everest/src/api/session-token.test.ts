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

const mockPost = vi.fn();

vi.mock('axios', () => ({
  default: {
    create: () => ({ post: mockPost }),
  },
}));

// Import after mocking so that the module picks up our mocked axios.create
const { refreshSession, setAccessToken, getAccessToken, clearAccessToken } =
  await import('./session-token');

describe('session-token', () => {
  beforeEach(() => {
    mockPost.mockReset();
    clearAccessToken();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('refreshSession — single-flight', () => {
    it('deduplicates concurrent calls into a single network request', async () => {
      mockPost.mockResolvedValueOnce({
        data: {
          access_token: 'new-token',
          token_type: 'Bearer',
          expires_in: 900,
        },
      });

      // Fire 5 concurrent refreshes
      const results = await Promise.all([
        refreshSession(),
        refreshSession(),
        refreshSession(),
        refreshSession(),
        refreshSession(),
      ]);

      // All should resolve to the same token
      results.forEach((token) => {
        expect(token).toBe('new-token');
      });

      // Only 1 actual network request was made
      expect(mockPost).toHaveBeenCalledTimes(1);
    });

    it('stores the new access token after successful refresh', async () => {
      mockPost.mockResolvedValueOnce({
        data: {
          access_token: 'refreshed-token',
          token_type: 'Bearer',
          expires_in: 900,
        },
      });

      await refreshSession();

      expect(getAccessToken()).toBe('refreshed-token');
    });

    it('returns null and clears token on refresh failure', async () => {
      setAccessToken('old-token', 900);
      mockPost.mockRejectedValueOnce(new Error('401'));

      const result = await refreshSession();

      expect(result).toBeNull();
      expect(getAccessToken()).toBeNull();
    });

    it('allows a new request after the previous one completes', async () => {
      mockPost
        .mockResolvedValueOnce({
          data: {
            access_token: 'first',
            token_type: 'Bearer',
            expires_in: 900,
          },
        })
        .mockResolvedValueOnce({
          data: {
            access_token: 'second',
            token_type: 'Bearer',
            expires_in: 900,
          },
        });

      const first = await refreshSession();
      expect(first).toBe('first');

      const second = await refreshSession();
      expect(second).toBe('second');

      expect(mockPost).toHaveBeenCalledTimes(2);
    });
  });
});
