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

import {expect, test, test as setup} from '@playwright/test';
import * as th from "@tests/utils/api";
import {
  PG_BACKUP_STORAGE_NAME_ENV, PG_RESTORE_DB_BACKUP_NAME_ENV,
  PG_RESTORE_DB_CLUSTER_NAME_ENV
} from "@tests/pg/consts";
import {getPGClusterDataSimple} from "@tests/utils/api";
import {TIMEOUTS} from "@root/constants";

const testPrefix = 'pg-db-res',
  dbClusterName = th.limitedSuffixedName(testPrefix),
  backupName = th.suffixedName(testPrefix),
  bsName = process.env[PG_BACKUP_STORAGE_NAME_ENV];

process.env[PG_RESTORE_DB_CLUSTER_NAME_ENV] = dbClusterName;
process.env[PG_RESTORE_DB_BACKUP_NAME_ENV] = backupName;

setup.describe.serial('PG cluster for restore setup', () => {
  setup.describe.configure({timeout: TIMEOUTS.FifteenMinutes});

  setup('Create PG cluster', async ({request}) => {

    await setup.step('Create PG cluster step', async () => {
      const data = getPGClusterDataSimple(dbClusterName);
      await th.createDBClusterWithData(request, data);
    });

    await setup.step('Wait for PG cluster to be ready step', async () => {
      await th.waitForDBClusterToBeReady(request, dbClusterName);
    });

    await test.step('create DB cluster backup', async () => {
      const backup = await th.createDBClusterBackup(request, dbClusterName, backupName, bsName)
      expect(backup.metadata.name).toMatch(backupName)
      expect(backup.spec.dbClusterName).toMatch(dbClusterName)
      expect(backup.spec.backupStorageName).toMatch(bsName)
    })

    await test.step('Wait for DB cluster backup to finish', async () => {
      await th.waitForDBClusterBackupToFinish(request, backupName)
    })

  });
});