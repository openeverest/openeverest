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

export const isRunningInIframe = (): boolean => {
  try {
    return window.self !== window.top;
  } catch {
    return true;
  }
};
