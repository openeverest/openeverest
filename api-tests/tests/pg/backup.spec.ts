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

import {expect, test} from '@playwright/test'
import * as th from '@tests/utils/api';
import {PG_BACKUP_DB_CLUSTER_NAME_ENV, PG_BACKUP_STORAGE_NAME_ENV} from "@tests/pg/consts";
import {TIMEOUTS} from "@root/constants";

const testPrefix = 'pg-backup';

test.describe.parallel('PG cluster backup tests', () => {
  test.describe.configure({timeout: TIMEOUTS.FiveMinutes});

  test('create/list/delete db cluster backup', async ({request}) => {
    const backupName = th.suffixedName(testPrefix),
      dbClusterName = process.env[PG_BACKUP_DB_CLUSTER_NAME_ENV],
      bsName = process.env[PG_BACKUP_STORAGE_NAME_ENV];

    await test.step('create DB cluster backup', async () => {
      const backup = await th.createDBClusterBackup(request, dbClusterName, backupName, bsName)
      expect(backup.metadata.name).toMatch(backupName)
      expect(backup.spec.dbClusterName).toMatch(dbClusterName)
      expect(backup.spec.backupStorageName).toMatch(bsName)
    })

    await test.step('Wait for DB cluster backup to finish', async () => {
      await th.waitForDBClusterBackupToFinish(request, backupName)
    })

    await test.step('list DB cluster backups', async () => {
      await expect(async () => {
        const list = await th.listDBClusterBackups(request, dbClusterName)
        expect(list.items.filter((i) => i.metadata.name == backupName).length).toBe(1)
      }).toPass({
        intervals: [TIMEOUTS.TenSeconds],
        timeout: TIMEOUTS.ThirtySeconds,
      })
    })

    await test.step('Delete DB cluster backup', async () => {
      await th.deleteDBClusterBackup(request, backupName)
    })
  })
});
