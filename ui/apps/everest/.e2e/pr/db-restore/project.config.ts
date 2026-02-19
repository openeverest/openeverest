import { PlaywrightTestProject } from '@playwright/test';
import {
  dbRestoreAction,
  dbRestoreActionProject,
} from './db-restore-action/project.config';
import {
  dbRestoreToNewCluster,
  dbRestoreToNewClusterProject,
} from './db-restore-to-new-cluster/project.config';

export const dbRestoreProject: PlaywrightTestProject[] = [
  {
    name: 'pr:db-restore',
    testMatch: /.^/,
    dependencies: [dbRestoreAction, dbRestoreToNewCluster],
  },
  ...dbRestoreActionProject,
  ...dbRestoreToNewClusterProject,
];
