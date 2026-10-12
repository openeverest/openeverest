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

import { AxiosError } from 'axios';
import { api } from './api';

const SSO_EXCHANGE_MAX_RETRIES = 3;

// Exchanges an IdP access token (JWT or opaque) for an Everest JWT. The endpoint is rate limited
// per IP, so a throttled (429) attempt is retried with jittered backoff.
export const exchangeSsoToken = async (idpAccessToken: string) => {
  for (let attempt = 0; ; attempt++) {
    try {
      const response = await api.post<{ token: string }>(
        '/session/sso',
        { token: idpAccessToken },
        { skipAuthRefresh: true }
      );
      return response.data.token;
    } catch (error) {
      const throttled =
        error instanceof AxiosError && error.response?.status === 429;
      if (!throttled || attempt >= SSO_EXCHANGE_MAX_RETRIES) {
        throw error;
      }
      const delayMs = 500 * 2 ** attempt + Math.random() * 500;
      await new Promise((resolve) => setTimeout(resolve, delayMs));
    }
  }
};
