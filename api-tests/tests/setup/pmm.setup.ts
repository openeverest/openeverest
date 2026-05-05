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

import {test as setup, expect} from '@playwright/test';
import * as th from "@tests/utils/api";

const createPmmApiKey = async (request, pmmUrl, saName) => {
  const sa_resp = await request.post(`${pmmUrl}/graph/api/serviceaccounts`, {
    data: {
      name: saName,
      role: 'Admin',
    },
  });

  expect(sa_resp.ok()).toBeTruthy();
  const sa_uuid = (await sa_resp.json()).uid;

  const token_resp = await request.post(`${pmmUrl}/graph/api/serviceaccounts/${sa_uuid}/tokens`, {
    data: {
      name: saName,
      role: 'Admin',
    },
  });
  expect(token_resp.ok()).toBeTruthy();
  return (await token_resp.json()).key;
}

const sa_1_name = th.limitedSuffixedName('-key-1'),
  sa_2_name = th.limitedSuffixedName('-key-2');

setup.describe.serial('Monitoring config setup', () => {
  setup('Create API key in PMM_1', async ({request}) => {
    process.env['PMM1_API_KEY'] = await createPmmApiKey(request, process.env.PMM_1_LOCAL_URL, sa_1_name);
  });

  setup('Create API key in PMM_2', async ({request}) => {
    process.env['PMM2_API_KEY'] = await createPmmApiKey(request, process.env.PMM_2_LOCAL_URL, sa_2_name);
  });
});