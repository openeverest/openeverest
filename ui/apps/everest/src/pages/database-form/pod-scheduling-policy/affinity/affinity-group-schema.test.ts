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
import {
  AffinityOperator,
  AffinityPriority,
  AffinityType,
} from 'shared-types/affinity.types';
import { affinityGroupSchema } from './affinity-group-schema';

const issuePaths = (input: unknown): string[] => {
  const result = affinityGroupSchema.safeParse(input);
  return result.success ? [] : result.error.issues.map((i) => i.path.join('.'));
};

describe('affinityGroupSchema', () => {
  it('accepts a valid required node-affinity group', () => {
    const result = affinityGroupSchema.safeParse({
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Required,
      conditions: [
        { key: 'disktype', operator: AffinityOperator.In, values: ['ssd'] },
      ],
    });
    expect(result.success).toBe(true);
  });

  it('flags a node condition missing key and operator', () => {
    const paths = issuePaths({
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Required,
      conditions: [{ values: [] }],
    });
    expect(paths).toContain('conditions.0.key');
    expect(paths).toContain('conditions.0.operator');
  });

  it('requires values when the operator consumes them (In/NotIn)', () => {
    const paths = issuePaths({
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Required,
      conditions: [{ key: 'disktype', operator: AffinityOperator.In }],
    });
    expect(paths).toContain('conditions.0.values');
  });

  it('rejects values that break the k8s label-value rules', () => {
    const result = affinityGroupSchema.safeParse({
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Required,
      conditions: [
        {
          key: 'disktype',
          operator: AffinityOperator.In,
          values: ['bad value!'],
        },
      ],
    });
    expect(result.success).toBe(false);
    expect(
      !result.success &&
        result.error.issues.some(
          (i) => i.path.join('.') === 'conditions.0.values'
        )
    ).toBe(true);
  });

  it('accepts well-formed label values', () => {
    const result = affinityGroupSchema.safeParse({
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Required,
      conditions: [
        {
          key: 'zone',
          operator: AffinityOperator.In,
          values: ['us-east-1a', 'ssd_1', 'v1.2'],
        },
      ],
    });
    expect(result.success).toBe(true);
  });

  it('does not require values for Exists/DoesNotExist', () => {
    const result = affinityGroupSchema.safeParse({
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Required,
      conditions: [{ key: 'gpu', operator: AffinityOperator.Exists }],
    });
    expect(result.success).toBe(true);
  });

  it('requires a topologyKey for pod (anti)affinity', () => {
    const paths = issuePaths({
      type: AffinityType.PodAntiAffinity,
      priority: AffinityPriority.Required,
      conditions: [
        { key: 'app', operator: AffinityOperator.In, values: ['db'] },
      ],
    });
    expect(paths).toContain('topologyKey');
  });

  it('allows a pod condition without a key (key optional for pods)', () => {
    const result = affinityGroupSchema.safeParse({
      type: AffinityType.PodAffinity,
      priority: AffinityPriority.Required,
      topologyKey: 'kubernetes.io/hostname',
      conditions: [{}],
    });
    expect(result.success).toBe(true);
  });

  it('requires a weight between 1 and 100 for preferred groups', () => {
    expect(
      issuePaths({
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Preferred,
        conditions: [
          { key: 'disktype', operator: AffinityOperator.In, values: ['ssd'] },
        ],
      })
    ).toContain('weight');

    expect(
      issuePaths({
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Preferred,
        weight: 150,
        conditions: [
          { key: 'disktype', operator: AffinityOperator.In, values: ['ssd'] },
        ],
      })
    ).toContain('weight');
  });

  it('flags an empty conditions list', () => {
    expect(
      issuePaths({
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Required,
        conditions: [],
      })
    ).toContain('conditions');
  });
});
