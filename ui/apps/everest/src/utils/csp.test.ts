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

import { getCspNonce } from './csp';

describe('getCspNonce', () => {
  afterEach(() => {
    // Clean up any meta tags added during tests
    document
      .querySelectorAll('meta[name="csp-nonce"]')
      .forEach((el) => el.remove());
  });

  it('should return empty string when no meta tag exists', () => {
    expect(getCspNonce()).toBe('');
  });

  it('should return the nonce value from the meta tag', () => {
    const meta = document.createElement('meta');
    meta.setAttribute('name', 'csp-nonce');
    meta.setAttribute('content', 'test-nonce-abc123');
    document.head.appendChild(meta);

    expect(getCspNonce()).toBe('test-nonce-abc123');
  });

  it('should return empty string when meta tag has no content attribute', () => {
    const meta = document.createElement('meta');
    meta.setAttribute('name', 'csp-nonce');
    document.head.appendChild(meta);

    expect(getCspNonce()).toBe('');
  });
});
