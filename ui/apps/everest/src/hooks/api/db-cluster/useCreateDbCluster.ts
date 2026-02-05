// everest
// Copyright (C) 2023 Percona LLC
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

import { dbTypeToDbEngine } from '@percona/utils';
import {
  UseMutationOptions,
  useMutation,
  useQuery,
} from '@tanstack/react-query';
import {
  createDbClusterFn,
  createDbClusterSecretFn,
  getDbClusterCredentialsFn,
} from 'api/dbClusterApi';
import {
  CUSTOM_NR_UNITS_INPUT_VALUE,
  MIN_NUMBER_OF_SHARDS,
} from 'components/cluster-form';
import { DbWizardType } from 'pages/database-form/database-form-schema.ts';
import {
  ClusterCredentials,
  DataSource,
  DbCluster,
  GetDbClusterCredentialsPayload,
  ProxyExposeType,
} from 'shared-types/dbCluster.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import cronConverter from 'utils/cron-converter';
import { getDataSource, getProxySpec } from './utils';
import { DbType } from '@percona/types';
import { useRBACPermissions } from 'hooks/rbac';
import { EMPTY_LOAD_BALANCER_CONFIGURATION } from 'consts';

type CreateDbClusterArgType = {
  dbPayload: DbWizardType;
  backupDataSource?: DataSource;
};

const formValuesToPayloadMapping = (
  dbPayload: DbWizardType,
  backupDataSource?: DataSource
): DbCluster => {
  const numberOfNodes = parseInt(
    dbPayload.numberOfNodes === CUSTOM_NR_UNITS_INPUT_VALUE
      ? dbPayload.customNrOfNodes || ''
      : dbPayload.numberOfNodes,
    10
  );

  const dbClusterPayload: DbCluster = {
    apiVersion: 'everest.percona.com/v1alpha1',
    kind: 'DatabaseCluster',
    metadata: {
      name: dbPayload.dbName,
      namespace: dbPayload.k8sNamespace || '',
    },
    spec: {
      backup: {
        ...(dbPayload.pitrEnabled && {
          pitr: {
            enabled:
              dbPayload.dbType === DbType.Postresql
                ? dbPayload.schedules.length > 0
                : dbPayload.pitrEnabled,
            backupStorageName:
              dbPayload.dbType === DbType.Postresql
                ? ''
                : typeof dbPayload.pitrStorageLocation === 'string'
                  ? dbPayload.pitrStorageLocation
                  : dbPayload.pitrStorageLocation!.name,
          },
        }),
        schedules:
          dbPayload.schedules?.length > 0
            ? dbPayload.schedules.map((schedule) => ({
                ...schedule,
                schedule: cronConverter(
                  schedule.schedule,
                  Intl.DateTimeFormat().resolvedOptions().timeZone,
                  'UTC'
                ),
              }))
            : undefined,
      },
      engine: {
        type: dbTypeToDbEngine(dbPayload.dbType),
        version: dbPayload.dbVersion,
        replicas: numberOfNodes,
        resources: {
          cpu: `${dbPayload.cpu}`,
          memory: `${dbPayload.memory}G`,
        },
        storage: {
          class: dbPayload.storageClass!,
          size: `${dbPayload.disk}${dbPayload.diskUnit}`,
        },
        config: dbPayload.engineParametersEnabled
          ? dbPayload.engineParameters
          : '',
        ...(dbPayload.dataImporter &&
        dbPayload.credentials &&
        Object.keys(dbPayload.credentials).length > 0
          ? {
              userSecretsName: `everest-secrets-${dbPayload.dbName}`,
            }
          : {}),
      },
      monitoring: {
        ...(!!dbPayload.monitoring && {
          monitoringConfigName: dbPayload?.monitoringInstance!,
        }),
      },
      proxy: getProxySpec(
        dbPayload.dbType,
        dbPayload.numberOfProxies,
        dbPayload.customNrOfProxies || '',
        dbPayload.exposureMethod,
        dbPayload.proxyCpu,
        dbPayload.proxyMemory,
        dbPayload.sharding,
        dbPayload.sourceRanges || [],
        dbPayload.exposureMethod === ProxyExposeType.LoadBalancer ||
          dbPayload.loadBalancerConfigName === EMPTY_LOAD_BALANCER_CONFIGURATION
          ? dbPayload.loadBalancerConfigName
          : undefined
      ),
      ...(dbPayload.dbType === DbType.Mongo && {
        sharding: {
          enabled: dbPayload.sharding,
          shards: +(dbPayload.shardNr ?? MIN_NUMBER_OF_SHARDS),
          configServer: {
            replicas: +(dbPayload.shardConfigServers ?? 3),
          },
        },
      }),
      ...(backupDataSource?.dbClusterBackupName || dbPayload.dataImporter
        ? {
            dataSource: {
              ...getDataSource({
                backupDataSource: backupDataSource,
                dbPayload: dbPayload,
              }),
            },
          }
        : {}),
      ...(dbPayload.podSchedulingPolicyEnabled &&
        dbPayload.podSchedulingPolicy && {
          podSchedulingPolicyName: dbPayload.podSchedulingPolicy,
        }),
      ...(!!dbPayload.externalAccess && {
        spec: {
          proxy: {
            expose: {
              loadBalancerConfigName: dbPayload.loadBalancerConfig,
              type: ProxyExposeType.LoadBalancer,
            },
          },
        },
      }),
      ...(dbPayload.splitHorizonDNSEnabled &&
        dbPayload.splitHorizonDNS && {
          engineFeatures: {
            psmdb: {
              splitHorizonDnsConfigName: dbPayload.splitHorizonDNS,
            },
          },
        }),
    },
  };

  return dbClusterPayload;
};

export const useCreateDbCluster = (
  options?: UseMutationOptions<
    DbCluster,
    unknown,
    CreateDbClusterArgType,
    unknown
  >
) =>
  useMutation({
    mutationFn: ({ dbPayload, backupDataSource }: CreateDbClusterArgType) =>
      createDbClusterFn(
        formValuesToPayloadMapping(dbPayload, backupDataSource),
        dbPayload.k8sNamespace || ''
      ),
    ...options,
  });

export const useDbClusterCredentials = (
  dbClusterName: string,
  namespace: string,
  options?: PerconaQueryOptions<ClusterCredentials, unknown, ClusterCredentials>
) => {
  const { canRead: canReadCredentials } = useRBACPermissions(
    'database-cluster-credentials',
    `${namespace}/${dbClusterName}`
  );

  return useQuery<GetDbClusterCredentialsPayload, unknown, ClusterCredentials>({
    queryKey: ['cluster-credentials', dbClusterName],
    queryFn: () => getDbClusterCredentialsFn(dbClusterName, namespace),
    select: canReadCredentials
      ? (creds) => creds
      : () => ({ username: '', password: '' }),
    ...options,
    enabled: (options?.enabled ?? true) && canReadCredentials,
  });
};

type CreateDbClusterSecretProps = {
  dbClusterName: string;
  namespace: string;
  credentials: Record<string, string>;
};

export const useCreateDbClusterSecret = () =>
  useMutation({
    mutationFn: ({
      dbClusterName,
      namespace,
      credentials,
    }: CreateDbClusterSecretProps) =>
      createDbClusterSecretFn(dbClusterName, namespace, credentials),
  });
