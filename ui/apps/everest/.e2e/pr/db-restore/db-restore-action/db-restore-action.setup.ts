import { test as setup } from '@playwright/test';
import { createDbClusterFn } from '@e2e/utils/db-cluster';
import { EVEREST_CI_NAMESPACES, getBucketNamespacesMap } from '@e2e/constants';
import { dbClusterName } from './project.config';

setup.describe.serial('DB Cluster Restore setup', () => {
  setup(`Create ${dbClusterName} cluster`, async ({ request }) => {
    await createDbClusterFn(
      request,
      {
        dbName: dbClusterName,
        dbType: 'mysql',
        numberOfNodes: '1',
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
        },
      },
      EVEREST_CI_NAMESPACES.EVEREST_UI
    );
  });
});
