import {
  AffinityRule,
  Affinity,
  AffinityComponent,
  AffinityType,
  AffinityPriority,
  AffinityOperator,
  PodSchedulingPolicy,
} from 'shared-types/affinity.types';
import {
  affinityRulesToDbPayload,
  insertAffinityRuleToExistingPolicy,
  removeRuleInExistingPolicy,
} from './db';
import { DbEngineType } from '@percona/types';

describe('affinityRulesToDbPayload', () => {
  const tests: [string, AffinityRule[], Affinity][] = [
    ['empty', [], {}],
    [
      'single required node affinity',
      [
        {
          component: AffinityComponent.DbNode,
          type: AffinityType.NodeAffinity,
          priority: AffinityPriority.Required,
          uid: '',
          key: 'my-key',
          operator: AffinityOperator.Exists,
        },
      ],
      {
        nodeAffinity: {
          requiredDuringSchedulingIgnoredDuringExecution: {
            nodeSelectorTerms: [
              {
                matchExpressions: [
                  {
                    key: 'my-key',
                    operator: AffinityOperator.Exists,
                  },
                ],
              },
            ],
          },
        },
      },
    ],
    [
      'multiple required node affinity',
      [
        {
          component: AffinityComponent.DbNode,
          type: AffinityType.NodeAffinity,
          priority: AffinityPriority.Required,
          uid: '',
          key: 'my-key',
          operator: AffinityOperator.Exists,
        },
        {
          component: AffinityComponent.DbNode,
          type: AffinityType.NodeAffinity,
          priority: AffinityPriority.Required,
          uid: '',
          key: 'my-other-key',
          operator: AffinityOperator.In,
          values: 'value1,value2',
        },
      ],
      {
        nodeAffinity: {
          requiredDuringSchedulingIgnoredDuringExecution: {
            nodeSelectorTerms: [
              {
                matchExpressions: [
                  {
                    key: 'my-key',
                    operator: AffinityOperator.Exists,
                  },
                  {
                    key: 'my-other-key',
                    operator: AffinityOperator.In,
                    values: ['value1', 'value2'],
                  },
                ],
              },
            ],
          },
        },
      },
    ],
    [
      'mixed affinities',
      [
        {
          component: AffinityComponent.DbNode,
          type: AffinityType.NodeAffinity,
          priority: AffinityPriority.Preferred,
          weight: 10,
          uid: '',
          key: 'my-key',
          operator: AffinityOperator.Exists,
        },
        {
          component: AffinityComponent.DbNode,
          type: AffinityType.PodAffinity,
          priority: AffinityPriority.Required,
          topologyKey: 'my-topology-key',
          uid: '',
        },
        {
          component: AffinityComponent.DbNode,
          type: AffinityType.PodAntiAffinity,
          priority: AffinityPriority.Preferred,
          weight: 20,
          topologyKey: 'my-topology-key',
          operator: AffinityOperator.NotIn,
          key: 'my-key',
          values: 'value1',
          uid: '',
        },
        {
          component: AffinityComponent.DbNode,
          type: AffinityType.PodAntiAffinity,
          priority: AffinityPriority.Preferred,
          weight: 15,
          topologyKey: 'my-topology-key',
          operator: AffinityOperator.Exists,
          key: 'my-key',
          // This rule is not using values, but we test if it does not end up in the payload
          values: 'value1',
          uid: '',
        },
      ],
      {
        nodeAffinity: {
          preferredDuringSchedulingIgnoredDuringExecution: [
            {
              weight: 10,
              preference: {
                matchExpressions: [
                  {
                    key: 'my-key',
                    operator: AffinityOperator.Exists,
                  },
                ],
              },
            },
          ],
        },
        podAffinity: {
          requiredDuringSchedulingIgnoredDuringExecution: [
            {
              topologyKey: 'my-topology-key',
            },
          ],
        },
        podAntiAffinity: {
          preferredDuringSchedulingIgnoredDuringExecution: [
            {
              weight: 20,
              podAffinityTerm: {
                topologyKey: 'my-topology-key',
                labelSelector: {
                  matchExpressions: [
                    {
                      key: 'my-key',
                      operator: AffinityOperator.NotIn,
                      values: ['value1'],
                    },
                  ],
                },
              },
            },
            {
              weight: 15,
              podAffinityTerm: {
                topologyKey: 'my-topology-key',
                labelSelector: {
                  matchExpressions: [
                    {
                      key: 'my-key',
                      operator: AffinityOperator.Exists,
                    },
                  ],
                },
              },
            },
          ],
        },
      },
    ],
  ];

  tests.map(([name, input, expected]) => {
    it(name, () => {
      expect(affinityRulesToDbPayload(input)).toEqual(expected);
    });
  });
});

