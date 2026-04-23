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

import {expect, test} from '@fixtures'
import * as th from '@tests/utils/api';

const testPrefix = 'bsv2'

test.describe.parallel('Backup Storage V2 tests', () => {
  test.describe.configure({timeout: 300 * 1000});

  const bsName = th.limitedSuffixedName(testPrefix + '-s3');

  test.afterAll(async ({request}) => {
    await th.deleteBackupStorageV2(request, bsName)
  })

  test('create/get/list/update/delete s3 backup storage', async ({request}) => {
    const payload = th.getBackupStorageV2Payload(bsName)

    try {
      let backupStorage

      await test.step('create backup storage', async () => {
        backupStorage = await th.createBackupStorageV2(request, payload)
        expect(backupStorage.metadata.name).toBe(bsName)
        expect(backupStorage.spec.type).toBe('s3')
        expect(backupStorage.spec.s3.bucket).toBe(payload.spec.s3.bucket)
        expect(backupStorage.spec.s3.region).toBe(payload.spec.s3.region)
        expect(backupStorage.spec.s3.endpointURL).toBe(payload.spec.s3.endpointURL)
      });

      await test.step('get backup storage', async () => {
        backupStorage = await th.getBackupStorageV2(request, bsName)
        expect(backupStorage.metadata.name).toBe(bsName)
        expect(backupStorage.spec.type).toBe('s3')
        expect(backupStorage.spec.s3.bucket).toBe(payload.spec.s3.bucket)
      });

      await test.step('list backup storages', async () => {
        const storages = await th.listBackupStoragesV2(request)
        expect(storages.items).toBeDefined()
        expect(storages.items.some((s) => s.metadata.name === bsName)).toBeTruthy()
      });

      await test.step('update backup storage', async () => {
        const updatePayload = {
          ...backupStorage,
          spec: {
            ...backupStorage.spec,
            s3: {
              ...backupStorage.spec.s3,
              bucket: `${payload.spec.s3.bucket}-upd`,
              accessKeyId: 'otherAccessKey',
              secretAccessKey: 'otherSecret',
            },
          },
        }

        await expect(async () => {
          backupStorage = await th.updateBackupStorageV2(request, bsName, updatePayload)
          expect(backupStorage.spec.s3.bucket).toBe(updatePayload.spec.s3.bucket)
          expect(backupStorage.spec.s3.region).toBe(payload.spec.s3.region)
          expect(backupStorage.spec.type).toBe('s3')
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      });

      await test.step('create backup storage already exists', async () => {
        const resp = await th.createBackupStorageRawV2(request, payload)
        expect(resp.status()).toBe(409)
      });

      await test.step('delete backup storage', async () => {
        await th.deleteBackupStorageV2(request, bsName)
      });
    } finally {
      await th.deleteBackupStorageV2(request, bsName)
    }
  })

  test('get: backup storage not found', async ({request}) => {
    const name = th.limitedSuffixedName('bsv2-non-existent'),
      response = await th.getBackupStorageRawV2(request, name)

    expect(response.status()).toBe(404)
  })

  test('update: backup storage not found', async ({request}) => {
    const name = th.limitedSuffixedName('bsv2-non-existent'),
      payload = th.getBackupStorageV2Payload(name),
      response = await th.updateBackupStorageRawV2(request, name, payload)

    expect(response.status()).toBe(404)
  })

  test('delete: backup storage not found', async ({request}) => {
    const name = th.limitedSuffixedName('bsv2-non-existent'),
      response = await th.deleteBackupStorageRawV2(request, name)

    expect(response.status()).toBe(204)
  })
});
