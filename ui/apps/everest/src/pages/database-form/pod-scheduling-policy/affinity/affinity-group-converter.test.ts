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
import {
  affinityToGroups,
  applyGroupEdit,
  groupsToAffinity,
  matchesNoPods,
} from './affinity-group-converter';
import { AffinityGroup } from './affinity-group.types';
import { toAffinityGroups } from '../pod-scheduling-policy.utils';

describe('groupsToAffinity', () => {
  it('keeps every condition of a node group as AND-ed matchExpressions (#1985)', () => {
    const affinity = groupsToAffinity([
      {
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Required,
        conditions: [
          { key: 'disktype', operator: AffinityOperator.In, values: ['ssd'] },
          { key: 'region', operator: AffinityOperator.In, values: ['eu'] },
        ],
      },
    ]);

    const terms =
      affinity.nodeAffinity?.requiredDuringSchedulingIgnoredDuringExecution
        ?.nodeSelectorTerms;
    expect(terms).toHaveLength(1);
    expect(terms?.[0].matchExpressions).toHaveLength(2);
  });

  it('OR-joins node required groups as separate nodeSelectorTerms', () => {
    const affinity = groupsToAffinity([
      {
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Required,
        conditions: [
          { key: 'a', operator: AffinityOperator.In, values: ['1'] },
        ],
      },
      {
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Required,
        conditions: [
          { key: 'b', operator: AffinityOperator.In, values: ['2'] },
        ],
      },
    ]);

    expect(
      affinity.nodeAffinity?.requiredDuringSchedulingIgnoredDuringExecution
        ?.nodeSelectorTerms
    ).toHaveLength(2);
  });

  it('adds weight for preferred node groups', () => {
    const affinity = groupsToAffinity([
      {
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Preferred,
        weight: 50,
        conditions: [
          { key: 'tier', operator: AffinityOperator.In, values: ['fast'] },
        ],
      },
    ]);

    const preferred =
      affinity.nodeAffinity?.preferredDuringSchedulingIgnoredDuringExecution;
    expect(preferred).toHaveLength(1);
    expect(preferred?.[0].weight).toBe(50);
    expect(preferred?.[0].preference.matchExpressions).toHaveLength(1);
  });

  it('maps a pod anti-affinity group to a PodAffinityTerm with topologyKey', () => {
    const affinity = groupsToAffinity([
      {
        type: AffinityType.PodAntiAffinity,
        priority: AffinityPriority.Required,
        topologyKey: 'kubernetes.io/hostname',
        conditions: [
          { key: 'app', operator: AffinityOperator.In, values: ['db'] },
        ],
      },
    ]);

    const term =
      affinity.podAntiAffinity
        ?.requiredDuringSchedulingIgnoredDuringExecution?.[0];
    expect(term?.topologyKey).toBe('kubernetes.io/hostname');
    expect(term?.labelSelector?.matchExpressions).toHaveLength(1);
  });

  it('omits values for operators that do not consume them (Exists)', () => {
    const affinity = groupsToAffinity([
      {
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Required,
        conditions: [{ key: 'gpu', operator: AffinityOperator.Exists }],
      },
    ]);

    const expression =
      affinity.nodeAffinity?.requiredDuringSchedulingIgnoredDuringExecution
        ?.nodeSelectorTerms[0].matchExpressions[0];
    expect(expression?.operator).toBe(AffinityOperator.Exists);
    expect(expression?.values).toBeUndefined();
  });

  it('returns an empty object for no groups', () => {
    expect(groupsToAffinity([])).toEqual({});
  });
});

describe('affinityToGroups round-trip', () => {
  it('restores the groups (incl. multiple conditions) after a k8s round-trip', () => {
    const groups: AffinityGroup[] = [
      {
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Required,
        conditions: [
          { key: 'disktype', operator: AffinityOperator.In, values: ['ssd'] },
          { key: 'region', operator: AffinityOperator.In, values: ['eu'] },
        ],
      },
      {
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Preferred,
        weight: 50,
        conditions: [
          { key: 'tier', operator: AffinityOperator.In, values: ['fast'] },
        ],
      },
      {
        type: AffinityType.PodAntiAffinity,
        priority: AffinityPriority.Required,
        topologyKey: 'kubernetes.io/hostname',
        conditions: [
          { key: 'app', operator: AffinityOperator.In, values: ['db'] },
        ],
      },
    ];

    const restored = affinityToGroups(groupsToAffinity(groups));

    expect(restored).toHaveLength(3);
    expect(restored).toEqual(expect.arrayContaining(groups));
  });
});