describe('insertAffinityRuleToExistingPolicy', () => {
  test('Add to empty policy', () => {
    const policy: PodSchedulingPolicy = {
      metadata: {
        generation: 1,
        resourceVersion: '1',
        name: 'test-policy',
        finalizers: [],
      },
      spec: {
        engineType: DbEngineType.PSMDB,
        affinityConfig: {},
      },
    };
    const input: AffinityRule = {
      component: AffinityComponent.DbNode,
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Preferred,
      weight: 10,
      uid: '',
      key: 'my-key',
      operator: AffinityOperator.In,
      values: 'value1,value2',
    };
    insertAffinityRuleToExistingPolicy(policy, input);
    expect(policy.spec.affinityConfig.psmdb).not.toBeUndefined();
    expect(policy.spec.affinityConfig.psmdb?.engine).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
    ).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution
    ).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution
    ).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution
    ).toHaveLength(1);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0].weight
    ).toEqual(10);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions
    ).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions
    ).toHaveLength(1);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions[0]?.key
    ).toEqual('my-key');
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions[0]?.operator
    ).toEqual(AffinityOperator.In);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions[0]?.values
    ).toEqual(['value1', 'value2']);
  });

  test('Add to existing policy with different priority', () => {
    const policy: PodSchedulingPolicy = {
      metadata: {
        generation: 1,
        resourceVersion: '1',
        name: 'test-policy',
        finalizers: [],
      },
      spec: {
        engineType: DbEngineType.PSMDB,
        affinityConfig: {
          psmdb: {
            engine: {
              nodeAffinity: {
                requiredDuringSchedulingIgnoredDuringExecution: {
                  nodeSelectorTerms: [
                    {
                      matchExpressions: [
                        {
                          key: 'my-key',
                          operator: AffinityOperator.Exists,
                        },
                      ],
                    },
                  ],
                },
              },
            },
          },
        },
      },
    };
    const input: AffinityRule = {
      component: AffinityComponent.DbNode,
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Preferred,
      weight: 10,
      uid: '',
      key: 'my-key',
      operator: AffinityOperator.NotIn,
      values: 'value1,value2',
    };
    insertAffinityRuleToExistingPolicy(policy, input);
    expect(policy.spec.affinityConfig.psmdb).not.toBeUndefined();
    expect(policy.spec.affinityConfig.psmdb?.engine).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution
    ).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution
    ).toHaveLength(1);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0].weight
    ).toEqual(10);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions
    ).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions
    ).toHaveLength(1);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions[0]?.key
    ).toEqual('my-key');
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions[0]?.operator
    ).toEqual(AffinityOperator.NotIn);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions[0]?.values
    ).toEqual(['value1', 'value2']);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.requiredDuringSchedulingIgnoredDuringExecution?.nodeSelectorTerms[0]
        .matchExpressions[0].key
    ).toEqual('my-key');
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.requiredDuringSchedulingIgnoredDuringExecution?.nodeSelectorTerms[0]
        .matchExpressions[0].operator
    ).toEqual(AffinityOperator.Exists);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.requiredDuringSchedulingIgnoredDuringExecution?.nodeSelectorTerms[0]
        .matchExpressions[0].values
    ).toEqual(undefined);
  });

  test('Add to existing policy with same priority', () => {
    const policy: PodSchedulingPolicy = {
      metadata: {
        generation: 1,
        resourceVersion: '1',
        name: 'test-policy',
        finalizers: [],
      },
      spec: {
        engineType: DbEngineType.PSMDB,
        affinityConfig: {
          psmdb: {
            engine: {
              nodeAffinity: {
                preferredDuringSchedulingIgnoredDuringExecution: [
                  {
                    weight: 10,
                    preference: {
                      matchExpressions: [
                        {
                          key: 'my-key',
                          operator: AffinityOperator.Exists,
                        },
                      ],
                    },
                  },
                ],
              },
            },
          },
        },
      },
    };
    const input: AffinityRule = {
      component: AffinityComponent.DbNode,
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Preferred,
      weight: 20,
      uid: '',
      key: 'my-other-key',
      operator: AffinityOperator.In,
      values: 'value1,value2',
    };
    insertAffinityRuleToExistingPolicy(policy, input);
    expect(policy.spec.affinityConfig.psmdb).not.toBeUndefined();
    expect(policy.spec.affinityConfig.psmdb?.engine).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution
    ).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution
    ).toHaveLength(2);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0].weight
    ).toEqual(10);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![1].weight
    ).toEqual(20);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![1]?.preference
        ?.matchExpressions
    ).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions[0]?.key
    ).toEqual('my-key');
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![1]?.preference
        ?.matchExpressions[0]?.key
    ).toEqual('my-other-key');
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0]?.preference
        ?.matchExpressions[0]?.operator
    ).toEqual(AffinityOperator.Exists);
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![1]?.preference
        ?.matchExpressions[0]?.operator
    ).toEqual(AffinityOperator.In);
  });
});

