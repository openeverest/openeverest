import { MongoIcon, MySqlIcon, PostgreSqlIcon } from '@percona/ui-lib';
import { DbEngineType, DbType, ProxyType } from '@percona/types';
import {
  DbCluster,
  DbClusterStatus,
  ManageableSchedules,
  Proxy,
  ProxyExposeType,
  Schedule,
} from 'shared-types/dbCluster.types';
import { can } from './rbac';
import {
  Affinity,
  AffinityComponent,
  AffinityMatchExpression,
  AffinityOperator,
  AffinityPriority,
  AffinityRule,
  AffinityType,
  NodeAffinity,
  PodAffinity,
  PodAffinityTerm,
  PodAntiAffinity,
  PodSchedulingPolicy,
  PreferredNodeSchedulingTerm,
  PreferredPodSchedulingTerm,
  RequiredNodeSchedulingTerm,
  RequiredPodSchedulingTerm,
} from 'shared-types/affinity.types';
import { generateShortUID } from './generateShortUID';
import { capitalize } from '@mui/material';
import { getProxySpec } from 'hooks/api/db-cluster/utils';
import { dbEngineToDbType } from '@percona/utils';
import { MIN_NUMBER_OF_SHARDS } from 'components/cluster-form';
import { Path, UseFormGetFieldState } from 'react-hook-form';
import cronConverter from './cron-converter';
import { EMPTY_LOAD_BALANCER_CONFIGURATION } from 'consts';
import { mapDeprecatedExposeType } from 'components/cluster-form/advanced-configuration/advanced-configuration.utils';

export const dbTypeToIcon = (dbType: DbType) => {
  switch (dbType) {
    case DbType.Mongo:
      return MongoIcon;
    case DbType.Mysql:
      return MySqlIcon;
    default:
      return PostgreSqlIcon;
  }
};

export const shortenOperatorName = (name: string) => {
  if (name.includes('postgresql')) {
    return 'postgresql';
  }

  if (name.includes('xtradb')) {
    return 'pxc';
  }

  if (name.includes('mongodb')) {
    return 'psmdb';
  }

  return name;
};

export const dbTypeToProxyType = (dbType: DbType): ProxyType => {
  switch (dbType) {
    case DbType.Mongo:
      return 'mongos';
    case DbType.Mysql:
      return 'haproxy';
    default:
      return 'pgbouncer';
  }
};

export const getProxyUnitNamesFromDbType = (
  dbType: DbType
): { singular: string; plural: string } => {
  switch (dbType) {
    case DbType.Postresql:
      return { singular: 'PG Bouncer', plural: 'PG Bouncers' };
    case DbType.Mongo:
      return { singular: 'router', plural: 'routers' };
    case DbType.Mysql:
    default:
      return { singular: 'proxy', plural: 'proxies' };
  }
};

export const getPreviewResourcesText = (
  type: 'CPU' | 'Memory' | 'Disk',
  parsedResource: number,
  sharding: boolean,
  measurementUnit: string,
  parsedShardNr?: number
) => {
  return Number.isNaN(parsedResource)
    ? ''
    : `${type} - ${sharding && parsedShardNr ? (parsedShardNr * parsedResource).toFixed(2) : parsedResource.toFixed(2)} ${measurementUnit}`;
};

export const someErrorInStateFields = <T extends Record<string, unknown>>(
  fieldStateGetter: UseFormGetFieldState<T>,
  fields: Path<T>[]
) => {
  return fields.some((field) => fieldStateGetter(field)?.error);
};

export const isProxy = (proxy: Proxy): boolean => {
  return proxy && typeof proxy.expose === 'object';
};
export const transformSchedulesIntoManageableSchedules = async (
  schedules: Schedule[],
  namespace: string,
  canCreateBackups: boolean,
  canUpdateDb: boolean
) => {
  const transformedSchedules: ManageableSchedules[] = await Promise.all(
    schedules.map(async (schedule) => ({
      ...schedule,
      canBeManaged:
        (await can(
          'read',
          'backup-storages',
          `${namespace}/${schedule.backupStorageName}`
        )) &&
        canCreateBackups &&
        canUpdateDb,
    }))
  );

  return transformedSchedules;
};

