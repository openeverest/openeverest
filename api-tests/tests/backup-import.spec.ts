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
import {TIMEOUTS} from '@root/constants';
import * as th from '@tests/utils/api';

const testPrefix = 'bi'

test.describe.parallel('Backup Import tests', () => {
  test.describe.configure({timeout: TIMEOUTS.OneMinute});

  test('create/get/list/delete backup import', async ({request}) => {
    const bsName = th.limitedSuffixedName(testPrefix + '-s3')
    const biName = th.limitedSuffixedName(testPrefix + '-imp')

    try {
      await th.generateBackupStorage(request, th.getBackupStoragePayload(bsName))

      const payload = th.getBackupImportPayload(biName, bsName)

      let backupImport

      await test.step('create backup import', async () => {
        backupImport = await th.createBackupImport(request, payload)
        expect(backupImport.metadata.name).toBe(biName)
        expect(backupImport.spec.classRef.name).toBe(payload.spec.classRef.name)
        expect(backupImport.spec.storageRef.name).toBe(bsName)
      });

      await test.step('get backup import', async () => {
        backupImport = await th.getBackupImport(request, biName)
        expect(backupImport.metadata.name).toBe(biName)
        expect(backupImport.spec.classRef.name).toBe(payload.spec.classRef.name)
        expect(backupImport.spec.storageRef.name).toBe(bsName)
      });

      await test.step('list backup imports', async () => {
        const imports = await th.listBackupImports(request)
        expect(imports.items).toBeDefined()
        expect(imports.items.some((i) => i.metadata.name === biName)).toBeTruthy()
      });

      await test.step('create backup import already exists', async () => {
        const resp = await th.createBackupImportRaw(request, payload)
        expect(resp.status()).toBe(409)
      });

      await test.step('delete backup import', async () => {
        await th.deleteBackupImport(request, biName)
      });
    } finally {
      await th.deleteBackupImportRaw(request, biName)
      await th.deleteBackupStorage(request, bsName)
    }
  })

  test('create: rejected when storage does not exist', async ({request}) => {
    const biName = th.limitedSuffixedName(testPrefix + '-nostor')
    const payload = th.getBackupImportPayload(biName, 'non-existent-storage')

    const response = await th.createBackupImportRaw(request, payload)
    expect(response.status()).toBe(400)
    expect(await response.text()).toContain('backup storage')
  })

  test('create: rejected when backup class does not exist', async ({request}) => {
    const bsName = th.limitedSuffixedName(testPrefix + '-noclass-s3')
    const biName = th.limitedSuffixedName(testPrefix + '-noclass')

    try {
      await th.generateBackupStorage(request, th.getBackupStoragePayload(bsName))

      const payload = th.getBackupImportPayload(biName, bsName, 'non-existent-class')
      const response = await th.createBackupImportRaw(request, payload)
      expect(response.status()).toBe(400)
      expect(await response.text()).toContain('backup class')
    } finally {
      await th.deleteBackupStorage(request, bsName)
    }
  })

  test('create: warns while the backup class is not claimed', async ({request}) => {
    // No provider runtime claims test-backup-class here, so its verdict stays Pending.
    const bsName = th.limitedSuffixedName(testPrefix + '-pend-s3'),
      biName = th.limitedSuffixedName(testPrefix + '-pend')

    try {
      await th.generateBackupStorage(request, th.getBackupStoragePayload(bsName))

      const response = await th.createBackupImportRaw(request, th.getBackupImportPayload(biName, bsName)),
        warnings: string[] = (response.headersArray() as {name: string, value: string}[])
          .filter((h) => h.name.toLowerCase() === 'warning')
          .map((h) => h.value)

      await th.checkError(response)
      expect(warnings.some((w) => w.includes('test-backup-class') && w.includes('is Pending for Provider'))).toBeTruthy()
    } finally {
      await th.deleteBackupImportRaw(request, biName)
      await th.deleteBackupStorage(request, bsName)
    }
  })

  test('create: rejected when the backup class is incompatible', async ({request, cli}) => {
    const className = th.limitedSuffixedName(testPrefix + '-drift'),
      bsName = th.limitedSuffixedName(testPrefix + '-drift-s3'),
      biName = th.limitedSuffixedName(testPrefix + '-drift-imp'),
      backupClass = {
        apiVersion: 'backup.openeverest.io/v1alpha1',
        kind: 'BackupClass',
        metadata: {name: className},
        spec: {
          executionMode: 'ProviderManaged',
          providerManaged: {providerRef: {name: 'test-provider'}},
          supportsImport: true,
          backupStorageClasses: [{name: 's3', authModes: ['AccessKey']}],
        },
      },
      // What the class's provider reports when the class differs from the one it ships.
      specDrift = {
        status: {
          conditions: [{
            type: 'Accepted',
            status: 'False',
            reason: 'SpecDrift',
            message: 'edited by the test',
            observedGeneration: 1,
            lastTransitionTime: '2026-01-01T00:00:00Z',
          }],
        },
      }

    try {
      await (await cli.execSilent(`echo '${JSON.stringify(backupClass)}' | kubectl apply -f -`)).assertSuccess()
      await (await cli.execSilent(
        `kubectl patch backupclass ${className} --subresource=status --type=merge -p '${JSON.stringify(specDrift)}'`,
      )).assertSuccess()
      await (await cli.execSilent(
        `kubectl wait backupclass/${className} --for=jsonpath='{.status.providers[0].compatibility}'=Incompatible --timeout=60s`,
      )).assertSuccess()
      await th.generateBackupStorage(request, th.getBackupStoragePayload(bsName))

      const response = await th.createBackupImportRaw(request, th.getBackupImportPayload(biName, bsName, className))

      expect(response.status()).toBe(400)
      expect(await response.text()).toContain('(SpecDrift)')
    } finally {
      await th.deleteBackupStorage(request, bsName)
      await cli.execSilent(`kubectl delete backupclass ${className} --ignore-not-found`)
    }
  })

  test('get: backup import not found', async ({request}) => {
    const response = await th.getBackupImportRaw(request, th.limitedSuffixedName(testPrefix + '-404'))
    expect(response.status()).toBe(404)
  })

  test('delete: backup import not found', async ({request}) => {
    const response = await th.deleteBackupImportRaw(request, th.limitedSuffixedName(testPrefix + '-404'))
    expect(response.status()).toBe(204)
  })
});
