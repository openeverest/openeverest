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

export const DEFAULT_CLUSTER_VERSION = '6.0.16-13';

export const basicInformationStepCheckForPG = async (
  page: Page,
  dbNamespace: string,
  engineVersions,
  recommendedEngineVersions,
  dbName: string
) => {
  await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });

  await expect(
    page.getByTestId('step-header').getByText('Basic information')
  ).toBeVisible();
  await expect(
    page
      .getByTestId('step-description')
      .getByText('Provide the basic information for your new database.')
  ).toBeVisible();

  // namespace
  await basicInformationSelectNamespaceCheck(page, dbNamespace);

  // db cluster name
  await page.getByTestId('text-input-db-name').fill(dbName);

  // engine version
  await page.getByTestId('select-db-version-button').click();
  const dbVersionOptions = page.getByRole('option');

  for (const version of engineVersions.postgresql) {
    await expect(
      dbVersionOptions.filter({ hasText: new RegExp(`^${version}$`) })
    ).toBeVisible();
  }

  const defaultOption = page.getByRole('option', { selected: true });
  expect(await defaultOption.textContent()).toBe(
    recommendedEngineVersions.postgresql
  );

  // TODO .filter({ hasText: DEFAULT_CLUSTER_VERSION })
  await page
    .getByRole('option')
    .filter({ hasText: recommendedEngineVersions.postgresql })
    .click();

  // check page bottom
  await expect(
    page.getByTestId('switch-input-sharding').getByRole('checkbox')
  ).not.toBeVisible();

  // -------------- Page control buttons --------------
  await expect(page.getByTestId('db-wizard-previous-button')).toBeVisible();
  await expect(page.getByTestId('db-wizard-previous-button')).toBeDisabled();

  await expect(page.getByTestId('db-wizard-continue-button')).toBeVisible();
  await expect(
    page.getByTestId('db-wizard-continue-button')
  ).not.toBeDisabled();

  await expect(page.getByTestId('db-wizard-cancel-button')).toBeVisible();
  await expect(page.getByTestId('db-wizard-cancel-button')).not.toBeDisabled();

  // -------------- DB Summary --------------
  await dbSummaryBasicInformationCheckForPG(
    page,
    dbNamespace,
    recommendedEngineVersions,
    dbName
  );
};

export const basicInformationSelectNamespaceCheck = async (
  page: Page,
  dbNamespace: string
) => {
  await page.getByTestId('text-input-k8s-namespace').click();
  const dbNsOption = page.getByRole('option').filter({ hasText: dbNamespace });
  await expect(dbNsOption).toBeVisible();
  await dbNsOption.click();
};

export const dbSummaryBasicInformationCheckForPG = async (
  page: Page,
  dbNamespace: string,
  recommendedEngineVersions,
  dbName: string
) => {
  // -------------- "Database Summary" section (right side) --------------
  // On this step "Basic Information" panel is only filled.
  const basicInfo = page.getByTestId('section-basic-information');
  await expect(basicInfo.getByText('1. Basic Information')).toBeVisible();

  // there are several 'preview-content' elements in 'Basic info' section
  const previewContents = basicInfo.getByTestId('preview-content');
  await expect(
    previewContents.getByText('Namespace: ' + dbNamespace)
  ).toBeVisible();
  await expect(previewContents.getByText('Type: PostgreSQL')).toBeVisible();
  await expect(previewContents.getByText('Name: ' + dbName)).toBeVisible();
  await expect(
    previewContents.getByText(
      'Version: ' + recommendedEngineVersions.postgresql
    )
  ).toBeVisible();
  // --------------------------------------------------------
};