export const dbPayloadToAffinityRules = (
  payload: PodSchedulingPolicy
): AffinityRule[] => {
  const rules: AffinityRule[] = [];
  const {
    spec: { engineType, affinityConfig = {} },
  } = payload;
  const engineAffinity = affinityConfig[engineType]?.engine || {};
  const proxyAffinity = affinityConfig[engineType]?.proxy || {};
  const configServerAffinity = affinityConfig[engineType]?.configServer || {};
  const allAffinities = [
    {
      affinityObject: engineAffinity,
      component: AffinityComponent.DbNode,
    },
    {
      affinityObject: proxyAffinity,
      component: AffinityComponent.Proxy,
    },
    {
      affinityObject: configServerAffinity,
      component: AffinityComponent.ConfigServer,
    },
  ];

  allAffinities.forEach(({ affinityObject, component }) => {
    if (affinityObject && Object.keys(affinityObject).length) {
      const { nodeAffinity, podAffinity, podAntiAffinity } = affinityObject;

      if (nodeAffinity) {
        const {
          preferredDuringSchedulingIgnoredDuringExecution = [],
          requiredDuringSchedulingIgnoredDuringExecution = {
            nodeSelectorTerms: [],
          },
        } = nodeAffinity;

        preferredDuringSchedulingIgnoredDuringExecution.forEach(
          ({ preference = { matchExpressions: [] }, weight }) => {
            rules.push({
              component,
              type: AffinityType.NodeAffinity,
              priority: AffinityPriority.Preferred,
              uid: generateShortUID(),
              weight,
              ...(preference.matchExpressions.length > 0 && {
                key: preference.matchExpressions[0].key,
                operator: preference.matchExpressions[0].operator,
                values: preference.matchExpressions[0].values?.join(','),
              }),
            });
          }
        );

        requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.forEach(
          ({ matchExpressions = [] }) => {
            rules.push({
              component,
              type: AffinityType.NodeAffinity,
              priority: AffinityPriority.Required,
              uid: generateShortUID(),
              ...(matchExpressions.length > 0 && {
                key: matchExpressions[0].key,
                operator: matchExpressions[0].operator,
                values: matchExpressions[0].values?.join(','),
              }),
            });
          }
        );
      }

      [podAffinity, podAntiAffinity].forEach((affinity) => {
        if (affinity) {
          const affinityType =
            affinity === podAffinity
              ? AffinityType.PodAffinity
              : AffinityType.PodAntiAffinity;
          const {
            preferredDuringSchedulingIgnoredDuringExecution = [],
            requiredDuringSchedulingIgnoredDuringExecution = [],
          } = affinity;
          preferredDuringSchedulingIgnoredDuringExecution.forEach(
            ({ weight, podAffinityTerm }) => {
              const { topologyKey, labelSelector = { matchExpressions: [] } } =
                podAffinityTerm;

              rules.push({
                component,
                type: affinityType,
                priority: AffinityPriority.Preferred,
                uid: generateShortUID(),
                weight,
                topologyKey,
                ...(labelSelector.matchExpressions.length > 0 && {
                  key: labelSelector.matchExpressions[0].key,
                  operator: labelSelector.matchExpressions[0].operator,
                  values: labelSelector.matchExpressions[0].values?.join(','),
                }),
              });
            }
          );

          requiredDuringSchedulingIgnoredDuringExecution.forEach(
            ({ topologyKey, labelSelector = { matchExpressions: [] } }) => {
              rules.push({
                component,
                type: affinityType,
                priority: AffinityPriority.Required,
                uid: generateShortUID(),
                topologyKey,
                ...(labelSelector.matchExpressions.length > 0 && {
                  key: labelSelector.matchExpressions[0].key,
                  operator: labelSelector.matchExpressions[0].operator,
                  values: labelSelector.matchExpressions[0].values?.join(','),
                }),
              });
            }
          );
        }
      });
    }
  });

  return rules.sort((a, b) => {
    if (a.component === b.component) {
      if (a.type === b.type) {
        if (a.priority === b.priority) {
          return (b.weight ?? 0) - (a.weight ?? 0);
        }
        return a.priority.localeCompare(b.priority);
      }
      return a.type.localeCompare(b.type);
    }
    return a.component.localeCompare(b.component);
  });
};

