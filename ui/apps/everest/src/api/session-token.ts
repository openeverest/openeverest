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

// Session token management for internal (username/password) logins.
//
// The short-lived access JWT is kept in memory only (never persisted), limiting
// exposure to XSS. The long-lived refresh token is carried by an HttpOnly cookie
// set by the API server (never accessible to JS). On page reload the access
// token is recovered with a silent refresh against /v1/auth/token.
//
// OIDC (SSO) sessions are unaffected: they keep using the 'everestToken'
// localStorage entry managed by oidc-react.

import axios from 'axios';
import type { HttpApi } from '@generated/api-types';

type AuthTokenRequest = HttpApi.components['schemas']['AuthTokenRequest'];
type AuthTokenResponse = HttpApi.components['schemas']['AuthTokenResponse'];

// Bare client without interceptors to avoid recursion from the api module's
// 401-handling interceptor.
const authClient = axios.create({ baseURL: '/v1/', withCredentials: true });

let accessToken: string | null = null;
let accessTokenExpiry: number | null = null; // epoch ms

export const setAccessToken = (token: string, expiresInSeconds: number) => {
  accessToken = token;
  accessTokenExpiry = Date.now() + expiresInSeconds * 1000;
};

export const getAccessToken = () => accessToken;

export const getAccessTokenExpiry = () => accessTokenExpiry;

export const clearAccessToken = () => {
  accessToken = null;
  accessTokenExpiry = null;
};

// Returns the credential for API requests: the in-memory access token for
// internal sessions, falling back to localStorage for OIDC sessions.
// TODO: Remove localStorage fallback once OIDC is migrated to the same
// cookie-based token mechanism (requires backend support).
export const getAuthToken = () =>
  accessToken || localStorage.getItem('everestToken');

let refreshPromise: Promise<string | null> | null = null;

// Exchanges the refresh token cookie for a new token pair (single-flight: all
// concurrent callers share one request). The refresh token is rotated by the
// server. Returns the new access token, or null if the session cannot be
// refreshed.
export const refreshSession = (): Promise<string | null> => {
  if (!refreshPromise) {
    const payload: AuthTokenRequest = {
      grant_type: 'refresh_token',
      refresh_token_delivery: 'cookie',
    };
    refreshPromise = authClient
      .post<AuthTokenResponse>('/auth/token', payload)
      .then(({ data }) => {
        setAccessToken(data.access_token, data.expires_in);
        return data.access_token;
      })
      .catch(() => {
        clearAccessToken();
        return null;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
};
