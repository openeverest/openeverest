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

import { getValueByPath } from '../get-value-by-path';

describe('getValueByPath', () => {
  it('returns nested value for a valid path', () => {
    const input = {
      spec: {
        settings: {
          replicas: 3,
        },
      },
    };

    expect(getValueByPath(input, 'spec.settings.replicas')).toBe(3);
  });

  it('returns undefined for missing path', () => {
    const input = { spec: { settings: {} } };

    expect(getValueByPath(input, 'spec.settings.replicas')).toBeUndefined();
  });

  it('returns undefined for invalid object/path', () => {
    expect(getValueByPath(null, 'spec.value')).toBeUndefined();
    expect(getValueByPath({ spec: { value: 1 } }, '')).toBeUndefined();
    expect(getValueByPath('not-object', 'spec.value')).toBeUndefined();
  });

  it('accepts multipath arrays and resolves value by the first valid path', () => {
    const input = {
      spec: {
        engine: { version: '8.0.41' },
        proxy: { version: '8.0.39' },
      },
    };

    expect(
      getValueByPath(input, ['spec.engine.version', 'spec.proxy.version'])
    ).toBe('8.0.41');
  });
});
