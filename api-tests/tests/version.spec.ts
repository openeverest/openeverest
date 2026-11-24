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

import {expect, test} from '@fixtures'
import {checkError} from '@tests/utils/api';

test.describe.parallel('Everest version tests', () => {

  test('version endpoint', async ({request, cli}) => {
    const version = await request.get('/v1/version')
    await checkError(version)

    const versionJSON = await version.json(),
      gitVersion = await cli.exec('git rev-parse --short HEAD')
    await gitVersion.assertSuccess()

    expect(versionJSON.projectName).toEqual('Everest API Server')
    expect(versionJSON.version).toEqual('v0.0.0-' + gitVersion.getStdOutLines()[0])

    // The window of contract versions the server was built with.
    const contractVersion = await cli.exec('git show HEAD:contract/VERSION'),
      minContractVersion = await cli.exec('git show HEAD:contract/MIN_VERSION')
    await contractVersion.assertSuccess()
    await minContractVersion.assertSuccess()
    expect(versionJSON.contractVersion).toEqual(contractVersion.getStdOutLines()[0])
    expect(versionJSON.minContractVersion).toEqual(minContractVersion.getStdOutLines()[0])
    expect(Array.isArray(versionJSON.upgradableFrom)).toBe(true)
  })
});
