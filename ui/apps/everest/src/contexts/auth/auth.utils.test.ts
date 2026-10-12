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

import { isReusableToken } from './auth.utils';

// Unsigned JWT; isReusableToken only decodes the payload.
const tokenExpiringIn = (seconds: number) =>
  [
    btoa(JSON.stringify({ alg: 'none' })),
    btoa(JSON.stringify({ exp: Math.floor(Date.now() / 1000) + seconds })),
    '',
  ].join('.');

describe('isReusableToken', () => {
  it('accepts a token with time left', () => {
    expect(isReusableToken(tokenExpiringIn(600))).toBe(true);
  });

  it('rejects a token that is about to expire', () => {
    expect(isReusableToken(tokenExpiringIn(30))).toBe(false);
  });

  it('rejects missing and malformed tokens', () => {
    expect(isReusableToken(null)).toBe(false);
    expect(isReusableToken('not-a-jwt')).toBe(false);
  });
});