export const doesAffinityOperatorRequireValues = (
  operator: AffinityOperator
): boolean => [AffinityOperator.In, AffinityOperator.NotIn].includes(operator);

export const affinityRulesToDbPayload = (
  affinityRules: AffinityRule[]
): Affinity => {
  const generatePodAffinityTerm = (
    topologyKey: string,
    operator: AffinityOperator,
    values: string[],
    key?: string
  ): PodAffinityTerm => ({
    topologyKey: topologyKey!,
    ...(key && {
      labelSelector: {
        matchExpressions: [
          {
            key,
            operator: operator!,
            ...(doesAffinityOperatorRequireValues(operator!) &&
              values.length > 0 && {
                values,
              }),
          },
        ],
      },
    }),
  });

  type AffinityRulesPayloadMap = {
    [AffinityType.NodeAffinity]: {
      preferred: PreferredNodeSchedulingTerm[];
      required: RequiredNodeSchedulingTerm;
    };
  } & {
    [key in AffinityType.PodAffinity | AffinityType.PodAntiAffinity]: {
      preferred: PreferredPodSchedulingTerm[];
      required: RequiredPodSchedulingTerm;
    };
  };

  const map: AffinityRulesPayloadMap = {
    [AffinityType.NodeAffinity]: {
      preferred: [],
      required: { nodeSelectorTerms: [] },
    },
    [AffinityType.PodAffinity]: {
      preferred: [],
      required: [],
    },
    [AffinityType.PodAntiAffinity]: {
      preferred: [],
      required: [],
    },
  };

  affinityRules.forEach((rule) => {
    const { values = '', type, weight, key, operator, topologyKey } = rule;
    const valuesList = values !== '' ? values.split(',') : [];

    const required = rule.priority === AffinityPriority.Required;

    switch (type) {
      case AffinityType.NodeAffinity: {
        const matchExpressions: AffinityMatchExpression[] = [
          {
            key: key!,
            operator: operator!,
            ...(doesAffinityOperatorRequireValues(operator!) && {
              values: valuesList,
            }),
          },
        ];

        if (required) {
          map[AffinityType.NodeAffinity].required.nodeSelectorTerms.push({
            matchExpressions,
          });
        } else {
          map[AffinityType.NodeAffinity].preferred.push({
            weight: weight!,
            preference: {
              matchExpressions,
            },
          });
        }
        break;
      }

      case AffinityType.PodAntiAffinity:
      case AffinityType.PodAffinity: {
        const podAffinityTerm = generatePodAffinityTerm(
          topologyKey!,
          operator!,
          valuesList,
          key
        );
        if (required) {
          map[type].required.push(podAffinityTerm);
        } else {
          map[type].preferred.push({
            weight: weight!,
            podAffinityTerm,
          });
        }
        break;
      }
    }
  });
  const nodeAffinity: NodeAffinity = {
    ...(map[AffinityType.NodeAffinity].preferred.length > 0 && {
      preferredDuringSchedulingIgnoredDuringExecution:
        map[AffinityType.NodeAffinity].preferred,
    }),
    ...(map[AffinityType.NodeAffinity].required.nodeSelectorTerms.length >
      0 && {
      requiredDuringSchedulingIgnoredDuringExecution:
        map[AffinityType.NodeAffinity].required,
    }),
  };
  const podAffinity: PodAffinity = {
    ...(map[AffinityType.PodAffinity].preferred.length > 0 && {
      preferredDuringSchedulingIgnoredDuringExecution:
        map[AffinityType.PodAffinity].preferred,
    }),
    ...(map[AffinityType.PodAffinity].required.length > 0 && {
      requiredDuringSchedulingIgnoredDuringExecution:
        map[AffinityType.PodAffinity].required,
    }),
  };
  const podAntiAffinity: PodAntiAffinity = {
    ...(map[AffinityType.PodAntiAffinity].preferred.length > 0 && {
      preferredDuringSchedulingIgnoredDuringExecution:
        map[AffinityType.PodAntiAffinity].preferred,
    }),
    ...(map[AffinityType.PodAntiAffinity].required.length > 0 && {
      requiredDuringSchedulingIgnoredDuringExecution:
        map[AffinityType.PodAntiAffinity].required,
    }),
  };

  const finalObject: Affinity = {
    ...(Object.keys(nodeAffinity).length > 0 && {
      nodeAffinity,
    }),
    ...(Object.keys(podAffinity).length > 0 && {
      podAffinity,
    }),
    ...(Object.keys(podAntiAffinity).length > 0 && {
      podAntiAffinity,
    }),
  };

  return finalObject;
};

