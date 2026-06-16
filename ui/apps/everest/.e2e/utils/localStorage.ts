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

import { request } from '@playwright/test';

const BASE_URL = process.env.EVEREST_URL || 'http://localhost:8080';

const { CI_USER, CI_PASSWORD, SESSION_USER, SESSION_PASS } = process.env;

type CachedToken = {
  token: string;
  expiresAt: number; // epoch ms
};

const tokenCache: Record<string, CachedToken> = {};

// The UI no longer persists the access token in localStorage: it is kept in
// memory only, while the refresh token lives in an HttpOnly cookie. Tests that
// need a token for direct API calls obtain one through the API instead.
const MAX_RETRIES = 3;
const RETRY_DELAY_MS = 1500;

const getApiToken = async (user: string, password: string) => {
  const cached = tokenCache[user];
  if (cached && cached.expiresAt - Date.now() > 60 * 1000) {
    return cached.token;
  }

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    const ctx = await request.newContext({ baseURL: BASE_URL });
    const resp = await ctx.post('/v1/auth/token', {
      data: {
        grant_type: 'password',
        username: user,
        password,
      },
    });

    if (resp.status() === 429 && attempt < MAX_RETRIES) {
      await ctx.dispose();
      await new Promise((r) => setTimeout(r, RETRY_DELAY_MS * (attempt + 1)));
      continue;
    }

    if (!resp.ok()) {
      await ctx.dispose();
      throw new Error(
        `Failed to obtain API token for user ${user}: ${resp.status()}`
      );
    }
    const body = await resp.json();
    await ctx.dispose();

    tokenCache[user] = {
      token: body.access_token,
      expiresAt: Date.now() + body.expires_in * 1000,
    };
    return body.access_token as string;
  }

  throw new Error(
    `Failed to obtain API token for user ${user} after ${MAX_RETRIES} retries`
  );
};

export const getCITokenFromLocalStorage = async () => {
  return getApiToken(CI_USER!, CI_PASSWORD!);
};

export const getSessionTokenFromLocalStorage = async () => {
  return getApiToken(SESSION_USER!, SESSION_PASS!);
};
