import { DbType } from '@percona/types';
import { dbTypeToProxyType } from '@percona/utils';
import { CUSTOM_NR_UNITS_INPUT_VALUE } from 'components/cluster-form';
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
    numberOfProxies === CUSTOM_NR_UNITS_INPUT_VALUE
      ? customNrOfProxies
      : numberOfProxies,
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
  if (dbPayload.dataImporter) {
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
