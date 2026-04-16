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

import { DbType } from '@percona/types';
import { dbTypeToProxyType } from '@percona/utils';
import { EMPTY_LOAD_BALANCER_CONFIGURATION } from 'consts';
import { DbWizardType } from 'pages/database-form/database-form-schema';
import {
  DataSource,
  Proxy,
  ProxyExposeConfig,
  ProxyExposeType,
} from 'shared-types/dbCluster.types';

const getExposteConfig = (
  exposureMethod: ProxyExposeType,
  loadBalancerConfigName?: string,
  sourceRanges?: Array<{ sourceRange?: string }>
): ProxyExposeConfig => {
  return {
    type: exposureMethod,
    loadBalancerConfigName,
    ...(exposureMethod === ProxyExposeType.LoadBalancer &&
      sourceRanges && {
        ipSourceRanges: sourceRanges.flatMap((source) =>
          source.sourceRange ? [source.sourceRange] : []
        ),
      }),
  };
};

export const getProxySpec = (
  dbType: DbType,
  numberOfProxies: string,
  customNrOfProxies: string,
  exposureMethod: ProxyExposeType,
  cpu: number,
  memory: number,
  sharding: boolean,
  sourceRanges?: Array<{ sourceRange?: string }>,
  loadBalancerConfigName?: string
): Proxy => {
  if (dbType === DbType.Mongo && !sharding) {
    return {
      expose: getExposteConfig(
        exposureMethod,
        loadBalancerConfigName !== EMPTY_LOAD_BALANCER_CONFIGURATION
          ? loadBalancerConfigName
          : '',
        sourceRanges
      ),
    } as unknown as Proxy;
  }
  const proxyNr = parseInt(
    numberOfProxies === 'custom' ? customNrOfProxies : numberOfProxies,
    10
  );

  return {
    type: dbTypeToProxyType(dbType),
    replicas: proxyNr,
    resources: {
      cpu: `${cpu}`,
      memory: `${memory}G`,
    },
    expose: getExposteConfig(
      exposureMethod,
      loadBalancerConfigName !== EMPTY_LOAD_BALANCER_CONFIGURATION
        ? loadBalancerConfigName
        : '',
      sourceRanges
    ),
  };
};

export const getDataSource = ({
  backupDataSource,
  dbPayload,
}: {
  backupDataSource?: DataSource;
  dbPayload: DbWizardType;
}) => {
  let dataSource = {};
  if (backupDataSource?.dbClusterBackupName) {
    dataSource = {
      dbClusterBackupName: backupDataSource.dbClusterBackupName,
      ...(backupDataSource?.pitr && {
        pitr: {
          date: backupDataSource.pitr.date,
          type: 'date',
        },
      }),
    };
  }
  if ('dataImporter' in dbPayload && dbPayload.dataImporter) {
    dataSource = {
      ...dataSource,
      dataImport: {
        dataImporterName: dbPayload.dataImporter,
        source: {
          path: dbPayload.filePath,
          s3: {
            accessKeyId: dbPayload.accessKey,
            bucket: dbPayload.bucketName,
            credentialsSecretName: `cred-secret-${dbPayload.dbName}`,
            endpointURL: dbPayload.endpoint,
            region: dbPayload.region,
            secretAccessKey: dbPayload.secretKey,
            verifyTLS: dbPayload.verifyTlS,
            forcePathStyle: dbPayload.forcePathStyle,
          },
        },
      },
    };
  }
  return dataSource;
};
