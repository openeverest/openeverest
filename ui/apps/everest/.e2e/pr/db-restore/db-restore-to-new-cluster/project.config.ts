import { CI_USER_STORAGE_STATE_FILE } from '@e2e/constants';

export const dbClusterName = 'pr-rstr-new';
export const dbRestoreToNewCluster = 'pr:db-restore:db-restore-to-new-cluster';

export const dbRestoreToNewClusterProject = [
  {
    name: `${dbRestoreToNewCluster}:setup`,
    testDir: './pr/db-restore',
    testMatch: /db-restore-to-new-cluster\.setup\.ts/,
    dependencies: ['global:auth:ci:setup', 'global:backup-storage:setup'],
    teardown: `${dbRestoreToNewCluster}:teardown`,
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: `${dbRestoreToNewCluster}:teardown`,
    testDir: './pr/db-restore',
    testMatch: /db-restore-to-new-cluster\.teardown\.ts/,
  },
  {
    name: dbRestoreToNewCluster,
    testDir: './pr/db-restore',
    testMatch: /db-restore-to-new-cluster\.e2e\.ts/,
    dependencies: [`${dbRestoreToNewCluster}:setup`],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
];
