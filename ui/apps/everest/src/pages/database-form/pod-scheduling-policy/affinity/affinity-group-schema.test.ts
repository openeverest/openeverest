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

  it('requires a key on pod conditions too (no selector matches no pods)', () => {
    const paths = issuePaths({
      type: AffinityType.PodAffinity,
      priority: AffinityPriority.Required,
      topologyKey: 'kubernetes.io/hostname',
      conditions: [{}],
    });
    expect(paths).toContain('conditions.0.key');
    expect(paths).toContain('conditions.0.operator');
  });

  it('accepts a group without conditions when its term keeps fields set outside the UI', () => {
    const source = {
      matchFields: [
        { key: 'metadata.name', operator: 'In', values: ['node-1'] },
      ],
    };
    const result = affinityGroupSchema.safeParse({
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Required,
      conditions: [],
      source,
    });
    expect(result.success).toBe(true);
    // The dialog submits the parsed value, so the term must come back intact.
    expect(result.success && result.data.source).toStrictEqual(source);
  });

  it('accepts a pod group without conditions whose empty selector matches all pods', () => {
    const result = affinityGroupSchema.safeParse({
      type: AffinityType.PodAntiAffinity,
      priority: AffinityPriority.Required,
      topologyKey: 'kubernetes.io/hostname',
      conditions: [],
      source: { topologyKey: 'kubernetes.io/hostname', labelSelector: {} },
    });
    expect(result.success).toBe(true);
  });

  it('requires conditions when the term selected by them alone', () => {
    const paths = issuePaths({
      type: AffinityType.PodAntiAffinity,
      priority: AffinityPriority.Required,
      topologyKey: 'kubernetes.io/hostname',
      conditions: [],
      source: {
        topologyKey: 'kubernetes.io/hostname',
        labelSelector: {
          matchExpressions: [{ key: 'app', operator: AffinityOperator.Exists }],
        },
      },
    });
    expect(paths).toContain('conditions');
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

  describe('Gt / Lt', () => {
    const nodeGroup = (values: string[]) => ({
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Required,
      conditions: [{ key: 'cpu-count', operator: AffinityOperator.Gt, values }],
    });

    it('accepts a single whole number, including a negative one', () => {
      expect(affinityGroupSchema.safeParse(nodeGroup(['4'])).success).toBe(
        true
      );
      expect(affinityGroupSchema.safeParse(nodeGroup(['-3'])).success).toBe(
        true
      );
    });

    it('rejects several values or a non-integer value', () => {
      expect(issuePaths(nodeGroup(['4', '8']))).toContain(
        'conditions.0.values'
      );
      expect(issuePaths(nodeGroup(['4.5']))).toContain('conditions.0.values');
      expect(issuePaths(nodeGroup(['ssd']))).toContain('conditions.0.values');
    });

    it('is allowed only for node affinity', () => {
      expect(
        issuePaths({
          ...nodeGroup(['4']),
          type: AffinityType.PodAffinity,
          topologyKey: 'kubernetes.io/hostname',
        })
      ).toContain('conditions.0.operator');
    });
  });
});
