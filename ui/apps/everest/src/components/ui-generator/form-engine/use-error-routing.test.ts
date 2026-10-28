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
import { flattenErrorPaths } from './use-error-routing';

const leaf = { message: 'required', type: 'required' };

describe('flattenErrorPaths', () => {
  it('collects nested leaf paths', () => {
    expect(flattenErrorPaths({ spec: { name: leaf } })).toEqual(['spec.name']);
  });

  it('keeps hyphenated field ids intact', () => {
    expect(flattenErrorPaths({ spec: { 'g-replicas': leaf } })).toEqual([
      'spec.g-replicas',
    ]);
  });

  it('keeps encoded qualified-name segments intact', () => {
    expect(
      flattenErrorPaths({ spec: { limits: { 'nvidia%2Ecom%2Fgpu': leaf } } })
    ).toEqual(['spec.limits.nvidia%2Ecom%2Fgpu']);
  });
});
