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

import { Page, expect } from '@playwright/test';
import { addFirstScheduleInDBWizard } from '../../db-wizard-utils';
import { TIMEOUTS } from '@e2e/constants';

export const backupsStepCheckForPG = async (page: Page) => {
  await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });

  await expect(
    page.getByTestId('step-header').getByText('Scheduled Backups')
  ).toBeVisible();
  await expect(
    page
      .getByTestId('step-description')
      .getByText(
        'Create a task that regularly backs up this database according to your specified schedule.'
      )
  ).toBeVisible();

  await expect(
    page.getByText('You currently do not have any backup schedules set up.')
  ).toBeVisible({ timeout: TIMEOUTS.TenSeconds });

  //  PITR section before adding schedules
  await expect(
    page.getByText('Point-in-time Recovery', { exact: true })
  ).toBeVisible({ timeout: TIMEOUTS.TenSeconds });

  await expect(
    page.getByText(
      'PITR provides continuous backups of your database, enabling you to restore it to'
    )
  ).toBeVisible({ timeout: TIMEOUTS.TenSeconds });

  const enabledPitrCheckbox = page
    .getByTestId('switch-input-pitr-enabled-label')
    .getByRole('checkbox');
  await expect(enabledPitrCheckbox).not.toBeChecked();
  await expect(enabledPitrCheckbox).toBeDisabled();

  await expect(
    page.getByText(
      'Create a task that regularly backs up this database according to your specified schedule.'
    )
  ).toBeVisible();

  await addFirstScheduleInDBWizard(page, 'testFirst');

  //  PITR section after adding schedules
  await expect(enabledPitrCheckbox).toBeChecked();
  await expect(enabledPitrCheckbox).toBeDisabled();

  // -------------- Page control buttons --------------
  await expect(page.getByTestId('db-wizard-previous-button')).toBeVisible();
  await expect(
    page.getByTestId('db-wizard-previous-button')
  ).not.toBeDisabled();

  await expect(page.getByTestId('db-wizard-continue-button')).toBeVisible();
  await expect(
    page.getByTestId('db-wizard-continue-button')
  ).not.toBeDisabled();

  await expect(page.getByTestId('db-wizard-cancel-button')).toBeVisible();
  await expect(page.getByTestId('db-wizard-cancel-button')).not.toBeDisabled();

  // -------------- DB Summary --------------
  await dbSummaryBackupsCheckForPG(page);
};

export const dbSummaryBackupsCheckForPG = async (page: Page) => {
  // -------------- "Database Summary" section (right side) --------------
  // Check for "Backups" panel.
  const backupInfo = page.getByTestId('section-backups');
  await expect(backupInfo.getByText('3. Backups')).toBeVisible();
  // there may be several 'preview-content' elements in 'Backups' section
  const previewContents = backupInfo.getByTestId('preview-content');
  await expect(
    previewContents.getByText('Monthly on day 10 at 1:05 AM')
  ).toBeVisible();
  await expect(previewContents.getByText('PITR Enabled')).toBeVisible();
};
