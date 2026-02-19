import { CI_USER_STORAGE_STATE_FILE } from '@e2e/constants';

export const rbacProject = [
  {
    name: 'pr:rbac',
    testMatch: /.^/,
    dependencies: [
      'pr:rbac:backups',
      'pr:rbac:clusters',
      'pr:rbac:namespaces',
      'pr:rbac:restores',
      'pr:rbac:schedules',
      'pr:rbac:storages',
    ],
  },
  {
    name: 'pr:rbac:backups',
    testDir: './pr/rbac',
    testMatch: /backups\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
  },
  {
    name: 'pr:rbac:clusters',
    testDir: './pr/rbac',
    testMatch: /clusters\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
  },
  {
    name: 'pr:rbac:namespaces',
    testDir: './pr/rbac',
    testMatch: /namespaces\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
  },
  {
    name: 'pr:rbac:restores',
    testDir: './pr/rbac',
    testMatch: /restores\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
  },
  {
    name: 'pr:rbac:schedules',
    testDir: './pr/rbac',
    testMatch: /schedules\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
  },
  {
    name: 'pr:rbac:storages',
    testDir: './pr/rbac',
    testMatch: /storages\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
  },
];
