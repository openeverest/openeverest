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
  canSaveWithoutConditions,
  groupsToAffinity,
  matchesAllPods,
  matchesNoPods,
} from './affinity-group-converter';
import { AffinityGroup } from './affinity-group.types';
import { toAffinityGroups } from '../pod-scheduling-policy.utils';

const HOSTNAME = 'kubernetes.io/hostname';
const ZONE = 'topology.kubernetes.io/zone';
const EXISTS_APP = { key: 'app', operator: AffinityOperator.Exists };
const NODE_1 = { key: 'metadata.name', operator: 'In', values: ['node-1'] };

const EMPTY_SELECTORS: [string, Record<string, unknown>][] = [
  ['{}', {}],
  ['{ matchExpressions: [] }', { matchExpressions: [] }],
];

const requiredAnti = (term: Record<string, unknown>) => ({
  podAntiAffinity: { requiredDuringSchedulingIgnoredDuringExecution: [term] },
});

const nodeRequired = (term: Record<string, unknown>) => ({
  nodeAffinity: {
    requiredDuringSchedulingIgnoredDuringExecution: {
      nodeSelectorTerms: [term],
    },
  },
});

// Mirrors the dialog: the first group comes back with the edited fields.
const editFirst = (payload: unknown, patch: Partial<AffinityGroup>) => {
  const [first, ...rest] = toAffinityGroups(payload);
  return groupsToAffinity([{ ...first, ...patch }, ...rest]);
};

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
        ?.nodeSelectorTerms[0].matchExpressions?.[0];
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
    expect(restored).toEqual(
      expect.arrayContaining(
        groups.map((group) => expect.objectContaining(group))
      )
    );
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
        ? {
            ...group,
            conditions: [
              {
                key: 'disktype',
                operator: AffinityOperator.In,
                values: ['nvme'],
              },
            ],
          }
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

    const switched: AffinityGroup = {
      ...podGroup,
      type: AffinityType.NodeAffinity,
      topologyKey: undefined,
    };

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

  it.each(EMPTY_SELECTORS)(
    'does not flag labelSelector %s, which matches every pod',
    (_, labelSelector) => {
      const [group] = toAffinityGroups(
        requiredAnti({ topologyKey: HOSTNAME, labelSelector })
      );
      expect(matchesNoPods(group)).toBe(false);
    }
  );

  it('never flags node groups', () => {
    const [group] = toAffinityGroups(nodeRequired({ matchFields: [NODE_1] }));
    expect(matchesNoPods(group)).toBe(false);
  });
});

// k8s: an empty label selector matches every pod, an absent one matches none,
// so a save must never turn one into the other.
describe('matchesAllPods', () => {
  it.each([...EMPTY_SELECTORS, ['{ matchLabels: {} }', { matchLabels: {} }]])(
    'flags labelSelector %s',
    (_, labelSelector) => {
      const [group] = toAffinityGroups(
        requiredAnti({ topologyKey: HOSTNAME, labelSelector })
      );
      expect(matchesAllPods(group)).toBe(true);
    }
  );

  it.each([
    ['a selector with matchLabels', { matchLabels: { app: 'mysql' } }],
    ['a selector with conditions', { matchExpressions: [EXISTS_APP] }],
    ['no selector', undefined],
  ])('does not flag %s', (_, labelSelector) => {
    const [group] = toAffinityGroups(
      requiredAnti({ topologyKey: HOSTNAME, labelSelector })
    );
    expect(matchesAllPods(group)).toBe(false);
  });

  it('stops flagging an empty selector once a condition is added', () => {
    const [group] = toAffinityGroups(
      requiredAnti({ topologyKey: HOSTNAME, labelSelector: {} })
    );
    expect(
      matchesAllPods({
        ...group,
        conditions: [{ key: 'app', operator: AffinityOperator.Exists }],
      })
    ).toBe(false);
  });

  it('never flags node groups', () => {
    const [group] = toAffinityGroups(nodeRequired({ matchFields: [NODE_1] }));
    expect(matchesAllPods(group)).toBe(false);
  });
});

