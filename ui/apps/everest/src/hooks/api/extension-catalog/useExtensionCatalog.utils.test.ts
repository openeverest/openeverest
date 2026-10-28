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

import type { RawExtensionIndex } from 'shared-types/extension-catalog.types';
import { buildExtensionCatalog } from './useExtensionCatalog.utils';

describe('buildExtensionCatalog', () => {
  it('lists each catalog provider once and skips non-provider entries', () => {
    const index: RawExtensionIndex = {
      extensions: [
        {
          name: 'valkey',
          type: 'provider',
          provider: { providerName: 'provider-valkey' },
        },
        { name: 'plugin-metrics', type: 'plugin' },
        { name: 'valkey', type: 'provider', displayName: 'Valkey' },
        { type: 'provider' },
      ],
    };

    const { providers, providerMeta } = buildExtensionCatalog(index, 'main');

    expect(providers.map((p) => p.displayName)).toEqual(['Valkey']);
    expect(providerMeta.get('provider-valkey')?.name).toBe('valkey');
  });

  it('returns an empty catalog when the hub is unavailable', () => {
    const { providers, providerMeta } = buildExtensionCatalog(null, 'main');
    expect(providers).toEqual([]);
    expect(providerMeta.size).toBe(0);
  });
});
