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
      },
      EVEREST_CI_NAMESPACES.EVEREST_UI
    );
  });
});