export const insertAffinityRuleToExistingPolicy = (
  policy: PodSchedulingPolicy,
  rule: AffinityRule
): PodSchedulingPolicy => {
  const dbType = policy.spec.engineType;
  const formattedRule = affinityRulesToDbPayload([rule]);

  if (!policy.spec.affinityConfig) {
    policy.spec.affinityConfig = {};
  }

  if (!policy.spec.affinityConfig[dbType]) {
    policy.spec.affinityConfig[dbType] = {};
  }

  if (!policy.spec.affinityConfig[dbType][rule.component]) {
    policy.spec.affinityConfig[dbType][rule.component] = {};
  }

  if (rule.type === AffinityType.NodeAffinity) {
    if (
      !policy.spec.affinityConfig[dbType][rule.component]?.nodeAffinity ||
      Object.keys(
        policy.spec.affinityConfig[dbType][rule.component]?.nodeAffinity || {}
      ).length === 0
    ) {
      policy.spec.affinityConfig[dbType][rule.component]!.nodeAffinity = {
        ...(rule.priority === AffinityPriority.Required
          ? {
              requiredDuringSchedulingIgnoredDuringExecution: {
                nodeSelectorTerms: [],
              },
            }
          : {
              preferredDuringSchedulingIgnoredDuringExecution: [],
            }),
      };
    }

    if (rule.priority === AffinityPriority.Required) {
      policy.spec.affinityConfig[dbType][
        rule.component
      ]!.nodeAffinity!.requiredDuringSchedulingIgnoredDuringExecution!.nodeSelectorTerms =
        [
          ...policy.spec.affinityConfig[dbType][rule.component]!.nodeAffinity!
            .requiredDuringSchedulingIgnoredDuringExecution!.nodeSelectorTerms,
          ...(formattedRule.nodeAffinity
            ?.requiredDuringSchedulingIgnoredDuringExecution
            ?.nodeSelectorTerms || []),
        ];
    } else {
      policy.spec.affinityConfig[dbType][
        rule.component
      ]!.nodeAffinity!.preferredDuringSchedulingIgnoredDuringExecution = [
        ...(policy.spec.affinityConfig[dbType][rule.component]!.nodeAffinity!
          .preferredDuringSchedulingIgnoredDuringExecution || []),
        ...(formattedRule.nodeAffinity
          ?.preferredDuringSchedulingIgnoredDuringExecution || []),
      ];
    }
  } else {
    const affinityType = rule.type;

    if (
      !policy.spec.affinityConfig[dbType][rule.component]![affinityType] ||
      Object.keys(
        policy.spec.affinityConfig[dbType][rule.component]![affinityType] || {}
      ).length === 0
    ) {
      policy.spec.affinityConfig[dbType][rule.component]![affinityType] = {
        ...(rule.priority === AffinityPriority.Required
          ? {
              requiredDuringSchedulingIgnoredDuringExecution: [],
            }
          : {
              preferredDuringSchedulingIgnoredDuringExecution: [],
            }),
      };
    }

    if (rule.priority === AffinityPriority.Required) {
      policy.spec.affinityConfig[dbType][rule.component]![
        affinityType
      ]!.requiredDuringSchedulingIgnoredDuringExecution = [
        ...(policy.spec.affinityConfig[dbType][rule.component]![affinityType]!
          .requiredDuringSchedulingIgnoredDuringExecution || []),
        ...(formattedRule[affinityType]
          ?.requiredDuringSchedulingIgnoredDuringExecution || []),
      ];
    } else {
      policy.spec.affinityConfig[dbType][rule.component]![
        affinityType
      ]!.preferredDuringSchedulingIgnoredDuringExecution = [
        ...(policy.spec.affinityConfig[dbType][rule.component]![affinityType]!
          .preferredDuringSchedulingIgnoredDuringExecution || []),
        ...(formattedRule[affinityType]
          ?.preferredDuringSchedulingIgnoredDuringExecution || []),
      ];
    }
  }

  return policy;
};

