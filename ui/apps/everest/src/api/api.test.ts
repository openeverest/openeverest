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

import { AxiosAdapter, AxiosError, AxiosResponse } from 'axios';
import {
  addApiAuthInterceptor,
  addApiErrorInterceptor,
  api,
  setTokenRefresher,
} from './api';

const navigations: string[] = [];
let acceptedToken = 'renewed';

const fakeServer: AxiosAdapter = async (config) => {
  const accepted =
    config.headers['Authorization'] === `Bearer ${acceptedToken}`;
  const response: AxiosResponse = {
    data: accepted ? {} : { message: 'invalid or expired jwt' },
    status: accepted ? 200 : 401,
    statusText: '',
    headers: {},
    config,
  };
  if (accepted) {
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

const renewingRefresher = () =>
  vi.fn(
    () =>
      new Promise<string>((resolve) =>
        setTimeout(() => {
          localStorage.setItem('everestToken', 'renewed');
          resolve('renewed');
        })
      )
  );

describe('api 401 handling', () => {
  beforeEach(() => {
    acceptedToken = 'renewed';
    navigations.length = 0;
    localStorage.setItem('everestToken', 'stale');
    addApiErrorInterceptor();
    addApiAuthInterceptor();
    api.defaults.adapter = fakeServer;
    vi.stubGlobal('location', {
      set href(url: string) {
        navigations.push(url);
      },
    });
  });

  afterEach(() => {
    setTokenRefresher(null);
    vi.unstubAllGlobals();
  });

  it('runs one renewal for concurrent 401s and passes the rejected token', async () => {
    const refresher = renewingRefresher();
    setTokenRefresher(refresher);

    await Promise.all([api.get('/a'), api.get('/b')]);

    expect(refresher).toHaveBeenCalledTimes(1);
    expect(refresher).toHaveBeenCalledWith('stale');
    expect(navigations).toEqual([]);
  });

  it('logs out instead of renewing again when the replayed request is rejected', async () => {
    acceptedToken = 'never';
    const refresher = renewingRefresher();
    setTokenRefresher(refresher);

    await api.get('/a');

    expect(refresher).toHaveBeenCalledTimes(1);
    expect(navigations).toEqual(['/logout']);
  });

  it('never renews requests marked with skipAuthRefresh', async () => {
    const refresher = renewingRefresher();
    setTokenRefresher(refresher);

    await api.post('/session/sso', {}, { skipAuthRefresh: true });

    expect(refresher).not.toHaveBeenCalled();
    expect(navigations).toEqual(['/logout']);
  });
});
