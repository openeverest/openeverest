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

import { APIRequestContext, expect } from '@playwright/test';
import { EVEREST_CI_CLUSTER } from '@e2e/constants';

export const getBackupStorage = async (
  request: APIRequestContext,
  namespace: string,
  name: string,
  token: string
) => {
  const response = await request.get(
    `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/backup-storages/${name}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  expect(response.status() === 200).toBeTruthy();
  return await response.json();
};

export const deleteBackupStorage = async (
  request: APIRequestContext,
  namespace: string,
  name: string,
  token: string
) => {
  const response = await request.delete(
      `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/backup-storages/${name}`,
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
      }
    ),
    code = response.status();
  expect(code === 204 || code === 404).toBeTruthy();
};
