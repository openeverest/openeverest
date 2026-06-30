import { Page } from '@playwright/test';
import {
  moveForward,
  openDbCreationForm,
  populateAdvancedConfig,
  populateBasicInformation,
  submitWizard,
} from './db-wizard';
import { waitForStatus } from './table';

export const createDbWithParameters = async ({
  page,
  dbName,
  dbType,
  namespace,
  storageClasses,
  addBackupSchedule = false,
  addMonitoring = false,
}: {
  page: Page;
  dbName: string;
  dbType: string;
  namespace: string;
  storageClasses: any[];
  addBackupSchedule?: boolean;
  addMonitoring?: boolean;
}) => {
  await openDbCreationForm(page, dbType);

  // Basic information Step
  await populateBasicInformation(
    page,
    namespace,
    dbName,
    dbType,
    storageClasses[0],
    false,
    null
  );
  await moveForward(page);

  // Skip resources step
  await moveForward(page);

  // Backup Schedules step
  if (addBackupSchedule) {
    await page.getByTestId('create-schedule').click();
    await page.getByTestId('form-dialog-create').click();
  }
  await moveForward(page);

  //A dvanced db config step
  await populateAdvancedConfig(page, dbType, false, '', true, '');
  await moveForward(page);

  // Monitoring step
  if (addMonitoring) {
    await page.getByTestId('switch-input-monitoring').click();
  }

  await submitWizard(page);

  await page.goto('/databases');
  await waitForStatus(page, dbName, 'Initializing', 120000);
  await waitForStatus(page, dbName, 'Up', 720000);
};
