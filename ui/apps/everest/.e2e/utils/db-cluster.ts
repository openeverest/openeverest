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

import { APIRequestContext, expect } from '@playwright/test';
import { dbTypeToDbEngine, dbTypeToProxyType } from '@percona/utils';
import { getEnginesVersions } from './database-engines';
import { getClusterDetailedInfo } from './storage-class';
import { getCITokenFromLocalStorage } from './localStorage';
import { getNamespacesFn } from './namespaces';
import { DbType } from '@percona/types';
import { execSync } from 'child_process';
import { checkError } from '@e2e/utils/generic';
import { getVersionServiceDBVersions } from '@e2e/utils/version-service';
import { TIMEOUTS } from '@e2e/constants';

export const createDbClusterFn = async (
  request: APIRequestContext,
  customOptions?,
  desiredNamespace?: string
) => {
  const token = await getCITokenFromLocalStorage();
  const namespaces = await getNamespacesFn(token, request);
  const namespace = desiredNamespace ?? namespaces[0];
  const dbEngines = await getEnginesVersions(token, namespace, request);
  const dbType = customOptions?.dbType || 'postgresql';
  const dbEngineType = dbTypeToDbEngine(dbType);
  const dbTypeVersions = dbEngines[dbEngineType];
  const dbClusterInfo = await getClusterDetailedInfo(token, request);
  const storageClassNames = dbClusterInfo?.storageClassNames[0];
  const lastVersion = dbTypeVersions[dbTypeVersions.length - 1];

  // const payload: DbCluster = {
  //         proxy: {
  //             replicas: +customOptions?.numberOfNodes || 1,
  //             expose: {
  //                 type:
  //                     customOptions?.externalAccess || false
  //                         ? ProxyExposeType.external
  //                         : ProxyExposeType.internal,
  //                 ...(!!(customOptions?.externalAccess || false) &&
  //                     (customOptions?.sourceRange || ['181.170.213.40']) && {
  //                         ipSourceRanges: [customOptions?.sourceRange] || [
  //                             '181.170.213.40',
  //                         ],
  //                     }),
  //             },
  //         },
  //     },
  // };

  const payload = {
    apiVersion: 'everest.percona.com/v1alpha1',
    kind: 'DatabaseCluster',
    metadata: {
      name: customOptions?.dbName || 'db-cluster-test-ui',
      namespace,
    },
    spec: {
      engine: {
        type: dbEngineType,
        version: customOptions?.dbVersion || lastVersion,
        replicas: +(customOptions?.numberOfNodes || 1),
        resources: {
          cpu: `${customOptions?.cpu || 1}`,
          memory: `${customOptions?.memory || 1}G`,
        },
        storage: {
          class: customOptions?.storageClass! || storageClassNames,
          size: `${customOptions?.disk || 1}Gi`,
        },
        // TODO return engineParams to tests
        // config: dbPayload.engineParametersEnabled
        //     ? dbPayload.engineParameters
        //     : '',
      },
      ...(customOptions?.backup && {
        backup: customOptions?.backup,
      }),
      // TODO return monitoring to tests
      monitoring: {
        ...(customOptions?.monitoringConfigName && {
          monitoringConfigName: customOptions?.monitoringConfigName,
        }),
        // ...(!!dbPayload.monitoring && {
        //     monitoringConfigName:
        //         typeof dbPayload.monitoringInstance === 'string'
        //             ? dbPayload.monitoringInstance
        //             : dbPayload?.monitoringInstance!.name,
        // }),
      },
      proxy: {
        type: dbTypeToProxyType(dbType),
        replicas: +(customOptions?.numberOfProxies || 1),
        resources: {
          cpu: `${customOptions?.proxyCpu || 1}`,
          memory: `${customOptions?.proxyMemory || 1}G`,
        },
        expose: {
          type: customOptions?.externalAccess ? 'LoadBalancer' : 'ClusterIP',
          ...(!!customOptions?.externalAccess &&
            customOptions?.sourceRanges && {
              ipSourceRanges: customOptions?.sourceRanges.flatMap((source) =>
                source.sourceRange ? [source.sourceRange] : []
              ),
            }),
        },
      },
      ...(customOptions.sharding &&
        dbType === DbType.Mongo && {
          sharding: {
            enabled: true,
            shards: customOptions.shards || 1,
            configServer: {
              replicas: customOptions.configServerReplicas || 3,
            },
          },
        }),
      // TODO return for backups tests
      // ...(backupDataSource?.dbClusterBackupName && {
      //     dataSource: {
      //         dbClusterBackupName: backupDataSource.dbClusterBackupName,
      //     },
      // }),
    },
  };

  await expect(async () => {
    const response = await request.post(
      `/v1/namespaces/${namespace}/database-clusters`,
      {
        data: payload,
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }
    );

    expect(response.ok()).toBeTruthy();
  }).toPass({
    intervals: [1000],
    timeout: TIMEOUTS.TenSeconds,
  });
};

export const deleteDbClusterFn = async (
  request: APIRequestContext,
  clusterName: string,
  desiredNamespace?: string
) => {
  const token = await getCITokenFromLocalStorage();
  const namespaces = await getNamespacesFn(token, request);
  const namespace = desiredNamespace ?? namespaces[0];
  const deleteResponse = await request.delete(
      `/v1/namespaces/${namespace}/database-clusters/${clusterName}`,
      {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }
    ),
    code = deleteResponse.status();
  expect(code === 204 || code === 404).toBeTruthy();
};

export const patchPSMDBFinalizers = async (
  cluster: string,
  namespace: string
) => {
  try {
    const command = `kubectl patch --namespace ${namespace} psmdb ${cluster} --type='merge' -p '{"metadata":{"finalizers":["percona.com/delete-psmdb-pvc"]}}'`;
    const output = execSync(command).toString();
    return true;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const getDbClusterAPI = async (
  clusterName: string,
  namespace: string,
  request: APIRequestContext,
  token: string
) => {
  const response = await request.get(
    `/v1/namespaces/${namespace}/database-clusters/${clusterName}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  await checkError(response);

  return await response.json();
};

export const updateDbClusterAPI = async (
  clusterName: string,
  namespace: string,
  payload: object,
  request: APIRequestContext,
  token: string
) => {
  const updatedCluster = await request.put(
    `/v1/namespaces/${namespace}/database-clusters/${clusterName}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      data: payload,
    }
  );
  await checkError(updatedCluster);
};

export const getDbAvailableUpgradeVersionK8S = async (
  clusterName: string,
  namespace: string,
  request: APIRequestContext,
  token: string
) => {
  const cluster = await getDbClusterAPI(clusterName, namespace, request, token);
  const dbCurrentVersion = cluster.spec.engine.version;
  const dbSplitVersion = dbCurrentVersion.split('.');
  const dbType = cluster.spec.engine.type;
  const crVersion = cluster.status.crVersion;

  const dbMajorVersion =
    dbType === 'postgresql'
      ? dbSplitVersion[0]
      : dbSplitVersion[0] + '.' + dbSplitVersion[1];

  try {
    const versions = await getVersionServiceDBVersions(
      dbType,
      crVersion,
      request,
      dbMajorVersion
    );
    const dbUpgradeVersion = versions[0];

    // return latest version only if different than current one
    return dbUpgradeVersion === dbCurrentVersion ? null : dbUpgradeVersion;
  } catch (error) {
    console.error('Error extracting database version:', error);
    return null;
  }
};
