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
import {EVEREST_CI_NAMESPACE, TIMEOUTS} from '@root/constants';
import * as th from '@tests/utils/api';

const url = `/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/monitoring-destinations`,
  instancesURL = `/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/instances`,
  getDestination = async (request, name: string) => {
    const response = await request.get(`${url}/${name}`)

    await th.checkError(response)

    return (await response.json()) as {
      metadata: {labels?: Record<string, string>, deletionTimestamp?: string},
      status?: {attachedInstances?: number},
    }
  },
  getPresetSpec = async (request) => {
    const response = await request.get('/v1/clusters/main/cluster-instance-presets/test-preset')

    await th.checkError(response)

    return ((await response.json()) as {spec: Record<string, unknown>}).spec
  }

test.describe('Monitoring destination dryRun', () => {
  test('writes answer as they would and store nothing', async ({request}) => {
    const name = th.limitedSuffixedName('md-dry-run'),
      // The class need not exist: a destination is checked against it at reconcile time.
      payload = {metadata: {name}, spec: {classRef: {name: 'pmm'}}}

    try {
      await test.step('create stores nothing', async () => {
        const response = await request.post(`${url}?dryRun=All`, {data: payload})

        await th.checkError(response)
        expect((await response.json()).metadata.name).toBe(name)

        expect((await request.get(`${url}/${name}`)).status()).toBe(404)
      });

      await th.checkError(await request.post(url, {data: payload}))

      await test.step('update answers with the updated destination and stores nothing', async () => {
        const response = await request.put(`${url}/${name}?dryRun=All`, {
          data: {...payload, metadata: {name, labels: {team: 'dry-run'}}},
        })

        await th.checkError(response)
        expect((await response.json()).metadata.labels).toEqual({team: 'dry-run'})

        expect((await getDestination(request, name)).metadata.labels).toBeUndefined()
      });

      await test.step('delete removes nothing', async () => {
        const response = await request.delete(`${url}/${name}?dryRun=All`)

        expect(response.status()).toBe(204)

        expect((await getDestination(request, name)).metadata.deletionTimestamp).toBeUndefined()
      });
    } finally {
      await request.delete(`${url}/${name}`)
    }
  })
});

test.describe('Monitoring destination in use', () => {
  test.describe.configure({timeout: TIMEOUTS.ThreeMinutes});

  test('deleting it is refused, dry run or not, and changes nothing', async ({request}) => {
    const name = th.limitedSuffixedName('md-in-use'),
      instanceName = th.limitedSuffixedName('md-in-use-db'),
      inUse = 'in use by 1 database; remove the destination from them first',
      presetSpec = await getPresetSpec(request)

    try {
      // The class need not exist: a destination is in use as soon as an Instance lists it.
      await th.checkError(await request.post(url, {data: {metadata: {name}, spec: {classRef: {name: 'pmm'}}}}))
      await th.checkError(await request.post(instancesURL, {
        data: {
          metadata: {name: instanceName},
          spec: {...presetSpec, monitoring: {destinations: [{name: 'metrics', destinationRef: {name}}]}},
        },
      }))

      await expect(async () => {
        expect((await getDestination(request, name)).status?.attachedInstances).toBe(1)
      }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})

      for (const query of ['?dryRun=All', '']) {
        await test.step(`DELETE${query} is refused`, async () => {
          const response = await request.delete(`${url}/${name}${query}`)

          expect(response.status()).toBe(403)
          expect((await response.json()).message).toContain(inUse)

          expect(await getDestination(request, name)).toMatchObject({status: {attachedInstances: 1}})
          expect((await getDestination(request, name)).metadata.deletionTimestamp).toBeUndefined()
        });
      }
    } finally {
      await request.patch(`${instancesURL}/${instanceName}`, {
        headers: {'Content-Type': 'application/merge-patch+json'},
        data: {spec: {monitoring: null}},
      })
      // Allowed once the controller has counted the entry gone.
      await expect(async () => {
        expect([204, 404]).toContain((await request.delete(`${url}/${name}`)).status())
      }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})
      await request.delete(`${instancesURL}/${instanceName}`)
    }
  })
});
