import { CI_USER_STORAGE_STATE_FILE } from '@e2e/constants';

export const dbClusterName = 'pr-rstr-act';
export const dbRestoreAction = 'pr:db-restore:db-restore-action';

export const dbRestoreActionProject = [
  {
    name: `${dbRestoreAction}:setup`,
    testDir: './pr/db-restore',
    testMatch: /db-restore-action\.setup\.ts/,
    dependencies: ['global:auth:ci:setup'],
    teardown: `${dbRestoreAction}:teardown`,
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: `${dbRestoreAction}:teardown`,
    testDir: './pr/db-restore',
    testMatch: /db-restore-action\.teardown\.ts/,
  },
  {
    name: dbRestoreAction,
    testDir: './pr/db-restore',
    testMatch: /db-cluster-restore-action\.e2e\.ts/,
    dependencies: [`${dbRestoreAction}:setup`],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
];