describe('fields the editor does not model', () => {
  // Set via kubectl / GitOps: matchFields, namespaces and matchLabels have no
  // editor controls but must survive a save.
  const affinity = {
    nodeAffinity: {
      requiredDuringSchedulingIgnoredDuringExecution: {
        nodeSelectorTerms: [
          {
            matchExpressions: [
              {
                key: 'disktype',
                operator: AffinityOperator.In,
                values: ['ssd'],
              },
            ],
            matchFields: [
              { key: 'metadata.name', operator: 'In', values: ['node-1'] },
            ],
          },
        ],
      },
    },
    podAntiAffinity: {
      preferredDuringSchedulingIgnoredDuringExecution: [
        {
          weight: 10,
          podAffinityTerm: {
            topologyKey: 'kubernetes.io/hostname',
            namespaces: ['db'],
            labelSelector: {
              matchLabels: { app: 'mysql' },
              matchExpressions: [
                { key: 'tier', operator: AffinityOperator.Exists },
              ],
            },
          },
        },
      ],
    },
  };

  it('writes them back unchanged after a round-trip', () => {
    expect(groupsToAffinity(affinityToGroups(affinity))).toEqual(affinity);
  });

  it('keeps them on every group when another group of the component is edited', () => {
    const groups = affinityToGroups(affinity);
    const nodeIndex = groups.findIndex(
      ({ type }) => type === AffinityType.NodeAffinity
    );
    const edited = groups.map((group, index) =>
      index === nodeIndex
        ? applyGroupEdit(group, {
            ...group,
            conditions: [
              {
                key: 'disktype',
                operator: AffinityOperator.In,
                values: ['nvme'],
              },
            ],
          })
        : group
    );

    const result = groupsToAffinity(edited);

    expect(
      result.nodeAffinity?.requiredDuringSchedulingIgnoredDuringExecution
        ?.nodeSelectorTerms[0]
    ).toEqual({
      matchExpressions: [
        { key: 'disktype', operator: AffinityOperator.In, values: ['nvme'] },
      ],
      matchFields: [
        { key: 'metadata.name', operator: 'In', values: ['node-1'] },
      ],
    });
    expect(result.podAntiAffinity).toEqual(affinity.podAntiAffinity);
  });

  it('drops them when an edit switches a group between pod and node affinity', () => {
    const [podGroup] = affinityToGroups({
      podAntiAffinity: affinity.podAntiAffinity,
    });

    const switched = applyGroupEdit(podGroup, {
      ...podGroup,
      type: AffinityType.NodeAffinity,
      topologyKey: undefined,
    });

    expect(
      groupsToAffinity([switched]).nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution?.[0].preference
    ).toEqual({
      matchExpressions: [{ key: 'tier', operator: AffinityOperator.Exists }],
    });
  });

  it('keeps the value of node Gt / Lt expressions the editor cannot select', () => {
    const numeric = {
      nodeAffinity: {
        requiredDuringSchedulingIgnoredDuringExecution: {
          nodeSelectorTerms: [
            {
              matchExpressions: [
                { key: 'cpu-count', operator: 'Gt', values: ['4'] },
              ],
            },
          ],
        },
      },
    };

    expect(groupsToAffinity(toAffinityGroups(numeric))).toEqual(numeric);
  });
});

describe('matchesNoPods', () => {
  it('flags a pod term without any label selector (the v1 default rule)', () => {
    const [group] = affinityToGroups({
      podAntiAffinity: {
        preferredDuringSchedulingIgnoredDuringExecution: [
          {
            weight: 1,
            podAffinityTerm: { topologyKey: 'kubernetes.io/hostname' },
          },
        ],
      },
    });
    expect(matchesNoPods(group)).toBe(true);
  });

  it('does not flag a pod term that selects by matchLabels only', () => {
    const [group] = toAffinityGroups({
      podAntiAffinity: {
        requiredDuringSchedulingIgnoredDuringExecution: [
          {
            topologyKey: 'kubernetes.io/hostname',
            labelSelector: { matchLabels: { app: 'mysql' } },
          },
        ],
      },
    });
    expect(matchesNoPods(group)).toBe(false);
  });
});
