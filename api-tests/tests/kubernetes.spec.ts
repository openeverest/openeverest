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

import {test, expect} from '@fixtures'
import {checkError} from '@tests/utils/api';

test.describe.parallel('K8S cluster tests', () => {

  test('get resource usage', async ({request}) => {
    const r = await request.get(`/v1/resources`),
      resources = await r.json()

    await checkError(r)

    expect(resources).toBeTruthy()
    expect(resources?.capacity).toBeTruthy()
    expect(resources?.available).toBeTruthy()
  })

  test('get cluster info', async ({request}) => {
    const r = await request.get(`/v1/cluster-info`),
      info = await r.json()

    await checkError(r)

    expect(info).toBeTruthy()
    expect(info?.clusterType).toBeTruthy()
    expect(info?.storageClassNames).toBeTruthy()
    expect(info?.storageClassNames).toHaveLength(1)
    expect(info?.storageClasses).toBeTruthy()
    expect(info?.storageClasses).toHaveLength(1)
  })

  test('list and get storage classes', async ({request}) => {
    const r = await request.get(`/v1/clusters/main/storage-classes`),
      list = await r.json(),
      name = list?.items?.[0]?.metadata?.name,
      g = await request.get(`/v1/clusters/main/storage-classes/${name}`),
      got = await g.json(),
      info = await (await request.get(`/v1/cluster-info`)).json()

    await checkError(r)
    await checkError(g)

    expect(list?.items).toHaveLength(1)
    expect(name).toBeTruthy()
    expect(list.items[0]?.provisioner).toBeTruthy()
    // The deprecated /cluster-info must agree until it is removed.
    expect(list.items.map((sc: { metadata: { name: string } }) => sc.metadata.name)).toEqual(info.storageClassNames)
    expect(got.metadata.name).toBe(name)
    expect(got.provisioner).toBe(list.items[0].provisioner)
  })

  test('get missing storage class', async ({request}) => {
    const r = await request.get(`/v1/clusters/main/storage-classes/does-not-exist`)

    expect(r.status()).toBe(404)
  })
});