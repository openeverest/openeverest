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
// Installed by api-tests/manifests/test-backup-class.yaml, writes to s3 with AccessKey.
const backupClassName = 'test-backup-class'

const condition = (bs, type: string) => (bs.status?.conditions ?? []).find((c) => c.type === type)
const names = (list) => list.items.map((s) => s.metadata.name)

test.describe.parallel('Backup Storage V2 tests', () => {
  test.describe.configure({timeout: 300 * 1000});

  const bsName = th.limitedSuffixedName(testPrefix + '-s3');

  test.afterAll(async ({request}) => {
    await th.deleteBackupStorage(request, bsName)
  })

  test('create/get/list/update/delete s3 backup storage', async ({request}) => {
    const payload = th.getBackupStoragePayload(bsName)

    try {
      let backupStorage

      await test.step('create backup storage', async () => {
        backupStorage = await th.generateBackupStorage(request, payload)
        expect(backupStorage.metadata.name).toBe(bsName)
        expect(backupStorage.spec.backupStorageClassRef.name).toBe('s3')
        expect(backupStorage.spec.location).toEqual(payload.spec.location)
        expect(backupStorage.spec.parameters).toEqual(payload.spec.parameters)
        expect(backupStorage.spec.authentication).toEqual(payload.spec.authentication)
        expect(backupStorage.spec.tls).toEqual(payload.spec.tls)
      });

      await test.step('credentials Secret is a plain Secret created through the secrets API', async () => {
        const secret = await th.getSecret(request, th.backupStorageCredentialsSecretName(bsName))
        expect(secret.metadata.labels['openeverest.io/managed']).toBe('true')
        expect(secret.metadata.labels['openeverest.io/definition']).toBeUndefined()
        expect(secret.metadata.labels['openeverest.io/provider']).toBeUndefined()
        expect(secret.data).toBeUndefined()
      });

      await test.step('get backup storage', async () => {
        backupStorage = await th.getBackupStorage(request, bsName)
        expect(backupStorage.metadata.name).toBe(bsName)
        expect(backupStorage.spec.backupStorageClassRef.name).toBe('s3')
        expect(backupStorage.spec.location).toEqual(payload.spec.location)
        expect(backupStorage.spec.parameters).toEqual(payload.spec.parameters)
        expect(backupStorage.spec.authentication.credentialsSecretRef.name).toBe(th.backupStorageCredentialsSecretName(bsName))
      });

      await test.step('backup storage is accepted and writable by the test backup class', async () => {
        await expect.poll(async () => {
          backupStorage = await th.getBackupStorage(request, bsName)
          return {
            accepted: condition(backupStorage, 'Accepted')?.status,
            compatible: (backupStorage.status?.backupClasses ?? []).find((c) => c.name === backupClassName)?.compatible,
          }
        }, {
          intervals: [1000],
          timeout: 30 * 1000,
        }).toEqual({accepted: 'True', compatible: true})
      });

      await test.step('list backup storages', async () => {
        const storages = await th.listBackupStorages(request)
        expect(storages.items).toBeDefined()
        expect(names(storages)).toContain(bsName)
      });

      await test.step('list backup storages by backup storage class', async () => {
        expect(names(await th.listBackupStorages(request, {backupStorageClass: 's3'}))).toContain(bsName)
        expect(names(await th.listBackupStorages(request, {backupStorageClass: 'does-not-exist'}))).not.toContain(bsName)
      });

      await test.step('list backup storages by backup class', async () => {
        expect(names(await th.listBackupStorages(request, {backupClass: backupClassName}))).toContain(bsName)
        expect(names(await th.listBackupStorages(request, {backupClass: 'does-not-exist'}))).not.toContain(bsName)
        expect(names(await th.listBackupStorages(request, {
          backupClass: backupClassName,
          backupStorageClass: 'does-not-exist',
        }))).not.toContain(bsName)
      });

      await test.step('update backup storage', async () => {
        await expect(async () => {
          // The controller writes status, so every attempt starts from the current resourceVersion.
          const current = await th.getBackupStorage(request, bsName)
          backupStorage = await th.updateBackupStorage(request, bsName, {
            ...current,
            spec: {...current.spec, parameters: {...current.spec.parameters, region: 'eu-west-1'}},
          })
          expect(backupStorage.spec.parameters.region).toBe('eu-west-1')
          expect(backupStorage.spec.parameters.endpointURL).toBe(payload.spec.parameters.endpointURL)
          expect(backupStorage.spec.location).toEqual(payload.spec.location)
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      });

      await test.step('patch backup storage', async () => {
        await expect(async () => {
          backupStorage = await th.patchBackupStorage(request, bsName, {spec: {parameters: {region: 'us-east-1'}}})
          expect(backupStorage.spec.parameters.region).toBe('us-east-1')
          // Members the patch did not name keep their stored value.
          expect(backupStorage.spec.parameters.endpointURL).toBe(payload.spec.parameters.endpointURL)
          expect(backupStorage.spec.parameters.forcePathStyle).toBe(true)
          expect(backupStorage.spec.authentication).toEqual(payload.spec.authentication)
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      });

      await test.step('location is immutable', async () => {
        for (const location of [{bucket: 'bucket-5'}, {prefix: 'elsewhere'}]) {
          const response = await th.patchBackupStorageRaw(request, bsName, {spec: {location: location}})
          expect(response.status(), `patch location ${JSON.stringify(location)}`).toBeGreaterThanOrEqual(400)
          expect(response.status(), `patch location ${JSON.stringify(location)}`).toBeLessThan(500)
          expect(await response.text()).toContain('location is immutable')
        }

        await expect(async () => {
          const current = await th.getBackupStorage(request, bsName)
          const response = await th.updateBackupStorageRaw(request, bsName, {
            ...current,
            spec: {...current.spec, location: {...current.spec.location, bucket: 'bucket-5'}},
          })
          expect(response.status()).toBeGreaterThanOrEqual(400)
          expect(response.status()).toBeLessThan(500)
          expect(await response.text()).toContain('location is immutable')
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      });

      await test.step('backup storage class is immutable', async () => {
        const response = await th.patchBackupStorageRaw(request, bsName, {spec: {backupStorageClassRef: {name: 'other'}}})
        expect(response.status()).toBeGreaterThanOrEqual(400)
        expect(response.status()).toBeLessThan(500)
        expect(await response.text()).toContain('backupStorageClassRef is immutable')
      });

      await test.step('create backup storage already exists', async () => {
        const resp = await th.createBackupStorageRaw(request, payload)
        expect(resp.status()).toBe(409)
      });

      await test.step('delete backup storage', async () => {
        await th.deleteBackupStorage(request, bsName)
      });
    } finally {
      await th.deleteBackupStorage(request, bsName)
    }
  })

  test('patch: merge patch semantics', async ({request}) => {
    const name = th.limitedSuffixedName(testPrefix + '-patch'),
      payload = th.getBackupStoragePayload(name),
      rotatedSecret = `${th.backupStorageCredentialsSecretName(name)}-2`

    try {
      await th.generateBackupStorage(request, payload)

      await test.step('null removes a member', async () => {
        const patched = await th.patchBackupStorage(request, name, {spec: {parameters: {forcePathStyle: null}}})
        expect(payload.spec.parameters.forcePathStyle).toBe(true)
        expect(patched.spec.parameters.forcePathStyle).toBeUndefined()
        expect(patched.spec.parameters.region).toBe(payload.spec.parameters.region)
      });

      await test.step('credentials rotate to another Secret without repeating the mode', async () => {
        await th.createBackupStorageCredentials(request, rotatedSecret)
        const patched = await th.patchBackupStorage(request, name, {
          spec: {authentication: {credentialsSecretRef: {name: rotatedSecret}}},
        })
        expect(patched.spec.authentication.mode).toBe('AccessKey')
        expect(patched.spec.authentication.credentialsSecretRef.name).toBe(rotatedSecret)
      });

      await test.step('the removed spec.s3 member is rejected rather than silently ignored', async () => {
        const response = await th.patchBackupStorageRaw(request, name, {spec: {s3: {bucket: 'bucket-7'}}})

        // Without fieldValidation=Strict the API server prunes the unknown member
        // and answers 200 having changed nothing.
        expect(response.status()).toBe(422)
        expect(await response.text()).toContain('spec.s3')
      });

      await test.step('a misspelt path is rejected rather than silently ignored', async () => {
        const response = await th.patchBackupStorageRaw(request, name, {
          spec: {authentication: {credentialSecretRef: {name: rotatedSecret}}},
        })

        expect(response.status()).toBe(422)
        expect(await response.text()).toContain('spec.authentication.credentialSecretRef')
      });

      await test.step('a member the caller may not set is rejected', async () => {
        for (const patch of [{status: {}}, {metadata: {finalizers: []}}, {metadata: {name: 'renamed'}}]) {
          const response = await th.patchBackupStorageRaw(request, name, patch)
          expect(response.status(), `patch ${JSON.stringify(patch)}`).toBe(400)
        }
      });
    } finally {
      await th.deleteBackupStorage(request, name)
      await th.deleteSecret(request, rotatedSecret)
    }
  })

  test('backup storage classes', async ({request}) => {
    await test.step('list backup storage classes', async () => {
      const classes = await th.listBackupStorageClasses(request)
      expect(names(classes)).toContain('s3')
    });

    await test.step('get the built-in s3 backup storage class', async () => {
      const s3 = await th.getBackupStorageClass(request, 's3')
      expect(s3.metadata.name).toBe('s3')
      const accessKey = s3.spec.authModes.find((m) => m.name === 'AccessKey')
      expect(accessKey).toBeDefined()
      expect(accessKey.credentials.required).toEqual(expect.arrayContaining(['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY']))
      expect(s3.spec.parametersSchema.openAPIV3Schema.required).toEqual(expect.arrayContaining(['region', 'endpointURL']))
    });

    await test.step('get: backup storage class not found', async () => {
      const response = await th.getBackupStorageClassRaw(request, 'does-not-exist')
      expect(response.status()).toBe(404)
    });
  })

  test('get: backup storage not found', async ({request}) => {
    const name = th.limitedSuffixedName('bsv2-non-existent'),
      response = await th.getBackupStorageRaw(request, name)

    expect(response.status()).toBe(404)
  })

  test('update: backup storage not found', async ({request}) => {
    const name = th.limitedSuffixedName('bsv2-non-existent'),
      payload = th.getBackupStoragePayload(name),
      response = await th.updateBackupStorageRaw(request, name, payload)

    expect(response.status()).toBe(404)
  })

  test('delete: backup storage not found', async ({request}) => {
    const name = th.limitedSuffixedName('bsv2-non-existent'),
      response = await th.deleteBackupStorageRaw(request, name)

    expect(response.status()).toBe(204)
  })
});