export const removeRuleInExistingPolicy = (
  policy: PodSchedulingPolicy,
  rule: AffinityRule
): PodSchedulingPolicy => {
  const { spec } = policy;
  const { engineType, affinityConfig } = spec;

  if (
    affinityConfig &&
    affinityConfig[engineType] &&
    affinityConfig[engineType][rule.component] &&
    affinityConfig[engineType][rule.component] &&
    affinityConfig[engineType][rule.component]![rule.type]
  ) {
    const obj = affinityConfig[engineType][rule.component]![rule.type];
    const { priority } = rule;

    if (priority === AffinityPriority.Required) {
      if (rule.type === AffinityType.NodeAffinity) {
        const matchingRuleIdx = (
          (
            (obj?.requiredDuringSchedulingIgnoredDuringExecution as RequiredNodeSchedulingTerm) ||
            {}
          ).nodeSelectorTerms || []
        ).findIndex(
          ({ matchExpressions }) =>
            matchExpressions.length &&
            matchExpressions[0].key === rule.key &&
            matchExpressions[0].operator === rule.operator &&
            matchExpressions[0].values?.join(',') === rule.values
        );

        if (matchingRuleIdx !== -1) {
          (
            obj?.requiredDuringSchedulingIgnoredDuringExecution as RequiredNodeSchedulingTerm
          ).nodeSelectorTerms.splice(matchingRuleIdx, 1);

          if (
            (
              obj?.requiredDuringSchedulingIgnoredDuringExecution as RequiredNodeSchedulingTerm
            ).nodeSelectorTerms.length === 0
          ) {
            delete obj?.requiredDuringSchedulingIgnoredDuringExecution;
          }
        }
      } else {
        const matchingRuleIdx = (
          (obj?.requiredDuringSchedulingIgnoredDuringExecution as RequiredPodSchedulingTerm) ||
          []
        ).findIndex(
          ({ topologyKey, labelSelector }) =>
            topologyKey === rule.topologyKey &&
            labelSelector?.matchExpressions[0].key === rule.key &&
            labelSelector?.matchExpressions[0].operator === rule.operator &&
            labelSelector?.matchExpressions[0].values?.join(',') === rule.values
        );

        if (matchingRuleIdx !== -1) {
          (
            obj?.requiredDuringSchedulingIgnoredDuringExecution as RequiredPodSchedulingTerm
          ).splice(matchingRuleIdx, 1);

          if (
            (
              obj?.requiredDuringSchedulingIgnoredDuringExecution as RequiredPodSchedulingTerm
            ).length === 0
          ) {
            delete obj?.requiredDuringSchedulingIgnoredDuringExecution;
          }
        }
      }
    } else {
      if (rule.type === AffinityType.NodeAffinity) {
        const matchingRuleIdx = (
          (obj?.preferredDuringSchedulingIgnoredDuringExecution as PreferredNodeSchedulingTerm[]) ||
          []
        ).findIndex(
          ({ weight, preference }) =>
            weight === rule.weight &&
            preference.matchExpressions[0].key === rule.key &&
            preference.matchExpressions[0].operator === rule.operator &&
            preference.matchExpressions[0].values?.join(',') === rule.values
        );

        if (matchingRuleIdx !== -1) {
          (
            obj?.preferredDuringSchedulingIgnoredDuringExecution as PreferredNodeSchedulingTerm[]
          ).splice(matchingRuleIdx, 1);

          if (
            (
              obj?.preferredDuringSchedulingIgnoredDuringExecution as PreferredNodeSchedulingTerm[]
            ).length === 0
          ) {
            delete obj?.preferredDuringSchedulingIgnoredDuringExecution;
          }
        }
      } else {
        const matchingRuleIdx = (
          (obj?.preferredDuringSchedulingIgnoredDuringExecution as PreferredPodSchedulingTerm[]) ||
          []
        ).findIndex(
          ({ weight, podAffinityTerm }) =>
            weight === rule.weight &&
            podAffinityTerm.topologyKey === rule.topologyKey &&
            podAffinityTerm.labelSelector?.matchExpressions[0].key ===
              rule.key &&
            podAffinityTerm.labelSelector?.matchExpressions[0].operator ===
              rule.operator &&
            podAffinityTerm.labelSelector?.matchExpressions[0].values?.join(
              ','
            ) === rule.values
        );

        if (matchingRuleIdx !== -1) {
          (
            obj?.preferredDuringSchedulingIgnoredDuringExecution as PreferredPodSchedulingTerm[]
          ).splice(matchingRuleIdx, 1);

          if (
            (
              obj?.preferredDuringSchedulingIgnoredDuringExecution as PreferredPodSchedulingTerm[]
            ).length === 0
          ) {
            delete obj?.preferredDuringSchedulingIgnoredDuringExecution;
          }
        }
      }
    }
  }

  if (
    affinityConfig &&
    affinityConfig[engineType] &&
    affinityConfig[engineType][rule.component] &&
    affinityConfig[engineType][rule.component]![rule.type] &&
    Object.keys(affinityConfig[engineType][rule.component]![rule.type] || {})
      .length === 0
  ) {
    delete affinityConfig[engineType][rule.component]![rule.type];
  }

  if (
    affinityConfig &&
    affinityConfig[engineType] &&
    affinityConfig[engineType][rule.component] &&
    Object.keys(affinityConfig[engineType][rule.component] || {}).length === 0
  ) {
    delete affinityConfig[engineType][rule.component];
  }

  if (
    affinityConfig &&
    Object.keys(affinityConfig[engineType] || {}).length === 0
  ) {
    delete affinityConfig[engineType];
  }

  if (Object.keys(affinityConfig || {}).length === 0) {
    // @ts-expect-error
    delete spec.affinityConfig;
  }

  return policy;
};

