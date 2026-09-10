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
import { renderHook, act } from '@testing-library/react';
import { useTopology } from './use-topology';
import { TopologyUISchemas } from '../ui-generator.types';

// The hook only reads top-level keys, so empty bodies suffice.
const schemaOf = (...keys: string[]): TopologyUISchemas =>
  Object.fromEntries(keys.map((k) => [k, {}])) as TopologyUISchemas;

describe('useTopology', () => {
  it('defaults the selection to the first topology', () => {
    const { result } = renderHook(() =>
      useTopology(schemaOf('replicaSet', 'sharded'))
    );

    expect(result.current.topologies).toEqual(['replicaSet', 'sharded']);
    expect(result.current.selectedTopology).toBe('replicaSet');
    expect(result.current.hasMultipleTopologies).toBe(true);
  });

  it('reports a single topology as not multiple', () => {
    const { result } = renderHook(() => useTopology(schemaOf('standalone')));

    expect(result.current.hasMultipleTopologies).toBe(false);
    expect(result.current.selectedTopology).toBe('standalone');
  });

  it('lets the caller select another existing topology', () => {
    const { result } = renderHook(() =>
      useTopology(schemaOf('replicaSet', 'sharded'))
    );

    act(() => result.current.setSelectedTopology('sharded'));
    expect(result.current.selectedTopology).toBe('sharded');
  });

  it('falls back to the first topology when the schema no longer contains the selection', () => {
    const { result, rerender } = renderHook(
      ({ schema }) => useTopology(schema),
      {
        initialProps: { schema: schemaOf('replicaSet', 'sharded') },
      }
    );
    act(() => result.current.setSelectedTopology('sharded'));

    // Live edit swaps the schema for one without 'sharded' (or 'replicaSet').
    rerender({ schema: schemaOf('standalone') });

    // Stale 'sharded' would render an empty step; it must fall back instead.
    expect(result.current.selectedTopology).toBe('standalone');
  });

  it('keeps the selection when the new schema still contains it', () => {
    const { result, rerender } = renderHook(
      ({ schema }) => useTopology(schema),
      {
        initialProps: { schema: schemaOf('replicaSet', 'sharded') },
      }
    );
    act(() => result.current.setSelectedTopology('sharded'));

    rerender({ schema: schemaOf('sharded', 'other') });

    expect(result.current.selectedTopology).toBe('sharded');
  });
});
