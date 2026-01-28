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

import {test, expect} from '@fixtures';
import {EVEREST_CI_NAMESPACE} from '@root/constants';
import {checkError} from '@tests/utils/api';
import {GetDatabaseEngineResponse, GetDatabaseEnginesResponse} from '@support/types/database-engines';

test.describe.parallel('DB engines tests', () => {

  test('check operators are installed', async ({request}) => {
    const enginesList = await request.get(`/v1/namespaces/${EVEREST_CI_NAMESPACE}/database-engines`);

    await checkError(enginesList);
    const engines = (await enginesList.json() as GetDatabaseEnginesResponse).items;

    engines
      .filter((engine) => engine.spec.type !== 'postgresql')
      .forEach((engine) => {
        expect(engine.status?.status).toBe('installed');
      });
  });

  test('get/edit database engine versions', async ({request}) => {
    let engineResponse = await request.get(`/v1/namespaces/${EVEREST_CI_NAMESPACE}/database-engines/percona-server-mongodb-operator`);

    await checkError(engineResponse);

    const engineData: GetDatabaseEngineResponse = await engineResponse.json(),
      availableVersions = engineData.status.availableVersions;

    expect(availableVersions.engine['8.0.4-1'].imageHash).toBe('873b201ce3d66d97b1225c26db392c5043a73cc19ee8db6f2dc1b8efd4783bcf');
    expect(availableVersions.backup['2.11.0'].status).toBe('recommended');

    const allowedVersions = ['6.0.19-16', '7.0.12-7', '7.0.14-8', '7.0.15-9', '8.0.4-1', '8.0.12-4', '8.0.17-6'];

    delete engineData.status;
    engineData.spec.allowedVersions = allowedVersions;

    const updateResponse = await request.put(`/v1/namespaces/${EVEREST_CI_NAMESPACE}/database-engines/percona-server-mongodb-operator`, {
      data: engineData,
    });

    await checkError(updateResponse);

    engineResponse = await request.get(`/v1/namespaces/${EVEREST_CI_NAMESPACE}/database-engines/percona-server-mongodb-operator`);
    await checkError(engineResponse);

    expect((await engineResponse.json() as GetDatabaseEngineResponse).spec.allowedVersions).toEqual(allowedVersions);
  });
})
