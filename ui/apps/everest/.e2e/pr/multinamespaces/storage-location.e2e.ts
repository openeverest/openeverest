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

import { test, expect } from '@playwright/test';
import { moveForward } from '@e2e/utils/db-wizard';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import { findDbAndClickRow } from '@e2e/utils/db-clusters-list';
import { DBClusterDetailsTabs } from '../../../src/pages/db-cluster-details/db-cluster-details.types';
import {
  openCreateScheduleDialogFromDBWizard,
  selectDbEngine,
} from '../db-cluster/db-wizard/db-wizard-utils';

const pgDbName = 'pr-mul-ns-db-pg';
const pxcDbName = 'pr-mul-ns-db-pxc';

test.describe.parallel('Namespaces: Backup Storage availability', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/databases');
  });

  test('Backup Storage autocomplete in DB Wizard has only backup storages in selected namespace', async ({
    page,
  }) => {
    await selectDbEngine(page, 'pxc');

    // setting everest-pxc namespace
    const namespacesAutocomplete = page.getByTestId(
      'k8s-namespace-autocomplete'
    );
    await namespacesAutocomplete.click();
    await page
      .getByRole('option', { name: EVEREST_CI_NAMESPACES.PXC_ONLY })
      .click();

    // Resources Step
    await moveForward(page);
    // Backups step
    await moveForward(page);

    await openCreateScheduleDialogFromDBWizard(page);
    const storageLocationAutocomplete = page.getByTestId(
      'text-input-storage-location'
    );
    await storageLocationAutocomplete.click();

    expect(await page.getByRole('option').count()).toBe(1);
    await page.getByRole('option').click();
    await page.getByTestId('form-dialog-create').click();

    const pitrCheckbox = page
      .getByTestId('switch-input-pitr-enabled')
      .getByRole('checkbox');
    await pitrCheckbox.setChecked(true);

    const pitrLocationAutocomplete = page.getByTestId(
      'pitr-storage-location-autocomplete'
    );
    await pitrLocationAutocomplete.click();
    expect(await page.getByRole('option').count()).toBe(1);
  });

  test('Backup storage autocomplete in create new backup modal has only available storages for namespace', async ({
    page,
  }) => {
    await findDbAndClickRow(page, pgDbName);

    // go to backups tab in db-cluster details
    const backupsTab = page.getByTestId(DBClusterDetailsTabs.backups);
    await backupsTab.click();

    // check backup storage dropdown in new backup modal
    await page.getByTestId('menu-button').click();
    await page.getByTestId('now-menu-item').click();
    await page.getByTestId('storage-location-autocomplete').click();
    await expect(page.getByRole('option')).toBeVisible();
    expect(await page.getByRole('option').count()).toBe(1);
  });

  test('Backup storage autocomplete in create schedule modal has only available storages for namespace', async ({
    page,
  }) => {
    await findDbAndClickRow(page, pxcDbName);

    // go to backups tab in db-cluster details
    const backupsTab = page.getByTestId(DBClusterDetailsTabs.backups);
    await backupsTab.click();

    // check backup storage dropdown in create schedule modal
    await page.getByTestId('menu-button').click();
    await page.getByTestId('schedule-menu-item').click();
    await page.getByTestId('storage-location-autocomplete').click();
    await expect(page.getByRole('option')).toBeVisible();
    expect(await page.getByRole('option').count()).toBe(1);
  });
});
