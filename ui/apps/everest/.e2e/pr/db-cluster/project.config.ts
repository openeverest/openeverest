import { CI_USER_STORAGE_STATE_FILE } from '@e2e/constants';
import { PlaywrightTestProject } from '@playwright/test';

export const dbClusterProject: PlaywrightTestProject[] = [
  {
    name: 'pr:db-cluster',
    testMatch: /.^/,
    dependencies: [
      'pr:db-cluster:db-overview',
      'pr:db-cluster:db-list',
      'pr:db-cluster:db-wizard',
    ],
  },
  {
    name: 'pr:db-cluster:db-overview:setup',
    testDir: './pr/db-cluster',
    testMatch: /db-cluster-overview\.setup\.ts/,
    dependencies: ['global:auth:ci:setup'],
    teardown: 'pr:db-cluster:db-overview:teardown',
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster:db-overview:teardown',
    testDir: './pr/db-cluster',
    testMatch: /db-cluster-overview\.teardown\.ts/,
  },
  {
    name: 'pr:db-cluster:db-overview',
    testDir: './pr/db-cluster',
    testMatch: /db-cluster-overview\.e2e\.ts/,
    dependencies: ['pr:db-cluster:db-overview:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster:db-list',
    testDir: './pr/db-cluster',
    testMatch: /db-clusters-list\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster:db-wizard',
    testMatch: /.^/,
    dependencies: [
      'pr:db-cluster:db-wizard:create',
      'pr:db-cluster:db-wizard:create:sharding:psmdb',
      'pr:db-cluster:db-wizard:errors',
    ],
  },
  {
    name: 'pr:db-cluster:db-wizard:create',
    testDir: './pr/db-cluster/db-wizard/create-db-cluster',
    testMatch: /create-db-cluster\.e2e\.ts/,
    dependencies: [
      'global:backup-storage:setup',
      'global:monitoring-instance:setup',
    ],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster:db-wizard:errors',
    testDir: './pr/db-cluster/db-wizard/create-db-cluster',
    testMatch: /errors-handling\.e2e\.ts/,
    dependencies: ['global:backup-storage:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster:db-wizard:create:sharding:psmdb',
    testDir: './pr/db-cluster/db-wizard/create-db-cluster',
    testMatch: /sharding\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
];
