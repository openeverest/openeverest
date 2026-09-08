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

import { parseVersion, satisfiesHostVersion } from './plugin-version';

describe('parseVersion', () => {
  it('accepts the v-prefixed versions Everest actually reports', () => {
    // RELEASE_VERSION is v-prefixed and may carry a build suffix.
    expect(parseVersion('v2.1.3')).toEqual([2, 1, 3]);
    expect(parseVersion('v0.3.0-1-a93bef')).toEqual([0, 3, 0]);
    expect(parseVersion('2.1.3')).toEqual([2, 1, 3]);
  });

  it('returns null for values that are not versions', () => {
    expect(parseVersion('dev')).toBeNull();
    // `vite dev` serves index.html without running the Go template.
    expect(parseVersion('{{.EverestVersion}}')).toBeNull();
    expect(parseVersion('')).toBeNull();
  });
});

describe('satisfiesHostVersion', () => {
  it('permits a plugin that declares no range', () => {
    expect(satisfiesHostVersion('v2.1.0', undefined)).toBe(true);
    expect(satisfiesHostVersion('v2.1.0', '')).toBe(true);
  });

  it('never gates when the host version is unusable', () => {
    expect(satisfiesHostVersion('dev', '>=2.0.0')).toBe(true);
    expect(satisfiesHostVersion('{{.EverestVersion}}', '>=2.0.0')).toBe(true);
  });

  it('never gates on a local build (v0.0.0-<sha>)', () => {
    expect(satisfiesHostVersion('v0.0.0-a93bef', '>=2.0.0')).toBe(true);
  });

  it('accepts a host inside the declared range', () => {
    expect(satisfiesHostVersion('v2.1.0', '>=2.0.0 <3.0.0')).toBe(true);
    expect(satisfiesHostVersion('v2.1.0', '^2.0.0')).toBe(true);
    expect(satisfiesHostVersion('v2.1.4', '~2.1.0')).toBe(true);
  });

  it('rejects a host outside the declared range', () => {
    expect(satisfiesHostVersion('v3.0.0', '>=2.0.0 <3.0.0')).toBe(false);
    expect(satisfiesHostVersion('v1.9.0', '>=2.0.0')).toBe(false);
    expect(satisfiesHostVersion('v3.0.0', '^2.0.0')).toBe(false);
    expect(satisfiesHostVersion('v2.2.0', '~2.1.0')).toBe(false);
  });

  it('supports || alternatives', () => {
    expect(satisfiesHostVersion('v3.1.0', '^2.0.0 || ^3.0.0')).toBe(true);
    expect(satisfiesHostVersion('v4.0.0', '^2.0.0 || ^3.0.0')).toBe(false);
  });

  it('tolerates a v-prefixed range', () => {
    expect(satisfiesHostVersion('v2.1.0', '>=v2.0.0')).toBe(true);
    expect(satisfiesHostVersion('v1.0.0', '>=v2.0.0')).toBe(false);
  });
});
