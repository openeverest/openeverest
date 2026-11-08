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
import {EVEREST_CI_NAMESPACE} from '@root/constants';
import * as th from '@tests/utils/api';

const url = `/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/monitoring-destinations`,
  getDestination = async (request, name: string) => {
    const response = await request.get(`${url}/${name}`)

    await th.checkError(response)

    return (await response.json()) as {metadata: {labels?: Record<string, string>, deletionTimestamp?: string}}
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
