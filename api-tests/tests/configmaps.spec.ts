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

const configMapName = 'test-configmap-name'

test.describe.parallel('ConfigMaps tests', () => {
  test.afterAll(async ({request}) => {
    await th.deleteConfigMap(request, configMapName)
  })

  test('create/get/list/delete configmap', async ({request}) => {
    await test.step('create configmap with data', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'ConfigMap',
        metadata: {
          name: configMapName,
          labels: {
            'openeverest.io/category': 'test-config',
            'openeverest.io/provider': 'test-provider',
          },
        },
        data: {
          'config.yaml': 'key: value\nanother: setting',
          'database.conf': 'host=localhost\nport=5432',
        },
      }

      const created = await th.createConfigMapWithData(request, data)
      expect(created.metadata.name).toBe(data.metadata.name)
      // Verify data is included in response for ConfigMap (unlike Secret)
      expect(created.data).toBeDefined()
      expect(created.data['config.yaml']).toBe('key: value\nanother: setting')
    })

    await test.step('get configmap - verify data is returned', async () => {
      const configMap = await th.getConfigMap(request, configMapName)
      expect(configMap.metadata.name).toBe(configMapName)
      expect(configMap.metadata.labels['openeverest.io/category']).toBe('test-config')
      // Verify data is returned (unlike secrets)
      expect(configMap.data).toBeDefined()
      expect(configMap.data['config.yaml']).toBe('key: value\nanother: setting')
      expect(configMap.data['database.conf']).toBe('host=localhost\nport=5432')
    })

    await test.step('list configmaps - verify data is returned', async () => {
      const list = await th.listConfigMaps(request)
      expect(list.items.length).toBeGreaterThan(0)
      
      // Find our created configmap
      const configMap = list.items.find(cm => cm.metadata.name === configMapName)
      
      expect(configMap).toBeDefined()
      expect(configMap.data).toBeDefined()
      expect(configMap.data['config.yaml']).toBe('key: value\nanother: setting')
    })

    await test.step('list configmaps with category filter', async () => {
      const list = await th.listConfigMapsWithCategory(request, 'test-config')
      
      const configMap1 = list.items.find(cm => cm.metadata.name === configMapName)
      expect(configMap1).toBeDefined()
      expect(configMap1.metadata.labels['openeverest.io/category']).toBe('test-config')
      
      const emptyList = await th.listConfigMapsWithCategory(request, 'non-existent-category')
      expect(emptyList.items.length).toBe(0)
    })

    await test.step('create configmap without category label should fail', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'ConfigMap',
        metadata: {
          name: th.limitedSuffixedName('no-category'),
        },
        data: {
          key: 'value',
        },
      }

      const response = await th.createConfigMapRaw(request, data)
      expect(response.ok()).toBeFalsy()
      const error = await response.json()
      expect(error.message).toContain('category')
    })

    await test.step('create configmap without data should fail', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'ConfigMap',
        metadata: {
          name: th.limitedSuffixedName('no-data'),
          labels: {
            'openeverest.io/category': 'test',
          },
        },
      }

      const response = await th.createConfigMapRaw(request, data)
      expect(response.ok()).toBeFalsy()
      const error = await response.json()
      expect(error.message).toContain('data')
    })

    await test.step('delete configmap', async () => {
      await th.deleteConfigMap(request, configMapName)
      
      // Verify configmap is deleted
      const response = await th.getConfigMapRaw(request, configMapName)
      expect(response.status()).toBe(404)
    })
  })
})
