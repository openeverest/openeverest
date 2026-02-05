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

import { expect, Locator, Page } from '@playwright/test';
import { getBucketNamespacesMap, TIMEOUTS } from '@e2e/constants';
import { beautifyDbTypeName } from '@percona/utils';
import { DbType } from '@percona/types';

export type ScheduleTimeOptions = {
  frequency: 'month' | 'week' | 'day' | 'hour';
  day?: string;
  weekDay?:
    | 'Sundays'
    | 'Mondays'
    | 'Tuesdays'
    | 'Wednesdays'
    | 'Thursdays'
    | 'Fridays'
    | 'Saturdays';
  amPm?: 'AM' | 'PM';
  hour?: string;
  minute?: string;
};

const defaultTimeOptions: ScheduleTimeOptions = {
  frequency: 'month',
  day: '10',
  amPm: 'AM',
  hour: '1',
  minute: '05',
};

export const addFirstScheduleInDBWizard = async (
  page: Page,
  backupStorage?: string
) => {
  const bucketNamespacesMap = getBucketNamespacesMap();
  // checking that we haven't schedules
  await expect(
    page.getByText('You currently do not have any backup schedules set up.')
  ).toBeVisible();

  // creating schedule with schedule modal form dialog
  await openCreateScheduleDialogFromDBWizard(page);
  await fillScheduleModalForm(
    page,
    defaultTimeOptions,
    '1',
    undefined,
    backupStorage
  );
  await page.getByTestId('form-dialog-create').click();
  // checking created schedule in dbWizard schedules list
  await expect(
    page.getByTestId('editable-item').getByText('Monthly on day 10 at 1:05 AM')
  ).toBeVisible();

  const namespace = (
    await page
      .getByTestId('section-basic-information')
      .getByText('Namespace: ', { exact: false })
      .innerText()
  ).split('Namespace: ')[1];

  let matchingBucketNamespace: string;
  if (backupStorage === 'testFirst' || backupStorage === undefined) {
    matchingBucketNamespace = bucketNamespacesMap.find((b) =>
      b[1].includes(namespace)
    )[0];
  } else {
    matchingBucketNamespace = backupStorage;
  }

  if (await checkDbTypeisVisibleInPreview(page, DbType.Mongo)) {
    expect(
      await page.getByText(matchingBucketNamespace).allInnerTexts()
    ).toHaveLength(2);
  } else {
    await expect(page.getByText(matchingBucketNamespace)).toBeVisible();
  }
};

export const addScheduleInDbWizard = async (
  page: Page,
  timeOptions: ScheduleTimeOptions = defaultTimeOptions
) => {
  await openCreateScheduleDialogFromDBWizard(page);
  await fillScheduleModalForm(page, timeOptions, '1', undefined, 'testFirst');
  await page.getByTestId('form-dialog-create').click();
};

const checkDbTypeisVisibleInPreview = async (page: Page, dbType: DbType) => {
  const dbTypeLocator = page.getByText(beautifyDbTypeName(dbType));
  return (await dbTypeLocator.allInnerTexts())?.length > 0;
};

const createScheduleFromTimeOptions = async (
  page: Page,
  timeOptions: ScheduleTimeOptions
) => {
  const { frequency, day, weekDay, amPm, hour, minute } = timeOptions;

  await page.getByTestId('select-selected-time-button').click();

  switch (frequency) {
    case 'month':
      await page.getByTestId('month-option').click();
      await page.getByTestId('select-on-day-button').click();
      await page.getByTestId(day).click();
      await page.getByTestId('select-hour-button').click();
      await page.getByRole('option', { name: hour, exact: true }).click();
      await page.getByTestId('select-minute-button').click();
      await page.getByRole('option', { name: minute, exact: true }).click();
      await page.getByTestId('select-am-pm-button').click();
      await page.getByRole('option', { name: amPm }).click();
      break;
    case 'week':
      await page.getByTestId('week-option').click();
      await page.getByTestId('select-week-day-button').click();
      await page.getByText(weekDay).click();
      await page.getByTestId('select-hour-button').click();
      await page.getByRole('option', { name: hour, exact: true }).click();
      await page.getByTestId('select-minute-button').click();
      await page.getByRole('option', { name: minute, exact: true }).click();
      await page.getByTestId('select-am-pm-button').click();
      await page.getByRole('option', { name: amPm }).click();
      break;
    case 'day':
      await page.getByTestId('day-option').click();
      await page.getByTestId('select-hour-button').click();
      await page.getByRole('option', { name: hour, exact: true }).click();
      await page.getByTestId('select-minute-button').click();
      await page.getByRole('option', { name: minute, exact: true }).click();
      await page.getByTestId('select-am-pm-button').click();
      await page.getByRole('option', { name: amPm }).click();
      break;
    case 'hour':
      await page.getByTestId('hour-option').click();
      await page.getByTestId('select-minute-button').click();
      await page.getByRole('option', { name: minute, exact: true }).click();
      break;
  }
};

