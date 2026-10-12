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

import { AxiosError, AxiosHeaders } from 'axios';
import { api } from './api';
import { exchangeSsoToken } from './session';

const axiosError = (status: number) =>
  new AxiosError('request failed', String(status), undefined, undefined, {
    status,
    statusText: '',
    data: {},
    headers: {},
    config: { headers: new AxiosHeaders() },
  });

describe('exchangeSsoToken', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('retries a throttled exchange and returns the token', async () => {
    const post = vi
      .spyOn(api, 'post')
      .mockRejectedValueOnce(axiosError(429))
      .mockResolvedValueOnce({ data: { token: 'everest-jwt' } });

    const result = exchangeSsoToken('oidc-token');
    await vi.runAllTimersAsync();

    await expect(result).resolves.toBe('everest-jwt');
    expect(post).toHaveBeenCalledTimes(2);
  });

  it('gives up after the retry budget', async () => {
    const post = vi.spyOn(api, 'post').mockRejectedValue(axiosError(429));

    const result = exchangeSsoToken('oidc-token');
    const assertion = expect(result).rejects.toBeInstanceOf(AxiosError);
    await vi.runAllTimersAsync();

    await assertion;
    expect(post).toHaveBeenCalledTimes(4);
  });

  it('does not retry an invalid token', async () => {
    const post = vi.spyOn(api, 'post').mockRejectedValue(axiosError(401));

    await expect(exchangeSsoToken('oidc-token')).rejects.toBeInstanceOf(
      AxiosError
    );
    expect(post).toHaveBeenCalledTimes(1);
  });
});
