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

import { jwtDecode } from 'jwt-decode';
import { exchangeSsoToken } from 'api/session';
import { EVEREST_JWT_ISSUER } from 'consts';
import { EverestJwtPayload } from './auth.context.types';

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

const SSO_RENEW_LOCK_NAME = 'everest-sso-renew';
const MIN_REUSABLE_TOKEN_VALIDITY_MS = 60_000;

export const exchangeAndStoreSsoToken = async (idpAccessToken: string) => {
  const everestToken = await exchangeSsoToken(idpAccessToken);
  localStorage.setItem('everestToken', everestToken);
  return everestToken;
};

// Built-in subs are "<user>:<capability>"; split like the server's rbac.GetUser.
export const getUsernameFromSub = (sub = '') => sub.split(':')[0];

// Exchanged SSO tokens carry oidc_issuer; legacy SSO sessions still hold the raw IdP token.
export const isSsoSessionToken = (token: string) => {
  try {
    const payload = jwtDecode<EverestJwtPayload>(token);
    return !!payload.oidc_issuer || payload.iss !== EVEREST_JWT_ISSUER;
  } catch {
    return false;
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
