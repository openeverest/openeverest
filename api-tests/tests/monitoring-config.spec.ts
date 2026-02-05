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

import {expect, test} from '@fixtures'
import * as th from '@tests/utils/api';

const testPrefix = 'mc',
  mcNameKey = th.limitedSuffixedName(testPrefix + '-key'),
  mcNamePass = th.limitedSuffixedName(testPrefix + '-pass')

test.describe.parallel('Monitoring instances tests', () => {
  test.describe.configure({timeout: 60 * 1000});

  test.afterAll(async ({request}) => {
    await th.deleteMonitoringConfig(request, mcNameKey)
    await th.deleteMonitoringConfig(request, mcNamePass)
  })

  test('create/update/delete monitoring instance', async ({request}) => {
    await test.step('create monitoring instance with api key', async () => {
      const data = {
        type: 'pmm',
        name: mcNameKey,
        url: `https://${process.env.PMM1_IP}`,
        pmm: {
          apiKey: `${process.env.PMM1_API_KEY}`,
        },
        verifyTLS: false,
      }

      const created = await th.createMonitoringConfigWithData(request, data)
      expect(created.name).toBe(data.name)
      expect(created.url).toBe(data.url)
      expect(created.type).toBe(data.type)
    })

    await test.step('create monitoring instance with login and password', async () => {
      const data = {
        type: 'pmm',
        name: mcNamePass,
        url: `https://${process.env.PMM2_IP}`,
        pmm: {
          user: "admin",
          password: "admin",
        },
        verifyTLS: false,
      }

      const created = await th.createMonitoringConfigWithData(request, data)
      expect(created.name).toBe(data.name)
      expect(created.url).toBe(data.url)
      expect(created.type).toBe(data.type)
    })

    await test.step('get monitoring instance key', async () => {
      const mc = await th.getMonitoringConfig(request, mcNameKey)
      expect(mc.name).toBe(mcNameKey)
    })

    await test.step('get monitoring instance password', async () => {
      const mc = await th.getMonitoringConfig(request, mcNamePass)
      expect(mc.name).toBe(mcNamePass)
    })

    await test.step('update monitoring instance', async () => {
      const patchData = {url: 'http://monitoring-service.everest-monitoring.svc.cluster.local'} // URL pointing to the same instance

      await expect(async () => {
        const updated = await th.updateMonitoringConfig(request, mcNameKey, patchData)
        expect(updated.url).toMatch(patchData.url)
      }).toPass({
        intervals: [1000],
        timeout: 30 * 1000,
      })
    })

    await test.step('update monitoring instance to not existing', async () => {
      const patchData = {
        url: 'http://not-existing-url', // existing other monitoring URL
      }

      await expect(async () => {
        const updated = await th.updateMonitoringConfigRaw(request, mcNameKey, patchData)
        expect(updated.ok()).toBeFalsy()
        expect((await updated.json()).message).toContain("dial tcp: lookup not-existing-url")
      }).toPass({
        intervals: [1000],
        timeout: 30 * 1000,
      })
    })

    await test.step('update monitoring instance to existing with apiKey', async () => {
      const patchData = {
        url: `https://${process.env.PMM2_IP}`, // existing other monitoring URL
        pmm: {
          apiKey: `${process.env.PMM2_API_KEY}`,
        },
      }

      await expect(async () => {
        const updated = await th.updateMonitoringConfig(request, mcNameKey, patchData)
        expect(updated.url).toBe(patchData.url)
      }).toPass({
        intervals: [1000],
        timeout: 30 * 1000,
      })
    })

    await test.step('update monitoring instance to existing with admin password', async () => {
      const patchData = {
        url: 'https://monitoring-service.everest-monitoring.svc.cluster.local', // existing other monitoring URL
        pmm: {
          user: 'admin',
          password: 'admin'
        },
      }

      await expect(async () => {
        const updated = await th.updateMonitoringConfig(request, mcNameKey, patchData)
        expect(updated.url).toBe(patchData.url)
      }).toPass({
        intervals: [1000],
        timeout: 30 * 1000,
      })
    })

    await test.step('update monitoring instances failures', async () => {
      const testCases = [
        {
          payload: {url: 'not-url'},
          errorText: '\'url\' is an invalid URL',
        },
        {
          payload: {pmm: {apiKey: ''}},
          errorText: 'Error at "/pmm/apiKey"',
        },
        {
          payload: {type: 'pmm',},
          errorText: 'pmm key is required',
        },
      ]

      for (const testCase of testCases) {
        await expect(async () => {
          const response = await th.updateMonitoringConfigRaw(request, mcNameKey, testCase.payload)
          expect(response.status()).toBe(400)
          expect((await response.json()).message).toMatch(testCase.errorText)
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      }
    })

    await test.step('delete monitoring instance', async () => {
      await th.deleteMonitoringConfig(request, mcNameKey)
      await th.deleteMonitoringConfig(request, mcNamePass)
    })
  })

  test('create monitoring instance missing pmm', async ({request}) => {
    const data = {
        type: 'pmm',
        name: th.limitedSuffixedName(testPrefix + '-fail'),
        url: 'http://monitoring-instance',
      },
      response = await th.createMonitoringConfigWithDataRaw(request, data)

    expect(response.status()).toBe(400)
  })

  test('create monitoring instance missing pmm credentials', async ({request}) => {
    const data = {
        type: 'pmm',
        name: th.limitedSuffixedName(testPrefix + '-fail'),
        url: 'http://monitoring-instance',
        pmm: {},
      },
      response = await th.createMonitoringConfigWithDataRaw(request, data)

    expect(response.status()).toBe(400)
  })

  test('create monitoring instance failures', async ({request}) => {
    const data = {},
      response = await th.createMonitoringConfigWithDataRaw(request, data)
    expect(response.status()).toBe(400)
  })

  test('update: monitoring instance not found', async ({request}) => {
    const name = th.limitedSuffixedName(testPrefix + '-non-existent'),
      response = await th.updateMonitoringConfigRaw(request, name, {url: `http://${process.env.PMM_IP}`})
    expect(response.status()).toBe(404)
  })

  test('delete: monitoring instance not found', async ({request}) => {
    const name = th.limitedSuffixedName(testPrefix + '-non-existent'),
      response = await th.deleteMonitoringConfigRaw(request, name)

    expect(response.status()).toBe(404)
  })

  test('get: monitoring instance not found', async ({request}) => {
    const name = th.limitedSuffixedName(testPrefix + '-non-existent'),
      response = await th.getMonitoringConfigRaw(request, name)

    expect(response.status()).toBe(404)
  })
});