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

import {test as setup} from '@playwright/test';
import * as th from "@tests/utils/api";
import {PG_BACKUP_DB_CLUSTER_NAME_ENV} from "@tests/pg/consts";
import {getPGClusterDataSimple} from "@tests/utils/api";
import {TIMEOUTS} from "@root/constants";

const dbClusterName = th.limitedSuffixedName('pg-db-bak-');
process.env[PG_BACKUP_DB_CLUSTER_NAME_ENV] = dbClusterName;

setup.describe.serial('PG cluster for backup setup', () => {
  setup.describe.configure({timeout: TIMEOUTS.FiveMinutes});

  setup('Create PG cluster', async ({request}) => {

    await setup.step('Create PG cluster step', async () => {
      const data = getPGClusterDataSimple(dbClusterName);
      await th.createDBClusterWithData(request, data);
    });

    await setup.step('Wait for PG cluster to be ready step', async () => {
      await th.waitForDBClusterToBeReady(request, dbClusterName);
    });
  });
});