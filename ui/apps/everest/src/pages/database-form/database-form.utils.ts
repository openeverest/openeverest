// everest
// Copyright (C) 2023 Percona LLC
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

import { DbEngineType, DbType } from '@percona/types';
import { DbCluster } from 'shared-types/dbCluster.types';
import { DbWizardFormFields } from 'consts.ts';
import { dbEngineToDbType } from '@percona/utils';
import {
  cpuParser,
  memoryParser,
  extractResourceCpuValue,
  extractResourceMemoryValue,
  extractResourceCpuRequestValue,
  extractResourceMemoryRequestValue,
  areResourceRequestsSynced,
} from 'utils/k8ResourceParser';
import { MAX_DB_CLUSTER_NAME_LENGTH } from 'consts';
import { DbWizardType } from './database-form-schema.ts';
import { DEFAULT_NODES } from './database-form.constants.ts';
import { generateShortUID } from 'utils/generateShortUID';
import {
  CUSTOM_NR_UNITS_INPUT_VALUE,
  getDefaultNumberOfconfigServersByNumberOfNodes,
  matchFieldsValueToResourceSize,
  NODES_DB_TYPE_MAP,
  ResourceSize,
  NODES_DEFAULT_SIZES,
  PROXIES_DEFAULT_SIZES,
} from 'components/cluster-form';
import { isProxy } from 'utils/db.tsx';
import { advancedConfigurationModalDefaultValues } from 'components/cluster-form/advanced-configuration/advanced-configuration.utils.ts';
import { WizardMode } from 'shared-types/wizard.types.ts';
import { ProxyExposeType } from 'shared-types/dbCluster.types';

export const getDbWizardDefaultValues = (dbType: DbType): DbWizardType => ({
  // TODO should be changed to true after  https://jira.percona.com/browse/EVEREST-509
  [DbWizardFormFields.schedules]: [],
  [DbWizardFormFields.pitrEnabled]: false,
  [DbWizardFormFields.pitrStorageLocation]: null,
  // @ts-ignore
  [DbWizardFormFields.storageLocation]: null,
  [DbWizardFormFields.dbType]: dbType,
  [DbWizardFormFields.dbName]: `${dbType}-${generateShortUID()}`,
  [DbWizardFormFields.dbVersion]: '',
  [DbWizardFormFields.storageClass]: '',
  [DbWizardFormFields.k8sNamespace]: null,
  [DbWizardFormFields.sourceRanges]: [{ sourceRange: '' }],
  [DbWizardFormFields.podSchedulingPolicyEnabled]: false,
  [DbWizardFormFields.podSchedulingPolicy]: '',
  [DbWizardFormFields.exposureMethod]: ProxyExposeType.ClusterIP,
  [DbWizardFormFields.loadBalancerConfigName]: '',
  [DbWizardFormFields.engineParametersEnabled]: false,
  [DbWizardFormFields.engineParameters]: '',
  [DbWizardFormFields.proxyConfigEnabled]: false,
  [DbWizardFormFields.proxyConfig]: '',
  [DbWizardFormFields.monitoring]: false,
  [DbWizardFormFields.monitoringInstance]: '',
  [DbWizardFormFields.numberOfNodes]: DEFAULT_NODES[dbType],
  [DbWizardFormFields.numberOfProxies]: DEFAULT_NODES[dbType],
  [DbWizardFormFields.resourceSizePerNode]: ResourceSize.small,
  [DbWizardFormFields.resourceSizePerProxy]: ResourceSize.small,
  [DbWizardFormFields.customNrOfNodes]: DEFAULT_NODES[dbType],
  [DbWizardFormFields.customNrOfProxies]: DEFAULT_NODES[dbType],
  [DbWizardFormFields.cpu]: NODES_DEFAULT_SIZES(dbType).small.cpu,
  [DbWizardFormFields.cpuRequests]: NODES_DEFAULT_SIZES(dbType).small.cpu,
  [DbWizardFormFields.proxyCpu]: PROXIES_DEFAULT_SIZES[dbType].small.cpu,
  [DbWizardFormFields.proxyCpuRequests]:
    PROXIES_DEFAULT_SIZES[dbType].small.cpu,
  [DbWizardFormFields.disk]: NODES_DEFAULT_SIZES(dbType).small.disk,
  [DbWizardFormFields.diskUnit]: 'Gi',
  [DbWizardFormFields.memory]: NODES_DEFAULT_SIZES(dbType).small.memory,
  [DbWizardFormFields.memoryRequests]: NODES_DEFAULT_SIZES(dbType).small.memory,
  [DbWizardFormFields.proxyMemory]: PROXIES_DEFAULT_SIZES[dbType].small.memory,
  [DbWizardFormFields.proxyMemoryRequests]:
    PROXIES_DEFAULT_SIZES[dbType].small.memory,
  [DbWizardFormFields.nodeRequestsSynced]: true,
  [DbWizardFormFields.proxyRequestsSynced]: true,
  [DbWizardFormFields.sharding]: false,
  [DbWizardFormFields.shardNr]: '2',
  [DbWizardFormFields.shardConfigServers]:
    getDefaultNumberOfconfigServersByNumberOfNodes(
      parseInt(DEFAULT_NODES[DbType.Mongo], 10)
    ),
  [DbWizardFormFields.dataImporter]: '',
  [DbWizardFormFields.bucketName]: '',
  [DbWizardFormFields.region]: '',
  [DbWizardFormFields.endpoint]: '',
  [DbWizardFormFields.accessKey]: '',
  [DbWizardFormFields.secretKey]: '',
  [DbWizardFormFields.filePath]: '',
  [DbWizardFormFields.verifyTlS]: true,
  [DbWizardFormFields.forcePathStyle]: false,
  [DbWizardFormFields.credentials]: {},
  [DbWizardFormFields.splitHorizonDNS]: '',
  [DbWizardFormFields.splitHorizonDNSEnabled]: false,
});

