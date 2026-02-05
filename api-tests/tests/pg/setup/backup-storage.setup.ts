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
import {PG_BACKUP_BUCKET_NAME, PG_BACKUP_STORAGE_NAME_ENV} from "@tests/pg/consts";
import {EVEREST_CI_NAMESPACE, TIMEOUTS} from "@root/constants";
import * as th from "@tests/utils/api";

const backupStorageName = th.limitedSuffixedName('pg-bak-str');
process.env[PG_BACKUP_STORAGE_NAME_ENV] = backupStorageName;

setup.describe.serial('PG Backup Storage setup', () => {
  setup.describe.configure({timeout: TIMEOUTS.ThirtySeconds});

  setup('Create Backup Storage for PG DB cluster', async ({request}) => {
    const payload = {
      type: 's3',
      name: backupStorageName,
      url: 'https://minio.minio.svc',
      description: 'PG backup storage',
      bucketName: PG_BACKUP_BUCKET_NAME,
      region: 'us-east-1',
      accessKey: 'minioadmin',
      secretKey: 'minioadmin',
      allowedNamespaces: [EVEREST_CI_NAMESPACE],
      verifyTLS: false,
      forcePathStyle: true,
    }
    await th.createBackupStorageWithData(request, payload)
  });
});