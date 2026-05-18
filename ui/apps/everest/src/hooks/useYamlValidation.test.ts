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

import { renderHook } from '@testing-library/react';
import { useYamlValidation } from 'hooks/useYamlValidation';

describe('useYamlValidation', () => {
  it('should return null parsed and no error for empty content', () => {
    const { result } = renderHook(() => useYamlValidation(''));

    expect(result.current.parsed).toBeNull();
    expect(result.current.error).toBeNull();
  });

  it('should return null parsed and no error for whitespace-only content', () => {
    const { result } = renderHook(() => useYamlValidation('   \n  '));

    expect(result.current.parsed).toBeNull();
    expect(result.current.error).toBeNull();
  });

  it('should parse valid YAML with scalar value', () => {
    const { result } = renderHook(() => useYamlValidation('name: test'));

    expect(result.current.parsed).toEqual({ name: 'test' });
    expect(result.current.error).toBeNull();
  });

  it('should parse valid YAML with nested structure', () => {
    const yaml = `
apiVersion: v1
metadata:
  name: my-db
  namespace: everest
spec:
  replicas: 3
`;
    const { result } = renderHook(() => useYamlValidation(yaml));

    expect(result.current.parsed).toEqual({
      apiVersion: 'v1',
      metadata: { name: 'my-db', namespace: 'everest' },
      spec: { replicas: 3 },
    });
    expect(result.current.error).toBeNull();
  });

  it('should return error for invalid YAML', () => {
    const invalidYaml = `
key: value
  bad_indent: oops
`;
    const { result } = renderHook(() => useYamlValidation(invalidYaml));

    expect(result.current.parsed).toBeNull();
    expect(result.current.error).toBeTruthy();
    expect(typeof result.current.error).toBe('string');
  });

  it('should parse YAML with array values', () => {
    const yaml = `
items:
  - name: first
  - name: second
`;
    const { result } = renderHook(() => useYamlValidation(yaml));

    expect(result.current.parsed).toEqual({
      items: [{ name: 'first' }, { name: 'second' }],
    });
    expect(result.current.error).toBeNull();
  });

  it('should handle YAML comments gracefully', () => {
    const yaml = `
# This is a comment
name: test  # inline comment
`;
    const { result } = renderHook(() => useYamlValidation(yaml));

    expect(result.current.parsed).toEqual({ name: 'test' });
    expect(result.current.error).toBeNull();
  });

  it('should update when content changes', () => {
    const { result, rerender } = renderHook(
      ({ content }) => useYamlValidation(content),
      { initialProps: { content: 'a: 1' } }
    );

    expect(result.current.parsed).toEqual({ a: 1 });

    rerender({ content: 'b: 2' });

    expect(result.current.parsed).toEqual({ b: 2 });
    expect(result.current.error).toBeNull();
  });
});