// Behaviour of backupStorage parameter is following:
// undefined - we don't change or test storage option
// 'testFirst' - tests the storage option, whichever is first in the combobox
// 'any_other_value' - sets and tests the storage option to desired storage location
export const fillScheduleModalForm = async (
  page: Page,
  timeOptions: ScheduleTimeOptions = defaultTimeOptions,
  retention: string,
  scheduleName?: string,
  backupStorage?: string
) => {
  // const bucketNamespacesMap = getBucketNamespacesMap();
  // TODO can be customizable
  if (await checkDbTypeisVisibleInPreview(page, DbType.Mongo)) {
    await expect(page.getByTestId('radio-option-logical')).toBeChecked();
  }

  if (scheduleName) {
    await page.getByTestId('text-input-schedule-name').fill(scheduleName);
  }
  await expect(page.getByTestId('text-input-schedule-name')).not.toBeEmpty();

  const storageLocationField = page.getByTestId('text-input-storage-location');
  await expect(storageLocationField).not.toBeEmpty();
  if (backupStorage === 'testFirst') {
    await storageLocationField.click();

    const storageOptions = page.getByRole('option');
    await storageOptions.first().click();
  } else if (backupStorage !== undefined) {
    await storageLocationField.click();

    await page.getByRole('option', { name: backupStorage }).click();
  }

  const retentionCopiesField = page.getByTestId('text-input-retention-copies');
  await expect(retentionCopiesField).not.toBeEmpty();

  await retentionCopiesField.fill(retention);

  await createScheduleFromTimeOptions(page, timeOptions);
};

export const openCreateScheduleDialogFromDBWizard = async (page: Page) => {
  await page.getByTestId('create-schedule').click();
  await page
    .getByTestId('new-scheduled-backup-form-dialog')
    .waitFor({ timeout: TIMEOUTS.TenSeconds });
  await expect(
    page.getByTestId('new-scheduled-backup-form-dialog')
  ).toBeVisible();
};

export const clickAddDbClusterBtn = async (page: Page) => {
  await page.getByTestId('add-db-cluster-button').waitFor();
  await page.getByTestId('add-db-cluster-button').click();
};

export const checkAmountOfDbEngines = async (page: Page): Promise<Locator> => {
  await clickAddDbClusterBtn(page);
  await page
    .getByTestId('add-db-cluster-button-menu')
    .getByRole('menuitem')
    .first()
    .waitFor();
  const dbEnginesButtons = page
    .getByTestId('add-db-cluster-button-menu')
    .getByRole('menuitem');
  expect(await dbEnginesButtons.count()).toBe(3);
  return dbEnginesButtons;
};

export const selectDbEngine = async (
  page: Page,
  dbType: 'pxc' | 'psmdb' | 'postgresql'
) => {
  await clickAddDbClusterBtn(page);
  await page
    .getByTestId('add-db-cluster-button-menu')
    .getByRole('menuitem')
    .first()
    .waitFor();
  expect(
    await page.getByTestId('add-db-cluster-button-psmdb').textContent()
  ).toBe('MongoDB');
  expect(
    await page.getByTestId('add-db-cluster-button-pxc').textContent()
  ).toBe('MySQL');
  expect(
    await page.getByTestId('add-db-cluster-button-postgresql').textContent()
  ).toBe('PostgreSQL');

  await page.getByTestId(`add-db-cluster-button-${dbType}`).click();

  await page.waitForURL('/databases/new');
  await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });
};
