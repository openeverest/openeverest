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
import * as th from '@tests/utils/api'
import {
  PG_RESTORE_DB_BACKUP_NAME_ENV,
  PG_RESTORE_DB_CLUSTER_NAME_ENV
} from "@tests/pg/consts";
import {TIMEOUTS} from "@root/constants";

const testPrefix = 'pg-res';

test.describe('PG restore tests', () => {
  test.describe.configure({timeout: TIMEOUTS.FiveMinutes});

  test('create/list/delete db cluster restore', async ({request}) => {
    const dbClusterName = process.env[PG_RESTORE_DB_CLUSTER_NAME_ENV],
      backupName = process.env[PG_RESTORE_DB_BACKUP_NAME_ENV],
      restoreName = th.suffixedName(testPrefix + '-db-restore');

    try {
      await test.step('create db clusters restore', async () => {
        await th.createDBClusterRestore(request, restoreName, dbClusterName, backupName)
      });

      await test.step('list db cluster restore', async () => {
        await expect(async () => {
          const list = await th.listDBClusterRestores(request, dbClusterName)
          expect(list.items.filter((i) => i.metadata.name == restoreName).length).toBe(1)
        }).toPass({
          intervals: [TIMEOUTS.TenSeconds],
          timeout: TIMEOUTS.ThirtySeconds,
        })
      })

      await test.step('wait for DB cluster restore to finish', async () => {
        await th.waitForDBClusterRestoreToFinish(request, restoreName)
      })

      await test.step('delete db clusters restore', async () => {
        await th.deleteDBClusterRestore(request, restoreName)
      });
    } finally {
      await th.deleteDBClusterRestore(request, restoreName)
    }

  })

})