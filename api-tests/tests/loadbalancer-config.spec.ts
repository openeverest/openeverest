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

const testPrefix = 'lbc'

test.describe.parallel('LoadBalancer config tests', () => {
  const lbcName_1 = th.limitedSuffixedName(testPrefix + '-first')
  const lbcName_2 = th.limitedSuffixedName(testPrefix + '-second')

  test.afterAll(async ({request}) => {
    await th.deleteLoadBalancerConfig(request, lbcName_1)
    await th.deleteLoadBalancerConfig(request, lbcName_2)
  });

  test('create/update/delete loadbalancerconfig instance', async ({request, page}) => {

    await test.step('create lbc', async () => {
      const data = {
        metadata: {
          name: lbcName_1
        },
        spec: {
          annotations: {
            "prefix.com/valid-annotation": "some-value",
            "prefix.com/another-valid-annotation123": "another-value",
          }
        },
      }
      const response = await th.createLoadBalancerConfigWithData(request, data)
      expect(response.metadata.name).toBe(data.metadata.name)
    })

    await test.step('create another lbc', async () => {
      const data = {
        metadata: {
          name: lbcName_2
        },
        spec: {
          annotations: {
            "no-prefix-annotation": "some-value1",
            "another-no-prefix-annotation123": "another-value2",
          }
        },
      }
      const response = await th.createLoadBalancerConfigWithData(request, data)

      expect(response.metadata.name).toBe(data.metadata.name)
    })

    await test.step('list lbcs', async () => {
      const list = await th.listLoadBalancerConfigs(request)
      expect(list.items.filter((i) => i.metadata.name == lbcName_1).length).toBe(1)
      expect(list.items.filter((i) => i.metadata.name == lbcName_2).length).toBe(1)
    })

    await test.step('get lbc', async () => {
      const resp = await th.getLoadBalancerConfig(request, lbcName_1)
      expect(resp.metadata.name).toBe(lbcName_1)
    })

    await test.step('update lbc', async () => {
      let lbc = await th.getLoadBalancerConfig(request, lbcName_1)
      lbc.spec.annotations = {"new-key": "newValue"}
      const updated = await th.updateLoadBalancerConfigWithData(request, lbcName_1, lbc);
      expect(updated.metadata.name).toBe(lbc.metadata.name)
      expect(updated.spec.annotations).toMatchObject(lbc.spec.annotations)
    })

    await test.step('update lbc failure', async () => {
      let lbc = await th.getLoadBalancerConfig(request, lbcName_1)
      lbc.spec.annotations = {"-invalidkey": "newValue"}
      let updated = await th.updateLoadBalancerConfigWithDataRaw(request, lbcName_1, lbc);
      expect(updated.ok()).toBeFalsy()
      expect((await updated.json()).message).toContain("invalid annotation key")
    })

    await test.step('delete lbc', async () => {
      let list = await th.listLoadBalancerConfigs(request)
      expect(list.items.filter((i) => i.metadata.name == lbcName_1).length).toBe(1)

      // remove lbc_1
      await th.deleteLoadBalancerConfig(request, lbcName_1)

      // list of the lbc is not updated immediately, there is some kind of cache
      await page.waitForTimeout(1000)

      list = await th.listLoadBalancerConfigs(request)
      expect(list.items.filter((i) => i.metadata.name == lbcName_1).length).toBe(0)

      // remove lbc_2
      await th.deleteLoadBalancerConfig(request, lbcName_2)

      // list of the lbc is not updated immediately, there is some kind of cache
      await page.waitForTimeout(1000)

      list = await th.listLoadBalancerConfigs(request)
      expect(list.items.filter((i) => i.metadata.name == lbcName_2).length).toBe(0)
    })
  })
});