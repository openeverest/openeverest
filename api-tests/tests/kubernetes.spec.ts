// everest
// Copyright (C) 2023 Percona LLC
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

import {test, expect} from '@fixtures'
import {checkError} from '@tests/utils/api';

test.describe.parallel('K8S cluster tests', () => {
  test('list and get storage classes', async ({request}) => {
    const r = await request.get(`/v1/clusters/main/storage-classes`),
      list = await r.json(),
      name = list?.items?.[0]?.metadata?.name,
      g = await request.get(`/v1/clusters/main/storage-classes/${name}`),
      got = await g.json()

    await checkError(r)
    await checkError(g)

    expect(list?.items).toHaveLength(1)
    expect(name).toBeTruthy()
    expect(list.items[0]?.provisioner).toBeTruthy()
    expect(got.metadata.name).toBe(name)
    expect(got.provisioner).toBe(list.items[0].provisioner)
  })

  test('get missing storage class', async ({request}) => {
    const r = await request.get(`/v1/clusters/main/storage-classes/does-not-exist`)

    expect(r.status()).toBe(404)
  })
});