describe('empty vs absent label selector', () => {
  it.each(EMPTY_SELECTORS)(
    'keeps labelSelector %s verbatim on a round-trip',
    (_, labelSelector) => {
      const payload = requiredAnti({ topologyKey: HOSTNAME, labelSelector });
      expect(groupsToAffinity(toAffinityGroups(payload))).toStrictEqual(
        payload
      );
    }
  );

  it.each(EMPTY_SELECTORS)(
    'keeps labelSelector %s when another group of the component is saved',
    (_, labelSelector) => {
      const payload = {
        ...nodeRequired({ matchExpressions: [EXISTS_APP] }),
        ...requiredAnti({ topologyKey: HOSTNAME, labelSelector }),
      };

      const result = editFirst(payload, {
        conditions: [{ key: 'gpu', operator: AffinityOperator.Exists }],
      });

      expect(result.podAntiAffinity).toStrictEqual(payload.podAntiAffinity);
    }
  );

  it.each(EMPTY_SELECTORS)(
    'keeps labelSelector %s when its own topologyKey is edited',
    (_, labelSelector) => {
      const result = editFirst(
        requiredAnti({ topologyKey: HOSTNAME, labelSelector }),
        { topologyKey: ZONE }
      );
      expect(result).toStrictEqual(
        requiredAnti({ topologyKey: ZONE, labelSelector })
      );
    }
  );

  it('writes added conditions into an empty selector', () => {
    const result = editFirst(
      requiredAnti({ topologyKey: HOSTNAME, labelSelector: {} }),
      { conditions: [{ key: 'app', operator: AffinityOperator.Exists }] }
    );
    expect(result).toStrictEqual(
      requiredAnti({
        topologyKey: HOSTNAME,
        labelSelector: { matchExpressions: [EXISTS_APP] },
      })
    );
  });

  it('drops the selector rather than widening it to all pods when every condition is removed', () => {
    const payload = requiredAnti({
      topologyKey: HOSTNAME,
      labelSelector: { matchExpressions: [EXISTS_APP] },
    });
    const [group] = toAffinityGroups(payload);
    const emptied = { ...group, conditions: [] };

    expect(groupsToAffinity([emptied])).toStrictEqual(
      requiredAnti({ topologyKey: HOSTNAME })
    );
    expect(matchesNoPods(emptied)).toBe(true);
  });

  it('keeps the other selector fields when every condition is removed', () => {
    const result = editFirst(
      requiredAnti({
        topologyKey: HOSTNAME,
        labelSelector: {
          matchLabels: { app: 'mysql' },
          matchExpressions: [EXISTS_APP],
        },
      }),
      { conditions: [] }
    );
    expect(result).toStrictEqual(
      requiredAnti({
        topologyKey: HOSTNAME,
        labelSelector: { matchLabels: { app: 'mysql' } },
      })
    );
  });
});