export const getAffinityRuleTypeLabel = (type: AffinityType): string => {
  switch (type) {
    case AffinityType.NodeAffinity:
      return 'Node Affinity';
    case AffinityType.PodAffinity:
      return 'Pod Affinity';
    case AffinityType.PodAntiAffinity:
      return 'Pod Anti-Affinity';
    default:
      return '';
  }
};

export const getAffinityComponentLabel = (
  dbType: DbType,
  component: AffinityComponent
): string => {
  switch (component) {
    case AffinityComponent.Proxy:
      return capitalize(getProxyUnitNamesFromDbType(dbType).singular);
    case AffinityComponent.ConfigServer:
      return 'Config Server';
    case AffinityComponent.DbNode:
      return 'DB Node';
    default:
      return '';
  }
};
export const changeDbClusterCrd = (
  dbCluster: DbCluster,
  newCrdVersion: string
): DbCluster => ({
  ...dbCluster,
  spec: {
    ...dbCluster.spec,
    engine: {
      ...dbCluster.spec.engine,
      crVersion: newCrdVersion,
    },
  },
});

export const changeDbClusterAdvancedConfig = (
  dbCluster: DbCluster,
  engineParametersEnabled = false,
  exposureMethod: ProxyExposeType,
  engineParameters = '',
  sourceRanges?: Array<{ sourceRange?: string }>,
  podSchedulingPolicyEnabled = false,
  podSchedulingPolicy = '',
  loadBalancerConfigName = '',
  splitHorizonDnsConfigName = ''
) => ({
  ...dbCluster,
  spec: {
    ...dbCluster.spec,
    ...(splitHorizonDnsConfigName && {
      engineFeatures: {
        ...dbCluster.spec.engineFeatures,
        psmdb: {
          ...dbCluster.spec.engineFeatures?.psmdb,
          splitHorizonDnsConfigName: splitHorizonDnsConfigName,
        },
      },
    }),
    podSchedulingPolicyName: podSchedulingPolicyEnabled
      ? podSchedulingPolicy
      : undefined,
    engine: {
      ...dbCluster.spec.engine,
      config: engineParametersEnabled ? engineParameters : '',
    },
    proxy: {
      ...dbCluster.spec.proxy,
      expose: {
        loadBalancerConfigName:
          (exposureMethod === ProxyExposeType.LoadBalancer &&
            loadBalancerConfigName !== EMPTY_LOAD_BALANCER_CONFIGURATION &&
            loadBalancerConfigName) ||
          undefined,
        type: exposureMethod,
        ...(exposureMethod === ProxyExposeType.LoadBalancer &&
          sourceRanges && {
            ipSourceRanges: sourceRanges.flatMap((source) =>
              source.sourceRange ? [source.sourceRange] : []
            ),
          }),
      },
    } as Proxy,
  },
});

