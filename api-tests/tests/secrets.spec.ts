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

const secretName1 = 'test-secret-name-1'
const secretName2 = 'test-secret-name-2'
const secretName3 = 'test-secret-name-3'

test.describe.parallel('Secrets tests', () => {
  test.afterAll(async ({request}) => {
    await th.deleteSecret(request, secretName1)
    await th.deleteSecret(request, secretName2)
    await th.deleteSecret(request, secretName3)
  })

  test('create/get/list/delete secret', async ({request}) => {
    await test.step('create secret with data', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'Secret',
        metadata: {
          name: secretName1,
          labels: {
            'openeverest.io/definition': 'test-credentials',
            'openeverest.io/provider': 'test-provider',
          },
        },
        type: 'Opaque',
        data: {
          accessKey: Buffer.from('myaccesskey').toString('base64'),
          secretKey: Buffer.from('mysecretkey').toString('base64'),
        },
      }

      const created = await th.createSecretWithData(request, data)
      expect(created.metadata.name).toBe(data.metadata.name)
      expect(created.type).toBe('Opaque')
      // Verify no data is returned on create
      expect(created.data).toBeUndefined()
      expect(created.stringData).toBeUndefined()
    })

    await test.step('get secret', async () => {
      const secret = await th.getSecret(request, secretName1)
      expect(secret.metadata.name).toBe(secretName1)
      expect(secret.metadata.labels['openeverest.io/managed']).toBe('true')
      expect(secret.metadata.labels['openeverest.io/definition']).toBe('test-credentials')
      expect(secret.metadata.labels['openeverest.io/provider']).toBe('test-provider')
      // Verify no sensitive data is returned
      expect(secret.data).toBeUndefined()
      expect(secret.stringData).toBeUndefined()
    })

    await test.step('list secrets', async () => {
      const list = await th.listSecrets(request)
      expect(list.items.length).toBeGreaterThan(0)
      
      // Find our created secrets
      const secret = list.items.find(s => s.metadata.name === secretName1)
      
      expect(secret).toBeDefined()
      expect(secret.data).toBeUndefined()
      expect(secret.stringData).toBeUndefined()
    })

    await test.step('list secrets with filters', async () => {
      const list = await th.listSecretsWithFilters(request, 'test-provider', 'test-credentials')
      
      const secret1 = list.items.find(s => s.metadata.name === secretName1)
      expect(secret1).toBeDefined()
      expect(secret1.metadata.labels['openeverest.io/definition']).toBe('test-credentials')
      
      const emptyList = await th.listSecretsWithFilters(request, 'test-provider', 'non-existent')
      expect(emptyList.items.length).toBe(0)
    })

    await test.step('create secret without definition label should fail', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'Secret',
        metadata: {
          name: secretName2,
          labels: {
            'openeverest.io/provider': 'test-provider',
          },
        },
        type: 'Opaque',
        data: {
          key: Buffer.from('value').toString('base64'),
        },
      }

      const response = await th.createSecretRaw(request, data)
      expect(response.status()).toBe(400)
      const error = await response.json()
      expect(error.message).toContain('definition')
    })

    await test.step('create secret without provider label should fail', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'Secret',
        metadata: {
          name: secretName3,
          labels: {
            'openeverest.io/definition': 'shared-secret',
          },
        },
        type: 'Opaque',
        stringData: {
          username: 'plainuser',
          password: 'plainpassword',
        },
      }

      const response = await th.createSecretRaw(request, data)
      expect(response.status()).toBe(400)
      const error = await response.json()
      expect(error.message).toContain('provider')
    })

    await test.step('delete secret', async () => {
      await th.deleteSecret(request, secretName1)
      
      // Verify secret is deleted
      const response = await th.getSecretRaw(request, secretName1)
      expect(response.status()).toBe(404)
    })
  })
})
