import { CI_USER_STORAGE_STATE_FILE } from '@e2e/constants';
import { PlaywrightTestProject } from '@playwright/test';

export const dbClusterDetailsProject: PlaywrightTestProject[] = [
  {
    name: 'pr:db-cluster-details',
    testMatch: /.^/,
    dependencies: [
      'pr:db-cluster-details:components',
      'pr:db-cluster-details:edit-db-cluster',
      'pr:db-cluster:db-wizard',
    ],
  },
  {
    name: 'pr:db-cluster-details:components:setup',
    testDir: './pr/db-cluster-details/components',
    testMatch: /components\.setup\.ts/,
    dependencies: ['global:auth:ci:setup'],
    teardown: 'pr:db-cluster-details:components:teardown',
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster-details:components:teardown',
    testDir: './pr/db-cluster-details/components',
    testMatch: /components\.teardown\.ts/,
  },
  {
    name: 'pr:db-cluster-details:components',
    testDir: './pr/db-cluster-details/components',
    testMatch: /components\.e2e\.ts/,
    dependencies: ['pr:db-cluster-details:components:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster-details:edit-db-cluster',
    testMatch: /.^/,
    dependencies: [
      'pr:db-cluster-details:edit-db-cluster:db-version-upgrade',
      // 'pr:db-cluster-details:edit-db-cluster:',
      // 'pr:db-cluster-details:edit-db-cluster:',
    ],
  },
  {
    name: 'pr:db-cluster-details:edit-db-cluster:db-version-upgrade',
    testDir: './pr/db-cluster-details/edit-db-cluster',
    testMatch: /db-version-upgrade\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
];
