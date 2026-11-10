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

import { test as teardown } from '@playwright/test';
import { getCITokenFromLocalStorage } from '../utils/localStorage';
import { backupStorageCredentialsSecretName } from '../utils/backup-storage';
import { EVEREST_CI_CLUSTER, getBucketNamespacesMap } from '../constants';

teardown.describe.serial('Backup Storage teardown', () => {
  teardown('Delete Backup Storages', async ({ request }) => {
    const token = await getCITokenFromLocalStorage();
    const promises: Promise<unknown>[] = [];
    const bucketNamespacesMap = getBucketNamespacesMap();
    const headers = { Authorization: `Bearer ${token}` };

    bucketNamespacesMap.forEach(async ([bucket, namespace]) => {
      const namespaceUrl = `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}`;
      promises.push(
        (async () => {
          await request.delete(`${namespaceUrl}/backup-storages/${bucket}`, {
            headers,
          });
          await request.delete(
            `${namespaceUrl}/secrets/${backupStorageCredentialsSecretName(bucket)}`,
            { headers }
          );
        })()
      );
    });
    await Promise.all(promises);
  });
});
