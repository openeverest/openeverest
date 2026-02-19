import { CI_USER_STORAGE_STATE_FILE } from '@e2e/constants';
import { PlaywrightTestProject } from '@playwright/test';

export const settingsProject: PlaywrightTestProject[] = [
  // pr:settings tests
  {
    name: 'pr:settings',
    testMatch: /.^/,
    dependencies: [
      'pr:settings:monitoring-instance',
      'pr:settings:backup-storage',
      'pr:settings:namespace',
      'pr:settings:psp',
      'pr:settings:operator-upgrade',
    ],
  },
  // pr:settings:backup-storage tests
  {
    name: 'pr:settings:backup-storage',
    testDir: './pr/settings',
    testMatch: /backup-storage\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  // pr:settings:monitoring-instance tests
  {
    name: 'pr:settings:monitoring-instance',
    testDir: './pr/settings',
    testMatch: /monitoring-instance\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  // pr:settings:namespace tests
  {
    name: 'pr:settings:namespace',
    testDir: './pr/settings',
    testMatch: /namespaces-list\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  // pr:settings:psp tests
  {
    name: 'pr:settings:psp',
    testDir: './pr/settings',
    testMatch: /pod-scheduling-policies\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  // pr:settings:operator-upgrade tests
  {
    name: 'pr:settings:operator-upgrade',
    testDir: './pr/settings',
    testMatch: /operator-upgrade\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
];
