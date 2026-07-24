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

const configMapName1 = 'test-config-map-1'
const configMapName2 = 'test-config-map-2'
const configMapName3 = 'test-config-map-3'

test.describe.parallel('ConfigMaps tests', () => {
  test.afterAll(async ({request}) => {
    await th.deleteConfigMap(request, configMapName1)
    await th.deleteConfigMap(request, configMapName2)
    await th.deleteConfigMap(request, configMapName3)
  })

  test('create/get/list/delete config-map', async ({request}) => {
    await test.step('create config-map with data', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'ConfigMap',
        metadata: {
          name: configMapName1,
          labels: {
            'openeverest.io/definition': 'test-config',
            'openeverest.io/provider': 'test-provider',
          },
        },
        data: {
          endpoint: 'https://example.com',
        },
      }

      const created = await th.createConfigMapWithData(request, data)
      expect(created.metadata.name).toBe(data.metadata.name)
      expect(created.data).toBeDefined()
      expect(created.data['endpoint']).toBe('https://example.com')
    })

    await test.step('get config-map - verify data is returned', async () => {
      const configMap = await th.getConfigMap(request, configMapName1)
      expect(configMap.metadata.name).toBe(configMapName1)
      expect(configMap.metadata.labels['openeverest.io/definition']).toBe('test-config')
      expect(configMap.metadata.labels['openeverest.io/provider']).toBe('test-provider')
      expect(configMap.metadata.labels['openeverest.io/managed']).toBe('true')
      expect(configMap.data).toBeDefined()
      expect(configMap.data['endpoint']).toBe('https://example.com')
    })

    await test.step('list config-maps - verify data is returned', async () => {
      const list = await th.listConfigMaps(request)
      expect(list.items.length).toBeGreaterThan(0)
      
      // Find our created config-map
      const configMap = list.items.find(cm => cm.metadata.name === configMapName1)
      
      expect(configMap).toBeDefined()
      expect(configMap.data).toBeDefined()
      expect(configMap.data['endpoint']).toBe('https://example.com')
    })

    await test.step('list config-maps with definition filter', async () => {
      const list = await th.listConfigMapsWithDefinition(request, 'test-config')
      
      const found = list.items.find(cm => cm.metadata.name === configMapName1)
      expect(found).toBeDefined()
      expect(found.metadata.labels['openeverest.io/definition']).toBe('test-config')
      
      const emptyList = await th.listConfigMapsWithDefinition(request, 'non-existent-definition')
      expect(emptyList.items.length).toBe(0)
    })

    await test.step('create config-map without definition label should fail', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'ConfigMap',
        metadata: {
          name: configMapName2,
          labels: {
            'openeverest.io/provider': 'test-provider',
          },
        },
        data: {
          key: 'value',
        },
      }

      const response = await th.createConfigMapRaw(request, data)
      expect(response.ok()).toBeFalsy()
      const error = await response.json()
      expect(response.status()).toBe(400)
      expect(error.message).toContain('definition')
    })

    await test.step('create config-map without provider label should fail', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'ConfigMap',
        metadata: {
          name: configMapName3,
          labels: {
            'openeverest.io/definition': 'test-config',
          },
        },
        data: {
          key: 'value',
        },
      }

      const response = await th.createConfigMapRaw(request, data)
      expect(response.ok()).toBeFalsy()
      expect(response.status()).toBe(400)
      const error = await response.json()
      expect(error.message).toContain('provider')
    })

    await test.step('create config-map with invalid schema should fail', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'ConfigMap',
        metadata: {
          name: configMapName3,
          labels: {
            'openeverest.io/definition': 'test-config',
            'openeverest.io/provider': 'test-provider',
          },
        },
        data: {
          // Missing required 'endpoint' field
          'other-key': 'value',
        },
      }

      const response = await th.createConfigMapRaw(request, data)
      expect(response.ok()).toBeFalsy()
      expect(response.status()).toBe(400)
      const error = await response.json()
      expect(error.message).toContain('schema')
    })

    await test.step('delete config-map', async () => {
      await th.deleteConfigMap(request, configMapName1)
      
      // Verify configmap is deleted
      const response = await th.getConfigMapRaw(request, configMapName1)
      expect(response.status()).toBe(404)
    })
  })
})
