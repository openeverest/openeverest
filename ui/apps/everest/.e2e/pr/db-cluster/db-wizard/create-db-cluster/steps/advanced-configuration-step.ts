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
import { TIMEOUTS } from '@e2e/constants';
// import {SOURCE_RANGE_PLACEHOLDER} from "../../../../../constants";

export const advancedConfigurationStepCheckForPG = async (page: Page) => {
  await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });
  await expect(
    page.getByTestId('step-header').getByText('Advanced Configurations')
  ).toBeVisible();

  // Select Storage Class - mandatory param
  await advancedConfigurationSelectFirstStorageClass(page);

  // await page.getByLabel('Enable External Access').check();
  // expect(
  //   await page.getByLabel('Enable External Access').isChecked()
  // ).toBeTruthy();
  //
  // const sourceRangeFirstField = page.getByTestId('text-input-source-ranges.0.source-range');
  // expect(sourceRangeFirstField).toHaveValue('');
  // expect(sourceRangeFirstField).toHaveAttribute('placeholder', SOURCE_RANGE_PLACEHOLDER);
  // await sourceRangeFirstField.fill('192.168.1.1/24');
  //
  // await page.getByTestId('add-text-input-button').click();
  //
  // const sourceRangeSecondField = page
  //     .getByTestId('text-input-source-ranges.1.source-range');
  // expect(sourceRangeSecondField).toHaveValue('');
  // expect(sourceRangeSecondField).toHaveAttribute('placeholder', SOURCE_RANGE_PLACEHOLDER);
  // await sourceRangeSecondField.fill('192.168.1.0');

  // await page
  //   .getByTestId('switch-input-engine-parameters-enabled-label')
  //   .getByRole('checkbox')
  //   .check();
  //
  // await page
  //   .getByTestId('text-input-engine-parameters')
  //   .fill('max_allowed_packet: 128M');

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
  await dbSummaryAdvancedConfigurationCheckForPG(page);
};

export const advancedConfigurationSelectFirstStorageClass = async (
  page: Page
) => {
  const storageClass = page.getByTestId('text-input-storage-class');
  await storageClass.waitFor({ timeout: TIMEOUTS.TenSeconds });
  await storageClass.click();
  await page.getByRole('option').first().click();
};

export const dbSummaryAdvancedConfigurationCheckForPG = async (page: Page) => {
  // -------------- "Database Summary" section (right side) --------------
  // Check for "Advanced Configurations" panel.
  const advConfigInfo = page.getByTestId('section-advanced-configurations');
  await expect(
    advConfigInfo.getByText('4. Advanced Configurations')
  ).toBeVisible();
  // there are several 'preview-content' elements in 'Advanced Configurations' section
  const previewContents = advConfigInfo.getByTestId('preview-content');
  await expect(
    previewContents.getByText('Storage class: local-path')
  ).toBeVisible();
  await expect(
    previewContents.getByText('Ext. access: disabled')
  ).toBeVisible();
  await expect(
    previewContents.getByText(
      'Pod scheduling policy: everest-default-postgresql'
    )
  ).toBeVisible();
};
