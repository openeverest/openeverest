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
  changeDbClusterResources,
} from './db';
import { DbEngineType } from '@percona/types';
import { DbCluster, Proxy } from 'shared-types/dbCluster.types';

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
                ],
              },
              {
                matchExpressions: [
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

describe('changeDbClusterResources', () => {
  const makeCluster = (
    engineType: DbEngineType,
    engineResources: Record<string, unknown>
  ): DbCluster =>
    ({
      apiVersion: 'everest.percona.com/v1alpha1',
      kind: 'DatabaseCluster',
      metadata: { name: 'cluster', namespace: 'default' },
      spec: {
        engine: {
          type: engineType,
          version: '1.0.0',
          replicas: 1,
          resources: engineResources,
          storage: { size: '25Gi', class: 'standard' },
        },
        proxy: {
          type: 'haproxy',
          replicas: 1,
          resources: { cpu: '1', memory: '2G' },
          expose: { type: 'internal' },
        },
      },
    }) as unknown as DbCluster;

  const newResources = {
    cpu: 2,
    memory: 4,
    disk: 25,
    diskUnit: 'Gi',
    numberOfNodes: 1,
    proxyCpu: 1,
    proxyMemory: 2,
    numberOfProxies: 1,
  };

  it('omits engine requests for a legacy cluster when requests stay synced', () => {
    const cluster = makeCluster(DbEngineType.PSMDB, {
      cpu: '2',
      memory: '4G',
    });

    const result = changeDbClusterResources(cluster, {
      ...newResources,
      nodeRequestsSynced: true,
      proxyRequestsSynced: true,
    });

    expect(result.spec.engine.resources?.limits).toEqual({
      cpu: '2',
      memory: '4G',
    });
    expect(result.spec.engine.resources?.requests).toBeUndefined();
  });

  it('keeps limits only for a legacy cluster when the user toggles requests back to synced after prefilling them', () => {
    const cluster = makeCluster(DbEngineType.PSMDB, {
      cpu: '2',
      memory: '4G',
    });

    // Simulates the user turning the sync toggle off (fields get prefilled with
    // the limit values) and then turning it back on before saving. The
    // prefilled request values are still present in the form payload, but
    // because the cluster was legacy and stays synced we must ignore them and
    // persist limits only, so PSMDB/PostgreSQL do not restart.
    const result = changeDbClusterResources(cluster, {
      ...newResources,
      cpuRequests: 2,
      memoryRequests: 4,
      nodeRequestsSynced: true,
      proxyRequestsSynced: true,
    });

    expect(result.spec.engine.resources?.limits).toEqual({
      cpu: '2',
      memory: '4G',
    });
    expect(result.spec.engine.resources?.requests).toBeUndefined();
  });

  it('writes engine requests for a legacy cluster when requests are desynced', () => {
    const cluster = makeCluster(DbEngineType.PSMDB, {
      cpu: '2',
      memory: '4G',
    });

    const result = changeDbClusterResources(cluster, {
      ...newResources,
      cpuRequests: 1,
      memoryRequests: 2,
      nodeRequestsSynced: false,
      proxyRequestsSynced: true,
    });

    expect(result.spec.engine.resources?.requests).toEqual({
      cpu: '1',
      memory: '2G',
    });
  });

  it('writes engine requests for a non-legacy cluster even when synced', () => {
    const cluster = makeCluster(DbEngineType.PSMDB, {
      limits: { cpu: '2', memory: '4G' },
      requests: { cpu: '2', memory: '4G' },
    });

    const result = changeDbClusterResources(cluster, {
      ...newResources,
      nodeRequestsSynced: true,
      proxyRequestsSynced: true,
    });

    expect(result.spec.engine.resources?.requests).toEqual({
      cpu: '2',
      memory: '4G',
    });
  });

  it('keeps limits only when re-saving a limits-only engine while synced', () => {
    // After a first synced save, a legacy cluster becomes "limits only" (no
    // explicit requests). Re-opening and saving it while still synced must not
    // add requests back, otherwise PSMDB/PostgreSQL would restart needlessly.
    const cluster = makeCluster(DbEngineType.PSMDB, {
      cpu: '0',
      memory: '0',
      limits: { cpu: '2', memory: '4G' },
    });

    const result = changeDbClusterResources(cluster, {
      ...newResources,
      nodeRequestsSynced: true,
      proxyRequestsSynced: true,
    });

    expect(result.spec.engine.resources?.limits).toEqual({
      cpu: '2',
      memory: '4G',
    });
    expect(result.spec.engine.resources?.requests).toBeUndefined();
  });

  it('omits proxy requests for a legacy proxy when requests stay synced', () => {
    const cluster = makeCluster(DbEngineType.POSTGRESQL, {
      cpu: '2',
      memory: '4G',
    });

    const result = changeDbClusterResources(cluster, {
      ...newResources,
      // Even with prefilled proxy requests, a legacy synced proxy keeps limits
      // only so the workload does not restart.
      proxyCpuRequests: 1,
      proxyMemoryRequests: 2,
      nodeRequestsSynced: true,
      proxyRequestsSynced: true,
    });

    const proxy = result.spec.proxy as Proxy;
    expect(proxy.resources?.limits).toEqual({ cpu: '1', memory: '2G' });
    expect(proxy.resources?.requests).toBeUndefined();
  });

  it('keeps limits only when re-saving a limits-only proxy while synced', () => {
    const cluster = makeCluster(DbEngineType.POSTGRESQL, {
      cpu: '2',
      memory: '4G',
    });
    // The proxy was already saved with limits only (no explicit requests).
    (cluster.spec.proxy as Proxy).resources = {
      cpu: '0',
      memory: '0',
      limits: { cpu: '1', memory: '2G' },
    } as Proxy['resources'];

    const result = changeDbClusterResources(cluster, {
      ...newResources,
      proxyCpuRequests: 1,
      proxyMemoryRequests: 2,
      nodeRequestsSynced: true,
      proxyRequestsSynced: true,
    });

    const proxy = result.spec.proxy as Proxy;
    expect(proxy.resources?.limits).toEqual({ cpu: '1', memory: '2G' });
    expect(proxy.resources?.requests).toBeUndefined();
  });

  it('writes proxy requests for a legacy proxy when requests are desynced', () => {
    const cluster = makeCluster(DbEngineType.PXC, {
      cpu: '2',
      memory: '4G',
    });

    const result = changeDbClusterResources(cluster, {
      ...newResources,
      proxyCpuRequests: 1,
      proxyMemoryRequests: 2,
      nodeRequestsSynced: true,
      proxyRequestsSynced: false,
    });

    const proxy = result.spec.proxy as Proxy;
    expect(proxy.resources?.requests).toEqual({ cpu: '1', memory: '2G' });
  });

  it('writes proxy requests for a non-legacy proxy even when synced', () => {
    const cluster = makeCluster(DbEngineType.PXC, {
      cpu: '2',
      memory: '4G',
    });
    // Promote the proxy to the new format so it is no longer legacy.
    (cluster.spec.proxy as Proxy).resources = {
      limits: { cpu: '1', memory: '2G' },
      requests: { cpu: '1', memory: '2G' },
    };

    const result = changeDbClusterResources(cluster, {
      ...newResources,
      nodeRequestsSynced: true,
      proxyRequestsSynced: true,
    });

    const proxy = result.spec.proxy as Proxy;
    expect(proxy.resources?.requests).toEqual({ cpu: '1', memory: '2G' });
  });

  it('always writes engine requests for a legacy PXC cluster even when synced', () => {
    // PXC does not default absent requests to the limits, so we must always
    // persist explicit requests (equal to the limits when synced) to keep the
    // effective resource configuration intact.
    const cluster = makeCluster(DbEngineType.PXC, {
      cpu: '2',
      memory: '4G',
    });

    const result = changeDbClusterResources(cluster, {
      ...newResources,
      nodeRequestsSynced: true,
      proxyRequestsSynced: true,
    });

    expect(result.spec.engine.resources?.limits).toEqual({
      cpu: '2',
      memory: '4G',
    });
    expect(result.spec.engine.resources?.requests).toEqual({
      cpu: '2',
      memory: '4G',
    });
  });

  it('always writes proxy requests for a legacy PXC proxy even when synced', () => {
    const cluster = makeCluster(DbEngineType.PXC, {
      cpu: '2',
      memory: '4G',
    });

    const result = changeDbClusterResources(cluster, {
      ...newResources,
      nodeRequestsSynced: true,
      proxyRequestsSynced: true,
    });

    const proxy = result.spec.proxy as Proxy;
    expect(proxy.resources?.limits).toEqual({ cpu: '1', memory: '2G' });
    expect(proxy.resources?.requests).toEqual({ cpu: '1', memory: '2G' });
  });
});
