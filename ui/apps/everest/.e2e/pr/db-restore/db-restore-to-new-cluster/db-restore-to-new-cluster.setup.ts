import { test as setup } from '@playwright/test';
import { createDbClusterFn, deleteDbClusterFn } from '@e2e/utils/db-cluster';
import { EVEREST_CI_NAMESPACES, getBucketNamespacesMap } from '@e2e/constants';
import { dbClusterName } from './project.config';

setup.describe.serial('DB Restore To New Cluster setup', () => {
  setup(`Create ${dbClusterName} cluster`, async ({ request }) => {
    // Try to delete the cluster first in case it exists from a previous run
    try {
      await deleteDbClusterFn(
        request,
        dbClusterName,
        EVEREST_CI_NAMESPACES.EVEREST_UI
      );
    } catch (error) {
      // Ignore error if cluster doesn't exist
    }

    await createDbClusterFn(
      request,
      {
        dbName: dbClusterName,
        dbType: 'mongodb',
        dbVersion: '8.0.4-1',
        numberOfNodes: '1',
        numberOfProxies: '3',
        disk: '3',
        memory: '2',
        cpu: '1',
        storageClass: 'my-storage-class',
        sharding: true,
        shards: 2,
        configServerReplicas: 3,
        backup: {
          enabled: true,
          schedules: [
            {
              backupStorageName: getBucketNamespacesMap()[0][0],
              enabled: true,
              name: 'backup-1',
              schedule: '0 * * * *',
            },
          ],
          pitr: {
            enabled: true,
            backupStorageName: 'minio',
          },
        },
        externalAccess: true,
        sourceRanges: [
          {
            sourceRange: '192.168.1.1/32',
          },
        ],
        monitoringConfigName: 'pmm',
      },
      EVEREST_CI_NAMESPACES.EVEREST_UI
    );
  });
});
