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

    expect(availableVersions.engine['8.0.19-7'].imageHash).toBe('779378a9f52cd9e617d0c356053b4b9c97209b0b759aa819300427836f6f2c66');
    expect(availableVersions.backup['2.12.0'].status).toBe('recommended');

    const allowedVersions = ['6.0.27-21', '7.0.24-13', '7.0.28-15', '7.0.30-16', '8.0.12-4', '8.0.17-6', '8.0.19-7'];

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
