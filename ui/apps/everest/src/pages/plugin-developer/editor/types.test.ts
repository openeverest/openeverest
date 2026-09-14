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

import { describe, it, expect } from 'vitest';
import { lineColToOffset } from './types';

describe('lineColToOffset', () => {
  const text = 'abc\ndef\nghi';

  it('maps line 1, column 1 to offset 0', () => {
    expect(lineColToOffset(text, 1, 1)).toBe(0);
  });

  it('maps a mid-document line/column to the right offset', () => {
    // line 2 ('def') starts at offset 4; column 2 -> offset 5
    expect(lineColToOffset(text, 2, 2)).toBe(5);
  });

  it('clamps a column past end-of-line to the line length', () => {
    // line 1 is 'abc' (len 3); column 99 clamps to offset 3
    expect(lineColToOffset(text, 1, 99)).toBe(3);
  });

  it('clamps a line past end-of-document to the last line', () => {
    // line 99 clamps to line 3 ('ghi' starts at offset 8); column 1 -> 8
    expect(lineColToOffset(text, 99, 1)).toBe(8);
  });

  it('never returns an offset beyond the text length', () => {
    expect(lineColToOffset(text, 99, 99)).toBe(text.length);
  });
});