export const changeDbClusterVersion = (
  dbCluster: DbCluster,
  dbVersion: string
) => ({
  ...dbCluster,
  spec: {
    ...dbCluster.spec,
    engine: {
      ...dbCluster.spec.engine,
      version: dbVersion,
    },
  },
});

export const changeDbClusterMonitoring = (
  dbCluster: DbCluster,
  monitoringName?: string
) => ({
  ...dbCluster,
  spec: {
    ...dbCluster.spec,
    monitoring: monitoringName
      ? {
          monitoringConfigName: monitoringName,
        }
      : {},
  },
});

export const changeDbClusterResources = (
  dbCluster: DbCluster,
  newResources: {
    cpu: number;
    memory: number;
    disk: number;
    diskUnit: string;
    numberOfNodes: number;
    proxyCpu: number;
    proxyMemory: number;
    numberOfProxies: number;
  },
  sharding = false,
  shardNr = '',
  shardConfigServers?: number
) => ({
  ...dbCluster,
  spec: {
    ...dbCluster.spec,
    engine: {
      ...dbCluster.spec.engine,
      replicas: newResources.numberOfNodes,
      resources: {
        cpu: `${newResources.cpu}`,
        memory: `${newResources.memory}G`,
      },
      storage: {
        ...dbCluster.spec.engine.storage,
        size: `${newResources.disk}${newResources.diskUnit}`,
      },
    },
    proxy: (() => {
      const exposeType = dbCluster.spec.proxy?.expose?.type;
      const mappedExposeType = mapDeprecatedExposeType(exposeType);

      return getProxySpec(
        dbEngineToDbType(dbCluster.spec.engine.type),
        newResources.numberOfProxies.toString(),
        '',
        mappedExposeType,
        newResources.proxyCpu,
        newResources.proxyMemory,
        !!sharding,
        ((dbCluster.spec.proxy as Proxy)?.expose?.ipSourceRanges || []).map(
          (sourceRange) => ({ sourceRange })
        ),
        mappedExposeType === ProxyExposeType.LoadBalancer
          ? dbCluster.spec.proxy?.expose?.loadBalancerConfigName
          : undefined
      );
    })(),
    ...(dbCluster.spec.engine.type === DbEngineType.PSMDB &&
      sharding && {
        sharding: {
          enabled: sharding,
          shards: +(shardNr ?? MIN_NUMBER_OF_SHARDS),
          configServer: {
            replicas: shardConfigServers ?? 3,
          },
        },
      }),
  },
});

export const changeDbClusterEngine = (
  dbCluster: DbCluster,
  newEngineVersion: string
) => ({
  ...dbCluster,
  spec: {
    ...dbCluster.spec,
    engine: {
      ...dbCluster.spec.engine,
      version: newEngineVersion,
    },
  },
});