describe('editing a group keeps what the editor does not model', () => {
  const podTerm = {
    topologyKey: HOSTNAME,
    namespaces: ['db'],
    namespaceSelector: {},
    matchLabelKeys: ['pod-template-hash'],
    labelSelector: {
      matchLabels: { app: 'mysql' },
      matchExpressions: [EXISTS_APP],
    },
  };

  it('keeps them when the group’s own conditions and topologyKey change', () => {
    const result = editFirst(requiredAnti(podTerm), {
      topologyKey: ZONE,
      conditions: [
        { key: 'tier', operator: AffinityOperator.In, values: ['db'] },
      ],
    });

    expect(result).toStrictEqual(
      requiredAnti({
        ...podTerm,
        topologyKey: ZONE,
        labelSelector: {
          matchLabels: { app: 'mysql' },
          matchExpressions: [
            { key: 'tier', operator: AffinityOperator.In, values: ['db'] },
          ],
        },
      })
    );
  });

  it('keeps them on a preferred term', () => {
    const payload = {
      podAntiAffinity: {
        preferredDuringSchedulingIgnoredDuringExecution: [
          { weight: 10, podAffinityTerm: podTerm },
        ],
      },
    };
    expect(groupsToAffinity(toAffinityGroups(payload))).toStrictEqual(payload);
  });

  it('keeps them when switching between pod affinity and anti-affinity', () => {
    const result = editFirst(requiredAnti(podTerm), {
      type: AffinityType.PodAffinity,
    });
    expect(result).toStrictEqual({
      podAffinity: {
        requiredDuringSchedulingIgnoredDuringExecution: [podTerm],
      },
    });
  });

  it('drops node-only fields when a node group is switched to pod affinity', () => {
    const result = editFirst(
      nodeRequired({ matchFields: [NODE_1], matchExpressions: [EXISTS_APP] }),
      { type: AffinityType.PodAntiAffinity, topologyKey: HOSTNAME }
    );
    expect(result).toStrictEqual(
      requiredAnti({
        topologyKey: HOSTNAME,
        labelSelector: { matchExpressions: [EXISTS_APP] },
      })
    );
  });

  it.each([
    ['required', nodeRequired({ matchFields: [NODE_1] })],
    [
      'preferred',
      {
        nodeAffinity: {
          preferredDuringSchedulingIgnoredDuringExecution: [
            { weight: 5, preference: { matchFields: [NODE_1] } },
          ],
        },
      },
    ],
  ])(
    'does not add matchExpressions to a %s node term that uses matchFields only',
    (_, payload) => {
      expect(groupsToAffinity(toAffinityGroups(payload))).toStrictEqual(
        payload
      );
    }
  );
});

describe('canSaveWithoutConditions', () => {
  const withoutConditions = (
    payload: unknown,
    patch: Partial<AffinityGroup> = {}
  ): AffinityGroup => {
    const [group] = toAffinityGroups(payload);
    return { ...group, ...patch, conditions: [] };
  };

  it.each([
    [
      'a node term with matchFields',
      nodeRequired({ matchFields: [NODE_1], matchExpressions: [EXISTS_APP] }),
      true,
    ],
    [
      'a node term with matchExpressions only',
      nodeRequired({ matchExpressions: [EXISTS_APP] }),
      false,
    ],
    [
      'a pod term with an empty selector',
      requiredAnti({ topologyKey: HOSTNAME, labelSelector: {} }),
      true,
    ],
    [
      'a pod term with matchLabels',
      requiredAnti({
        topologyKey: HOSTNAME,
        labelSelector: {
          matchLabels: { app: 'mysql' },
          matchExpressions: [EXISTS_APP],
        },
      }),
      true,
    ],
    [
      'a pod term with namespaces',
      requiredAnti({
        topologyKey: HOSTNAME,
        namespaces: ['db'],
        labelSelector: { matchExpressions: [EXISTS_APP] },
      }),
      true,
    ],
    [
      'a pod term with matchExpressions only',
      requiredAnti({
        topologyKey: HOSTNAME,
        labelSelector: { matchExpressions: [EXISTS_APP] },
      }),
      false,
    ],
    [
      'a pod term without a selector (the v1 default rule)',
      requiredAnti({ topologyKey: HOSTNAME }),
      false,
    ],
  ])('%s: %s', (_, payload, expected) => {
    expect(canSaveWithoutConditions(withoutConditions(payload))).toBe(expected);
  });

  it('is false for a group created in the editor', () => {
    expect(
      canSaveWithoutConditions({
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Required,
        conditions: [],
      })
    ).toBe(false);
  });

  it('ignores a source of the other kind after a node/pod switch', () => {
    const switched = withoutConditions(
      nodeRequired({ matchFields: [NODE_1] }),
      { type: AffinityType.PodAntiAffinity, topologyKey: HOSTNAME }
    );
    expect(canSaveWithoutConditions(switched)).toBe(false);
  });
});
