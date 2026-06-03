// everest
// Copyright (C) 2025 Percona LLC
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

import {expect, test} from '@fixtures'
import * as th from '@tests/utils/api'

const testPrefix = 'ds-val'

// These tests exercise the admission-time DataSource S3 validation introduced
// in https://github.com/openeverest/openeverest/issues/2229. The validator
// resolves credentials before invoking the (debug-skipped) S3 probe, so the
// resolution failures here surface even when running against a debug build.
test.describe.parallel('DataSource S3 admission validation', () => {
  test.describe.configure({timeout: 60 * 1000})

  test('reject: DataImport references non-existent CredentialsSecret', async ({request}) => {
    const dbName = th.limitedSuffixedName(testPrefix + '-di-sec')
    const payload = th.getPGClusterDataSimple(dbName)

    // Inline accessKeyId / secretAccessKey omitted on purpose: the validator
    // must fall back to the referenced Secret, which does not exist.
    payload.spec.dataSource = {
      dataImport: {
        dataImporterName: 'test-data-importer',
        source: {
          path: 'test.sql',
          s3: {
            bucket: 'irrelevant-bucket',
            region: 'us-east-1',
            endpointURL: 'https://s3.us-east-1.amazonaws.com',
            credentialsSecretName: th.limitedSuffixedName('missing-sec'),
          },
        },
      },
    }

    const resp = await th.createDBClusterWithDataRaw(request, payload)
    expect(resp.status()).toBeGreaterThanOrEqual(400)
    expect(resp.status()).toBeLessThan(500)
    const body = await resp.json()
    expect(body.message ?? '').toMatch(/Secret|credentials/i)
  })

  test('reject: BackupSource references non-existent BackupStorage', async ({request}) => {
    const dbName = th.limitedSuffixedName(testPrefix + '-bs-miss')
    const payload = th.getPGClusterDataSimple(dbName)

    payload.spec.dataSource = {
      backupSource: {
        backupStorageName: th.limitedSuffixedName('missing-bs'),
        path: 'snapshots/2025/01',
      },
    }

    const resp = await th.createDBClusterWithDataRaw(request, payload)
    expect(resp.status()).toBeGreaterThanOrEqual(400)
    expect(resp.status()).toBeLessThan(500)
    const body = await resp.json()
    expect(body.message ?? '').toMatch(/BackupStorage/i)
  })

  test('accept: DataImport with inline credentials skips Secret lookup', async ({request}) => {
    const dbName = th.limitedSuffixedName(testPrefix + '-di-inline')
    const payload = th.getPGClusterDataSimple(dbName)

    // Inline creds present → validator does not look up the referenced Secret.
    // The S3 probe itself is skipped on debug builds, so this exercises the
    // happy admission path through credential resolution.
    payload.spec.dataSource = {
      dataImport: {
        dataImporterName: 'test-data-importer',
        source: {
          path: 'test.sql',
          s3: {
            bucket: 'irrelevant-bucket',
            region: 'us-east-1',
            endpointURL: 'https://s3.us-east-1.amazonaws.com',
            credentialsSecretName: 'unused-secret-ref',
            accessKeyId: 'AKIAEXAMPLE000000001',
            secretAccessKey: 'examplesecretkeyvalue0000000000000000000',
          },
        },
      },
    }

    try {
      const resp = await th.createDBClusterWithDataRaw(request, payload)
      expect(resp.ok()).toBeTruthy()
    } finally {
      await th.deleteDBClusterRaw(request, dbName)
    }
  })
})
