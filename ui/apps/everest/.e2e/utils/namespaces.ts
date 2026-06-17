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

import { APIRequestContext, expect, Page } from '@playwright/test';
import { EVEREST_CI_NAMESPACES } from '../constants';
export const getNamespacesFn = async (
  token: string,
  request: APIRequestContext,
  clusterName = 'main'
) => {
  const namespacesInfo = await request.get(
    `/v1/clusters/${clusterName}/namespaces`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  expect(namespacesInfo.ok()).toBeTruthy();
  return namespacesInfo.json();
};

export const setNamespace = async (
  page: Page,
  namespaceName: EVEREST_CI_NAMESPACES
) => {
  await page.getByTestId('k8s-namespace-autocomplete').click();
  await page.getByRole('option', { name: namespaceName }).click();
  await expect(page.getByTestId('text-input-k8s-namespace')).not.toBeEmpty();
};
