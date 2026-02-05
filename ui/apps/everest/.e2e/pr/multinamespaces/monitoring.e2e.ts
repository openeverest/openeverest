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

import { expect, test } from '@playwright/test';
import { moveForward } from '@e2e/utils/db-wizard';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import { setNamespace } from '@e2e/utils/namespaces';
import { selectDbEngine } from '../db-cluster/db-wizard/db-wizard-utils';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';

test.describe.serial('Namespaces: Monitoring availability', () => {
  let token: string;

  test.beforeAll(async ({}) => {
    token = await getCITokenFromLocalStorage();
  });

  test('Monitoring autocomplete in DB Wizard has only endpoints in selected namespace', async ({
    page,
  }) => {
    await page.goto('/databases');
    await selectDbEngine(page, 'pxc');

    // setting everest-pxc namespace
    await setNamespace(page, EVEREST_CI_NAMESPACES.PXC_ONLY);

    // Resources Step
    await moveForward(page);
    // Backups step
    await moveForward(page);
    // Advanced Configuration step
    await moveForward(page);
    // Monitoring Step
    await moveForward(page);
    const monitoringCheckbox = page
      .getByTestId('switch-input-monitoring-label')
      .getByRole('checkbox');
    expect(await monitoringCheckbox.isChecked()).toBeFalsy();
    await monitoringCheckbox.check();

    await page.getByTestId('text-input-monitoring-instance').click();
    const namespaces = page.getByRole('option');
    // This might eventually fail if someday we change the namespaces env variable logic
    // But now, we know we add one endpoint per namespace
    expect(await namespaces.count()).toBeGreaterThanOrEqual(1);
  });
});
