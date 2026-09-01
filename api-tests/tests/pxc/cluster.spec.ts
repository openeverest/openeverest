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
import * as th from '@tests/utils/api';

const testPrefix = 'pxc';

test.describe.serial('PXC cluster tests', () => {
  test.describe.configure({timeout: 120 * 1000});

  const dbClusterSinName = th.limitedSuffixedName(testPrefix + '-sin'),
    dbClusterMulName = th.limitedSuffixedName(testPrefix + '-mul');

  test.afterAll(async ({request}) => {
    await th.deleteDBCluster(request, dbClusterSinName)
    await th.deleteDBCluster(request, dbClusterMulName)
  })

  test('create/scale/expose/delete single node pxc cluster', async ({request}) => {
    let dbClusterPayload = th.getPXCClusterDataSimple(dbClusterSinName)
    let dbCluster

    try {
      await test.step('create DB cluster', async () => {
        await th.createDBClusterWithData(request, dbClusterPayload)

        // Wait for DB cluster creation.
        await expect(async () => {
          dbCluster = await th.getDBCluster(request, dbClusterSinName)
          expect(dbCluster.spec).toMatchObject(dbClusterPayload.spec)
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      })

      await test.step('scale up DB cluster (engine=3, proxy=3)', async () => {
        await expect(async () => {
          const dbEngineReplicas = 3
          const dbProxyReplicas = 3
          dbCluster = await th.getDBCluster(request, dbClusterSinName)
          dbCluster.spec.engine.replicas = dbEngineReplicas
          dbCluster.spec.proxy.replicas = dbProxyReplicas

          dbCluster = await th.updateDBCluster(request, dbClusterSinName, dbCluster)
          expect(dbCluster.spec.engine.replicas).toBe(dbEngineReplicas)
          expect(dbCluster.spec.proxy.replicas).toBe(dbProxyReplicas)
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      })

      await test.step('expose DB cluster', async () => {
        await expect(async () => {
          dbCluster = await th.getDBCluster(request, dbClusterSinName)
          dbCluster.spec.proxy.expose.type = 'LoadBalancer'

          dbCluster = await th.updateDBCluster(request, dbClusterSinName, dbCluster)
          expect(dbCluster.spec.proxy.expose.type).toMatch('LoadBalancer')
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      })

      await test.step('delete DB cluster', async () => {
        await th.deleteDBCluster(request, dbClusterSinName)
      });

    } finally {
      await th.deleteDBCluster(request, dbClusterSinName)
    }
  })

  test('create/scale/expose/delete multi node pxc cluster', async ({request}) => {
    let dbClusterPayload = th.getPXCClusterDataSimple(dbClusterMulName)
    dbClusterPayload.spec.engine.replicas = 3
    dbClusterPayload.spec.proxy.replicas = 3
    let dbCluster

    try {
      await test.step('create DB cluster(engine=3, proxy=3)', async () => {
        await th.createDBClusterWithData(request, dbClusterPayload)

        await expect(async () => {
          dbCluster = await th.getDBCluster(request, dbClusterMulName)
          expect(dbCluster.spec).toMatchObject(dbClusterPayload.spec)
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      })

      await test.step('scale up DB cluster (engine=5, proxy=5)', async () => {
        await expect(async () => {
          const dbEngineReplicas = 5
          const dbProxyReplicas = 5
          dbCluster = await th.getDBCluster(request, dbClusterMulName)
          dbCluster.spec.engine.replicas = dbEngineReplicas
          dbCluster.spec.proxy.replicas = dbProxyReplicas

          dbCluster = await th.updateDBCluster(request, dbClusterMulName, dbCluster)
          expect(dbCluster.spec.engine.replicas).toBe(dbEngineReplicas)
          expect(dbCluster.spec.proxy.replicas).toBe(dbProxyReplicas)
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      })

      await test.step('expose DB cluster', async () => {
        await expect(async () => {
          dbCluster = await th.getDBCluster(request, dbClusterMulName)
          dbCluster.spec.proxy.expose.type = 'LoadBalancer'

          dbCluster = await th.updateDBCluster(request, dbClusterMulName, dbCluster)
          expect(dbCluster.spec.proxy.expose.type).toMatch('LoadBalancer')
        }).toPass({
          intervals: [1000],
          timeout: 30 * 1000,
        })
      })

      await test.step('delete DB cluster', async () => {
        await th.deleteDBCluster(request, dbClusterMulName)
      });

    } finally {
      await th.deleteDBCluster(request, dbClusterMulName)
    }
  })

});