describe('removeRuleInExistingPolicy', () => {
  test('Remove from empty policy', () => {
    const policy: PodSchedulingPolicy = {
      metadata: {
        generation: 1,
        resourceVersion: '1',
        name: 'test-policy',
        finalizers: [],
      },
      spec: {
        engineType: DbEngineType.PSMDB,
        affinityConfig: {},
      },
    };
    const input: AffinityRule = {
      component: AffinityComponent.DbNode,
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Preferred,
      weight: 10,
      uid: '',
      key: 'my-key',
      operator: AffinityOperator.In,
      values: 'value1,value2',
    };
    removeRuleInExistingPolicy(policy, input);
    expect(policy.spec.affinityConfig).toBeUndefined();
  });

  test('Do not remove unexisting rule', () => {
    const policy: PodSchedulingPolicy = {
      metadata: {
        generation: 1,
        resourceVersion: '1',
        name: 'test-policy',
        finalizers: [],
      },
      spec: {
        engineType: DbEngineType.PSMDB,
        affinityConfig: {
          psmdb: {
            engine: {
              nodeAffinity: {
                preferredDuringSchedulingIgnoredDuringExecution: [
                  {
                    weight: 10,
                    preference: {
                      matchExpressions: [
                        {
                          key: 'my-key',
                          operator: AffinityOperator.Exists,
                        },
                      ],
                    },
                  },
                ],
              },
            },
          },
        },
      },
    };
    const input: AffinityRule = {
      component: AffinityComponent.DbNode,
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Required,
      weight: 10,
      uid: '',
      key: 'my-key',
      operator: AffinityOperator.NotIn,
      values: 'value1,value2',
    };
    removeRuleInExistingPolicy(policy, input);
    expect(policy.spec.affinityConfig.psmdb).not.toBeUndefined();
    expect(policy.spec.affinityConfig.psmdb?.engine).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.requiredDuringSchedulingIgnoredDuringExecution
    ).toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution![0].preference
        .matchExpressions[0].key
    ).toEqual('my-key');
  });

  test('Remove existing rule', () => {
    const policy: PodSchedulingPolicy = {
      metadata: {
        generation: 1,
        resourceVersion: '1',
        name: 'test-policy',
        finalizers: [],
      },
      spec: {
        engineType: DbEngineType.PSMDB,
        affinityConfig: {
          psmdb: {
            engine: {
              nodeAffinity: {
                preferredDuringSchedulingIgnoredDuringExecution: [
                  {
                    weight: 10,
                    preference: {
                      matchExpressions: [
                        {
                          key: 'my-key',
                          operator: AffinityOperator.Exists,
                        },
                      ],
                    },
                  },
                  {
                    weight: 20,
                    preference: {
                      matchExpressions: [
                        {
                          key: 'my-other-key',
                          operator: AffinityOperator.Exists,
                        },
                      ],
                    },
                  },
                ],
              },
            },
          },
        },
      },
    };
    const input: AffinityRule = {
      component: AffinityComponent.DbNode,
      type: AffinityType.NodeAffinity,
      priority: AffinityPriority.Preferred,
      weight: 10,
      uid: '',
      key: 'my-key',
      operator: AffinityOperator.Exists,
    };
    removeRuleInExistingPolicy(policy, input);
    expect(policy.spec.affinityConfig.psmdb).not.toBeUndefined();
    expect(policy.spec.affinityConfig.psmdb?.engine).not.toBeUndefined();
    expect(
      policy.spec.affinityConfig.psmdb?.engine?.nodeAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution
    ).toHaveLength(1);
  });
});