const replicasToNodes = (replicas: string, dbType: DbType): string => {
  const nodeOptions = NODES_DB_TYPE_MAP[dbType];
  const replicasString = replicas.toString();

  if (nodeOptions.includes(replicasString)) {
    return replicasString;
  }

  return CUSTOM_NR_UNITS_INPUT_VALUE;
};

export const DbClusterPayloadToFormValues = (
  dbCluster: DbCluster,
  mode: WizardMode,
  namespace: string
): DbWizardType => {
  const dbType = dbEngineToDbType(dbCluster?.spec?.engine?.type);
  const defaults = getDbWizardDefaultValues(dbType);
  const backup = dbCluster?.spec?.backup;
  const replicas = dbCluster?.spec?.engine?.replicas.toString();
  const proxies = (
    isProxy(dbCluster?.spec?.proxy) ? dbCluster?.spec?.proxy?.replicas || 0 : 0
  ).toString();
  const diskValues = memoryParser(
    dbCluster?.spec?.engine?.storage?.size.toString()
  );

  const sharding = dbCluster?.spec?.sharding;
  const numberOfNodes = replicasToNodes(replicas, dbType);

  const engineResources = dbCluster?.spec?.engine?.resources;
  const proxyResources = isProxy(dbCluster?.spec?.proxy)
    ? dbCluster?.spec?.proxy?.resources
    : undefined;

  const nodeCpuLimit = cpuParser(
    extractResourceCpuValue(engineResources).toString()
  );
  const nodeMemoryLimit = memoryParser(
    extractResourceMemoryValue(engineResources).toString(),
    'G'
  ).value;
  const proxyCpuLimit = proxyResources
    ? cpuParser(extractResourceCpuValue(proxyResources).toString())
    : 0;
  const proxyMemoryLimit = proxyResources
    ? memoryParser(extractResourceMemoryValue(proxyResources).toString(), 'G')
        .value
    : 0;

  const rawNodeCpuRequest = extractResourceCpuRequestValue(engineResources);
  const rawNodeMemoryRequest =
    extractResourceMemoryRequestValue(engineResources);
  const rawProxyCpuRequest = extractResourceCpuRequestValue(proxyResources);
  const rawProxyMemoryRequest =
    extractResourceMemoryRequestValue(proxyResources);

  const nodeCpuRequest =
    rawNodeCpuRequest !== undefined
      ? cpuParser(rawNodeCpuRequest.toString())
      : undefined;
  const nodeMemoryRequest =
    rawNodeMemoryRequest !== undefined
      ? memoryParser(rawNodeMemoryRequest.toString(), 'G').value
      : undefined;
  const proxyCpuRequest =
    rawProxyCpuRequest !== undefined
      ? cpuParser(rawProxyCpuRequest.toString())
      : undefined;
  const proxyMemoryRequest =
    rawProxyMemoryRequest !== undefined
      ? memoryParser(rawProxyMemoryRequest.toString(), 'G').value
      : undefined;

  // Requests are "synced" with limits when they are absent or identical to the
  // limits. When they differ, the user configured them separately (desynced).
  const nodeRequestsSynced = areResourceRequestsSynced(engineResources);
  const proxyRequestsSynced = areResourceRequestsSynced(proxyResources);

  return {
    //basic info
    [DbWizardFormFields.k8sNamespace]:
      namespace || defaults[DbWizardFormFields.k8sNamespace],
    [DbWizardFormFields.dbType]: dbType,
    [DbWizardFormFields.dbName]:
      mode === WizardMode.Restore
        ? `${dbCluster?.metadata?.name.slice(
            0,
            MAX_DB_CLUSTER_NAME_LENGTH - 4
          )}-${generateShortUID()}`
        : dbCluster?.metadata?.name,
    [DbWizardFormFields.dbVersion]: dbCluster?.spec?.engine?.version || '',

    //resources
    [DbWizardFormFields.numberOfNodes]: numberOfNodes,
    [DbWizardFormFields.numberOfProxies]: replicasToNodes(proxies, dbType),
    [DbWizardFormFields.customNrOfNodes]: replicas,
    [DbWizardFormFields.customNrOfProxies]: proxies,
    [DbWizardFormFields.resourceSizePerNode]: matchFieldsValueToResourceSize(
      NODES_DEFAULT_SIZES(dbType, dbCluster?.spec?.engine?.version || ''),
      dbCluster?.spec?.engine?.resources
    ),
    [DbWizardFormFields.resourceSizePerProxy]: isProxy(dbCluster?.spec?.proxy)
      ? matchFieldsValueToResourceSize(
          PROXIES_DEFAULT_SIZES[dbType],
          dbCluster?.spec?.proxy.resources
        )
      : ResourceSize.small,
    [DbWizardFormFields.sharding]: dbCluster?.spec?.sharding?.enabled || false,
    [DbWizardFormFields.shardConfigServers]:
      sharding?.configServer?.replicas ||
      getDefaultNumberOfconfigServersByNumberOfNodes(+numberOfNodes),
    [DbWizardFormFields.shardNr]: (
      sharding?.shards || (defaults[DbWizardFormFields.shardNr] as string)
    ).toString(),
    [DbWizardFormFields.cpu]: nodeCpuLimit,
    [DbWizardFormFields.proxyCpu]: proxyCpuLimit,
    [DbWizardFormFields.disk]: diskValues.value,
    [DbWizardFormFields.diskUnit]: diskValues.originalUnit,
    [DbWizardFormFields.memory]: nodeMemoryLimit,
    [DbWizardFormFields.cpuRequests]: nodeRequestsSynced
      ? nodeCpuLimit
      : nodeCpuRequest,
    [DbWizardFormFields.memoryRequests]: nodeRequestsSynced
      ? nodeMemoryLimit
      : nodeMemoryRequest,
    [DbWizardFormFields.proxyMemory]: proxyMemoryLimit,
    [DbWizardFormFields.proxyCpuRequests]: proxyRequestsSynced
      ? proxyCpuLimit
      : proxyCpuRequest,
    [DbWizardFormFields.proxyMemoryRequests]: proxyRequestsSynced
      ? proxyMemoryLimit
      : proxyMemoryRequest,
    [DbWizardFormFields.nodeRequestsSynced]: nodeRequestsSynced,
    [DbWizardFormFields.proxyRequestsSynced]: proxyRequestsSynced,
    //backups

    [DbWizardFormFields.backupsEnabled]: (backup?.schedules || []).length > 0,
    [DbWizardFormFields.pitrEnabled]:
      dbCluster?.spec?.engine?.type === DbEngineType.POSTGRESQL
        ? (backup?.schedules || []).length > 0
        : backup?.pitr?.enabled || false,
    [DbWizardFormFields.pitrStorageLocation]:
      backup?.pitr?.enabled && mode === WizardMode.Restore
        ? backup?.pitr?.backupStorageName || null
        : defaults[DbWizardFormFields.pitrStorageLocation],
    [DbWizardFormFields.schedules]: backup?.schedules || [],

    //advanced configuration
    ...advancedConfigurationModalDefaultValues(dbCluster),

    //monitoring
    [DbWizardFormFields.monitoring]:
      !!dbCluster?.spec?.monitoring?.monitoringConfigName,
    [DbWizardFormFields.monitoringInstance]:
      dbCluster?.spec?.monitoring?.monitoringConfigName || '',
  };
};
