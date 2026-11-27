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

import { describe, expect, it } from 'vitest';
import { keyValueRow } from 'components/ui-generator/test-fixtures/key-value-row.fixture';
import { toRowAtIndex } from './map-field.utils';

describe('toRowAtIndex', () => {
  it('focuses the input a new row draws first, which componentsOrder decides', () => {
    const row = { ...keyValueRow(), componentsOrder: ['value', 'key'] };

    expect(
      toRowAtIndex(row, 0, { disabled: false, focus: true }).components
    ).toMatchObject({
      key: { _renderOverrides: { autoFocus: false } },
      value: { _renderOverrides: { autoFocus: true } },
    });
  });
});
