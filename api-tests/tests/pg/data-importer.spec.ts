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

const testPrefix = 'pg-di'

test.describe.parallel('PG data importer tests', () => {
  test.describe.configure({timeout: 300 * 1000});

  const dbClusterName = th.limitedSuffixedName(testPrefix);

  test.afterAll(async ({request, cli}) => {
    await th.deleteDBCluster(request, dbClusterName);
  })

  test('list data importers', async ({request}) => {
    let res = await request.get(`/v1/data-importers`)
    await th.checkError(res)

    let list = await res.json()
    expect(list.items.some((item: any) => item.metadata.name === 'test-data-importer')).toBe(true)
  })

  test('import data into fresh cluster', async ({request}) => {
    let dbClusterPayload = th.getPGClusterDataSimple(dbClusterName)
    dbClusterPayload.spec.dataSource = {
      dataImport: {
        dataImporterName: 'test-data-importer',
        source: {
          path: "test.sql",
          s3: {
            bucket: "test-bucket",
            region: "us-west-2",
            endpointURL: "https://s3.us-west-2.amazonaws.com",
            credentialsSecretName: "test-s3-credentials",
            // everest-operator will hide this behind a Secret
            accessKeyId: "test-access-key-id",
            secretAccessKey: "test-secret-access-key",
          },
        },
      },
    }

    let dbCluster

    await test.step('prepare env', async () => {
      await th.createDBClusterWithData(request, dbClusterPayload)
    });

    await test.step('check db cluster data importing', async () => {
      await expect(async () => {
        dbCluster = await th.getDBCluster(request, dbClusterName)
        expect(dbCluster.status.status).toBe('importing')
      }).toPass({
        intervals: [1000],
        timeout: 300 * 1000,
      })
    });

    await test.step('check data importer job', async () => {
      await expect(async () => {
        const dijList = await th.getDataImportJobs(request, dbClusterName)
        expect(dijList.items.length).toBe(1)
        const dij = dijList.items[0]
        expect(dij.status.state).toBe('Running')
      }).toPass({
        intervals: [1000],
        timeout: 30 * 1000,
      });
    })
  })
})
