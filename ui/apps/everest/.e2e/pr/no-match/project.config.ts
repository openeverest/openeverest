import { CI_USER_STORAGE_STATE_FILE } from '@e2e/constants';
import { PlaywrightTestProject } from '@playwright/test';

export const noMatchProject: PlaywrightTestProject[] = [
  // pr:no-match tests
  {
    name: 'pr:no-match',
    testDir: './pr/no-match',
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
];
