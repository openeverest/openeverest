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

import type { ResolvedExtensionMeta } from 'shared-types/extension-catalog.types';
import { buildHubLink, pickSpotlightProviders } from './no-providers.utils';
import { EXTENSION_HUB_WEBSITE_URL } from './no-providers.constants';

const provider = (
  name: string,
  overrides: Partial<ResolvedExtensionMeta> = {}
): ResolvedExtensionMeta => ({
  name,
  displayName: name,
  categories: ['database'],
  maturity: 'alpha',
  ...overrides,
});

const names = (providers: ResolvedExtensionMeta[]) =>
  providers.map((p) => p.name);

describe('pickSpotlightProviders', () => {
  it('takes one provider per primary category before repeating one', () => {
    const picked = pickSpotlightProviders(
      [
        provider('mysql'),
        provider('postgres'),
        provider('object-store', { categories: ['storage'] }),
        provider('model-serving', { categories: ['ai'] }),
      ],
      3
    );
    expect(names(picked)).toEqual(['mysql', 'object-store', 'model-serving']);
  });

  it('fills remaining slots in rank order when categories run out', () => {
    const picked = pickSpotlightProviders(
      [provider('mysql'), provider('postgres'), provider('mongo')],
      2
    );
    expect(names(picked)).toEqual(['mysql', 'postgres']);
  });

  it('prefers verified, then more mature providers, keeping catalog order otherwise', () => {
    const picked = pickSpotlightProviders(
      [
        provider('a-alpha', { categories: ['a'] }),
        provider('b-stable', { categories: ['b'], maturity: 'stable' }),
        provider('c-unknown', { categories: ['c'], maturity: undefined }),
        provider('d-verified', { categories: ['d'], verified: true }),
        provider('e-beta', { categories: ['e'], maturity: 'beta' }),
      ],
      5
    );
    expect(names(picked)).toEqual([
      'd-verified',
      'b-stable',
      'e-beta',
      'a-alpha',
      'c-unknown',
    ]);
  });

  it('never suggests deprecated providers', () => {
    const picked = pickSpotlightProviders(
      [
        provider('old', { maturity: 'deprecated', verified: true }),
        provider('new'),
      ],
      2
    );
    expect(names(picked)).toEqual(['new']);
  });

  it('does not reorder the caller array', () => {
    const providers = [provider('a'), provider('b', { verified: true })];
    pickSpotlightProviders(providers, 2);
    expect(names(providers)).toEqual(['a', 'b']);
  });
});

describe('buildHubLink', () => {
  it('routes into the plugin-hub UI when it is registered', () => {
    expect(buildHubLink(true, 'providers/valkey')).toEqual({
      to: '/plugins/plugin-hub/providers/valkey',
    });
  });

  it('falls back to the public Extension Hub otherwise', () => {
    expect(buildHubLink(false, 'providers/valkey')).toEqual({
      href: EXTENSION_HUB_WEBSITE_URL,
    });
  });
});
