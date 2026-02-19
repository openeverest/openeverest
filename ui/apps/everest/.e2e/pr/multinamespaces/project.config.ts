import { CI_USER_STORAGE_STATE_FILE } from '@e2e/constants';
import { PlaywrightTestProject } from '@playwright/test';

export const multinamespacesProject: PlaywrightTestProject[] = [
  {
    name: 'pr:multinamespaces',
    testMatch: /.^/,
    dependencies: [
      'pr:multinamespaces:db-wizard',
      'pr:multinamespaces:monitoring',
      'pr:multinamespaces:storage-location',
    ],
  },
  // pr:multinamespaces:db-wizard tests
  {
    name: 'pr:multinamespaces:db-wizard',
    testDir: './pr/multinamespaces',
    testMatch: /db-wizard\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  // pr:multinamespaces:monitoring tests
  {
    name: 'pr:multinamespaces:monitoring',
    testDir: './pr/multinamespaces',
    testMatch: /monitoring\.e2e\.ts/,
    dependencies: ['global:monitoring-instance:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  // pr:multinamespaces:storage-location tests
  {
    name: 'pr:multinamespaces:storage-location:setup',
    testDir: './pr/multinamespaces',
    testMatch: /storage-location\.setup\.ts/,
    dependencies: ['global:auth:ci:setup'],
    teardown: 'pr:multinamespaces:storage-location:teardown',
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:multinamespaces:storage-location:teardown',
    testDir: './pr/multinamespaces',
    testMatch: /storage-location\.teardown\.ts/,
  },
  {
    name: 'pr:multinamespaces:storage-location',
    testDir: './pr/multinamespaces',
    testMatch: /storage-location\.e2e\.ts/,
    dependencies: [
      'global:backup-storage:setup',
      'pr:multinamespaces:storage-location:setup',
    ],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
];
