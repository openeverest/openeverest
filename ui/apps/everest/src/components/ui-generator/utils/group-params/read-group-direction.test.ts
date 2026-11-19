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
import { GroupDirection } from 'components/ui-generator/ui-generator.types';
import { readGroupDirection } from './read-group-direction';

describe('readGroupDirection', () => {
  it('defaults to column when groupParams are omitted', () => {
    expect(readGroupDirection()).toBe(GroupDirection.Column);
  });

  it('defaults to column when direction is missing or invalid', () => {
    expect(readGroupDirection({})).toBe(GroupDirection.Column);
    expect(readGroupDirection({ direction: 'horizontal' })).toBe(
      GroupDirection.Column
    );
  });

  it('returns row when direction is row', () => {
    expect(readGroupDirection({ direction: GroupDirection.Row })).toBe(
      GroupDirection.Row
    );
  });
});
