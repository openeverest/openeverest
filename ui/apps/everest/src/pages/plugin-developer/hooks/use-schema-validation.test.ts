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
import { renderHook } from '@testing-library/react';
import { useSchemaValidation } from './use-schema-validation';

const VALID_YAML = [
  'standalone:',
  '  sections:',
  '    basics:',
  '      components:',
  '        nodes:',
  '          uiType: number',
  '          path: x',
  '',
].join('\n');

describe('useSchemaValidation', () => {
  it('returns a parsed schema and no diagnostics for valid YAML', () => {
    const { result } = renderHook(() => useSchemaValidation(VALID_YAML));

    expect(result.current.parsed).not.toBeNull();
    expect(result.current.diagnostics).toEqual([]);
  });

  it('returns diagnostics and a null parse for invalid YAML', () => {
    const { result } = renderHook(() => useSchemaValidation('just a string'));

    expect(result.current.parsed).toBeNull();
    expect(result.current.diagnostics.length).toBeGreaterThan(0);
  });

  it('memoizes the result while the text is unchanged', () => {
    const { result, rerender } = renderHook(
      ({ text }) => useSchemaValidation(text),
      { initialProps: { text: VALID_YAML } }
    );
    const first = result.current;

    rerender({ text: VALID_YAML });
    // Same input -> same reference, so consumers don't re-render.
    expect(result.current).toBe(first);
  });

  it('recomputes when the text changes', () => {
    const { result, rerender } = renderHook(
      ({ text }) => useSchemaValidation(text),
      { initialProps: { text: VALID_YAML } }
    );
    const first = result.current;

    rerender({ text: 'just a string' });
    expect(result.current).not.toBe(first);
    expect(result.current.parsed).toBeNull();
  });
});
