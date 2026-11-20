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
import { jwtDecode } from 'jwt-decode';
import { api } from 'api/api';
import type { HttpApi } from '@generated/api-types';

// Axios/network errors carry the request config (including Authorization headers
// with bearer tokens), so dumping the whole object would leak credentials into the
// browser console.
const sanitizeError = (error: unknown): string => {
  if (error instanceof Error) {
    return error.stack || `${error.name}: ${error.message}`;
  }
  if (typeof error === 'string') {
    return error;
  }
  return 'Unknown error';
};

// Logs authentication-related failures to the browser console.
export const logAuthError = (context: string, error: unknown) => {
  // eslint-disable-next-line no-console
  console.error(`[auth] ${context}: ${sanitizeError(error)}`);
};

export const SSO_LOGIN_ERROR_KEY = 'ssoLoginError';

const SSO_EXCHANGE_MAX_RETRIES = 3;
const SSO_RENEW_LOCK_NAME = 'everest-sso-renew';
const MIN_REUSABLE_TOKEN_VALIDITY_MS = 60_000;

// Exchanges the OIDC ID token for an Everest-signed JWT via the RFC 8693 token-exchange grant.
// The ID token, unlike the access token, proves that the IdP issued it to Everest. The exchange
// shares a per-IP rate limit, so a throttled (429) attempt is retried with jittered backoff.
export const exchangeSsoToken = async (idToken: string): Promise<string> => {
  for (let attempt = 0; ; attempt++) {
    try {
      const response = await api.post<
        HttpApi.components['schemas']['AuthTokenResponse']
      >('/auth/token', {
        grant_type: 'urn:ietf:params:oauth:grant-type:token-exchange',
        subject_token: idToken,
        subject_token_type: 'urn:ietf:params:oauth:token-type:id_token',
      } satisfies HttpApi.components['schemas']['AuthTokenRequest']);
      return response.data.access_token;
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

// Serializes SSO renewals across all tabs of this origin, so tabs whose timers fire together
// renew once instead of each hitting the IdP and the exchange endpoint.
export const withSsoRenewLock = <T>(fn: () => Promise<T>): Promise<T> =>
  'locks' in navigator
    ? navigator.locks.request(SSO_RENEW_LOCK_NAME, fn)
    : fn();

// True if the Everest token is valid long enough to be reused instead of renewed.
export const isReusableToken = (token: string | null): token is string => {
  if (!token) {
    return false;
  }
  try {
    const { exp } = jwtDecode(token);
    return !!exp && exp * 1000 - Date.now() > MIN_REUSABLE_TOKEN_VALIDITY_MS;
  } catch {
    return false;
  }
};

export const isRunningInIframe = (): boolean => {
  try {
    return window.self !== window.top;
  } catch {
    return true;
  }
};
