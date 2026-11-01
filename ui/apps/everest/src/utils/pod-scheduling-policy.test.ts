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
import { Provider } from 'shared-types/api.types';
import {
  deriveSchedulingSupport,
  resolveAffinityTargets,
} from './pod-scheduling-policy';

// Builds a component `supportedFields` that declares the given schedulingPolicy
// sub-fields. No fields → schedulingPolicy declared as a whole.
const declaresScheduling = (
  fields?: string[]
): { openAPIV3Schema: unknown } => ({
  openAPIV3Schema: {
    properties: {
      schedulingPolicy: fields
        ? { properties: Object.fromEntries(fields.map((f) => [f, {}])) }
        : {},
    },
  },
});

const provider = (
  components: NonNullable<
    NonNullable<Provider['spec']['topologies']>[string]['components']
  >,
  topology = 'main'
): Provider => ({
  spec: { topologies: { [topology]: { components } } },
});

const allTypes = [
  'affinity',
  'nodeSelector',
  'tolerations',
  'schedulerName',
  'topologySpreadConstraints',
];

describe('deriveSchedulingSupport', () => {
  it('returns {} when the provider is undefined', () => {
    expect(deriveSchedulingSupport(undefined, 'main')).toEqual({});
  });

  it('returns {} when the topology is undefined or unknown', () => {
    const p = provider({ engine: {} });
    expect(deriveSchedulingSupport(p, undefined)).toEqual({});
    expect(deriveSchedulingSupport(p, 'nope')).toEqual({});
  });

  it('treats a component with no supportedFields as supporting every type', () => {
    const result = deriveSchedulingSupport(provider({ engine: {} }), 'main');
    expect(result.engine).toEqual(allTypes);
  });

  it('narrows support to the declared schedulingPolicy sub-fields', () => {
    const result = deriveSchedulingSupport(
      provider({
        engine: { supportedFields: declaresScheduling(['affinity']) },
      }),
      'main'
    );
    expect(result.engine).toEqual(['affinity']);
  });

  it('keeps the CRD field names and normalizes their order', () => {
    const result = deriveSchedulingSupport(
      provider({
        engine: {
          supportedFields: declaresScheduling([
            'topologySpreadConstraints',
            'affinity',
          ]),
        },
      }),
      'main'
    );
    expect(result.engine).toEqual(['affinity', 'topologySpreadConstraints']);
  });

  it('supports every type when schedulingPolicy is declared without nested properties', () => {
    const result = deriveSchedulingSupport(
      provider({ engine: { supportedFields: declaresScheduling() } }),
      'main'
    );
    expect(result.engine).toEqual(allTypes);
  });

  it('omits a component that declares supportedFields without schedulingPolicy', () => {
    const result = deriveSchedulingSupport(
      provider({
        engine: {
          supportedFields: {
            openAPIV3Schema: { properties: { resources: {} } },
          },
        },
      }),
      'main'
    );
    expect(result).toEqual({});
  });

  it('omits a component whose supportedFields has no usable schema', () => {
    const result = deriveSchedulingSupport(
      provider({ engine: { supportedFields: { openAPIV3Schema: undefined } } }),
      'main'
    );
    expect(result).toEqual({});
  });

  it('builds a per-component matrix for a sharded-like topology', () => {
    const result = deriveSchedulingSupport(
      provider({
        engine: {
          supportedFields: declaresScheduling([
            'affinity',
            'tolerations',
            'nodeSelector',
          ]),
        },
        proxy: {
          supportedFields: declaresScheduling(['affinity', 'tolerations']),
        },
        configServer: {
          supportedFields: declaresScheduling(['affinity']),
        },
      }),
      'main'
    );
    expect(result).toEqual({
      engine: ['affinity', 'nodeSelector', 'tolerations'],
      proxy: ['affinity', 'tolerations'],
      configServer: ['affinity'],
    });
  });
});

describe('resolveAffinityTargets', () => {
  it('returns one affinity path per component that accepts affinity', () => {
    const p = provider({
      engine: { supportedFields: declaresScheduling(['affinity']) },
      proxy: { supportedFields: declaresScheduling(['tolerations']) },
    });

    expect(resolveAffinityTargets(p, 'main')).toEqual([
      {
        key: 'engine',
        path: 'spec.components.engine.schedulingPolicy.affinity',
      },
    ]);
  });

  it('returns no targets without a provider', () => {
    expect(resolveAffinityTargets(undefined, 'main')).toEqual([]);
  });
});
