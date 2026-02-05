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

export const resourcesStepCheckForPG = async (page: Page) => {
  await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });

  await expect(
    page.getByTestId('step-header').getByText('Resources')
  ).toBeVisible();
  await expect(
    page
      .getByTestId('step-description')
      .getByText(
        'Configure the resources your new database will have access to.'
      )
  ).toBeVisible();

  const nodesAccordion = page.getByTestId('nodes-accordion');
  await nodesAccordion.waitFor({ timeout: TIMEOUTS.ThirtySeconds });

  // -------------- "Number of Nodes" section --------------
  await expect(nodesAccordion.getByText('Number of nodes')).toBeVisible();
  await expect(
    nodesAccordion.getByTestId('toggle-button-nodes-1')
  ).toBeVisible();
  await expect(
    nodesAccordion.getByTestId('toggle-button-nodes-2')
  ).toBeVisible();
  await expect(
    nodesAccordion.getByTestId('toggle-button-nodes-3')
  ).toBeVisible();
  await expect(
    nodesAccordion.getByTestId('toggle-button-nodes-custom')
  ).toBeVisible();
  // Select "Number of Nodes = 1"
  await nodesAccordion.getByTestId('toggle-button-nodes-1').click();

  // -------------- "Resource size per node" section --------------
  await expect(
    nodesAccordion.getByText('Resource size per node')
  ).toBeVisible();
  await expect(
    nodesAccordion.getByTestId('node-resources-toggle-button-small')
  ).toBeVisible();
  await expect(
    nodesAccordion.getByTestId('node-resources-toggle-button-medium')
  ).toBeVisible();
  await expect(
    nodesAccordion.getByTestId('node-resources-toggle-button-large')
  ).toBeVisible();
  await expect(
    nodesAccordion.getByTestId('node-resources-toggle-button-custom')
  ).toBeVisible();
  // Select "Resource size per node = Small" and set custom resources
  await nodesAccordion
    .getByTestId('node-resources-toggle-button-small')
    .click();
  await nodesAccordion.getByTestId('text-input-cpu').fill('0.6');
  await nodesAccordion.getByTestId('text-input-memory').fill('1');
  await nodesAccordion.getByTestId('text-input-disk').fill('1');

  await expect(page.getByTestId('cpu-resource-sum')).toHaveText(' = 0.60 CPU');
  await expect(page.getByTestId('memory-resource-sum')).toHaveText(
    ' = 1.00 GB'
  );
  await expect(page.getByTestId('disk-resource-sum')).toHaveText(' = 1.00 Gi');

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
  await dbSummaryResourcesCheckForPG(page);
};

export const dbSummaryResourcesCheckForPG = async (page: Page) => {
  // -------------- "Database Summary" section (right side) --------------
  // Check for "Resources" panel.
  const resourcesInfo = page.getByTestId('section-resources');
  await expect(resourcesInfo.getByText('2. Resources')).toBeVisible();
  // there are several 'preview-content' elements in 'Resources' section
  const previewContents = resourcesInfo.getByTestId('preview-content');
  await expect(
    previewContents.getByText(
      '1 node - CPU - 0.60 CPU; Memory - 1.00 GB; Disk - 1.00 Gi'
    )
  ).toBeVisible();
  await expect(
    previewContents.getByText('1 PG Bouncer - CPU - 1.00 CPU; Memory - 0.03 GB')
  ).toBeVisible();
};