export const changeDbClusterPITR = (
  dbCluster: DbCluster,
  enabled: boolean,
  backupStorageName: string | { name: string }
) => ({
  ...dbCluster,
  spec: {
    ...dbCluster.spec,
    backup: {
      ...dbCluster.spec.backup!,
      pitr: enabled
        ? {
            backupStorageName:
              typeof backupStorageName === 'string'
                ? backupStorageName
                : backupStorageName!.name,
            enabled: true,
          }
        : { enabled: false, backupStorageName: '' },
    },
  },
});

export const deleteScheduleFromDbCluster = (
  scheduleName: string,
  dbCluster: DbCluster,
  disablePITR: boolean
): DbCluster => {
  const schedules = dbCluster?.spec?.backup?.schedules || [];
  const filteredSchedulesWithCronCorrection = schedules.reduce(
    (result: Schedule[], schedule) => {
      if (schedule?.name !== scheduleName) {
        result.push(schedule);
      }
      return result;
    },
    []
  );

  return {
    apiVersion: 'everest.percona.com/v1alpha1',
    kind: 'DatabaseCluster',
    metadata: dbCluster.metadata,
    spec: {
      ...dbCluster?.spec,
      backup: {
        ...dbCluster.spec.backup,
        ...(disablePITR && {
          pitr: {
            ...dbCluster.spec.backup?.pitr,
            backupStorageName:
              dbCluster.spec.backup?.pitr?.backupStorageName || '',
            enabled: false,
          },
        }),
        schedules:
          filteredSchedulesWithCronCorrection.length > 0
            ? filteredSchedulesWithCronCorrection
            : undefined,
      },
    },
  };
};

export const setDbClusterPausedStatus = (
  dbCluster: DbCluster,
  paused: boolean
) => ({
  ...dbCluster,
  spec: {
    ...dbCluster.spec,
    paused,
  },
});

export const setDbClusterRestart = (dbCluster: DbCluster) => ({
  ...dbCluster,
  metadata: {
    ...dbCluster.metadata,
    annotations: {
      'everest.percona.com/restart': 'true',
    },
  },
});

const humanizedDbMap: Record<DbType, string> = {
  [DbType.Postresql]: 'PostgreSQL',
  [DbType.Mongo]: 'MongoDB',
  [DbType.Mysql]: 'MySQL',
};

export const humanizeDbType = (type: DbType): string => humanizedDbMap[type];

// This does not apply to the delete action, which is only blocked when the db is being deleted itself
export const shouldDbActionsBeBlocked = (status?: DbClusterStatus) => {
  return [
    DbClusterStatus.restoring,
    DbClusterStatus.deleting,
    DbClusterStatus.resizingVolumes,
    DbClusterStatus.upgrading,
    DbClusterStatus.importing,
  ].includes(status || ('' as DbClusterStatus));
};

export const mergeNewDbClusterData = (
  oldDbClusterData: DbCluster = {} as DbCluster,
  newDbClusterData: DbCluster,
  // When using setQueryData, the data is already in UTC and will be used by queryClient BEFORE `select`, so it has to be in UTC, as if coming from the API
  // In those cases, we don't want to convert the schedule to local timezone
  convertToLocalTimezone: boolean
): DbCluster => {
  const newCluster = {
    ...oldDbClusterData,
    ...newDbClusterData,
    spec: {
      ...newDbClusterData.spec,
      ...(newDbClusterData.spec?.backup?.schedules && {
        backup: {
          ...newDbClusterData.spec.backup,
          schedules: newDbClusterData.spec.backup.schedules.map((schedule) => ({
            ...schedule,
            schedule: convertToLocalTimezone
              ? cronConverter(
                  schedule.schedule,
                  'UTC',
                  Intl.DateTimeFormat().resolvedOptions().timeZone
                )
              : schedule.schedule,
          })),
        },
      }),
    },
  };

  return newCluster;
};

export const fileToBase64 = (file: File) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => {
      const result = reader.result as string;
      const base64 = result.split(',')[1];
      resolve(base64);
    };
    reader.onerror = reject;
  });

export const isSplitHorizonDNSEnabled = (dbType: DbType): boolean => {
  return dbType === DbType.Mongo;
};
