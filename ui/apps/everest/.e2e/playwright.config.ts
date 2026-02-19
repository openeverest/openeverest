// everest
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { defineConfig } from '@playwright/test';
import path from 'path';
import { dirname } from 'path';
import { fileURLToPath } from 'url';
import { CI_USER_STORAGE_STATE_FILE } from './constants';
import 'dotenv/config';
import { dbClusterProject } from './pr/db-cluster/project.config';
import { dbClusterDetailsProject } from './pr/db-cluster-details/project.config';
import { dbRestoreProject } from './pr/db-restore/project.config';
import { multinamespacesProject } from './pr/multinamespaces/project.config';
import { noMatchProject } from './pr/no-match/project.config';
import { settingsProject } from './pr/settings/project.config';
import { rbacProject } from './pr/rbac/project.config';

// Convert 'import.meta.url' to the equivalent __dirname
const __dirname = dirname(fileURLToPath(import.meta.url));

/**
 * Read environment variables from file.
 * https://github.com/motdotla/dotenv
 */
// require('dotenv').config();

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  testDir: path.join(__dirname, 'tests-out'),
  outputDir: './test-results',
  testMatch: /.*\.e2e\.(js|ts)x?/,
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: 4,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: [
    ['github'],
    ['list'],
    ['html', { open: 'never', outputFolder: './playwright-report' }],
    ['json', { outputFile: './playwright-report/report.json' }],
  ],
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: process.env.EVEREST_URL || 'http://localhost:8080',
    headless: true,
    extraHTTPHeaders: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'on',
  },

  /* Configure projects for major browsers */
  projects: [
    // ---------------------- global setup and teardown ----------------------
    // global:auth
    {
      name: 'global:auth:ci:setup',
      testDir: './setup',
      testMatch: /auth.setup\.ts/,
      teardown: 'global:auth:ci:teardown',
    },
    {
      name: 'global:auth:ci:teardown',
      testDir: './teardown',
      testMatch: /auth\.teardown\.ts/,
      use: {
        storageState: CI_USER_STORAGE_STATE_FILE,
      },
    },
    // global:backup-storage
    {
      name: 'global:backup-storage:setup',
      testDir: './setup',
      testMatch: /backup-storage.setup\.ts/,
      teardown: 'global:backup-storage:teardown',
      dependencies: ['global:auth:ci:setup'],
    },
    {
      name: 'global:backup-storage:teardown',
      testDir: './teardown',
      testMatch: /backup-storage\.teardown\.ts/,
    },
    // global:monitoring-instance:
    {
      name: 'global:monitoring-instance:setup',
      testDir: './setup',
      testMatch: /monitoring-instance.setup\.ts/,
      teardown: 'global:monitoring-instance:teardown',
      dependencies: ['global:auth:ci:setup'],
    },
    {
      name: 'global:monitoring-instance:teardown',
      testDir: './teardown',
      testMatch: /monitoring-instance\.teardown\.ts/,
    },
    // global:session:
    {
      name: 'global:session:setup',
      testDir: './setup',
      testMatch: /session\.setup\.ts$/,
      teardown: 'global:session:teardown',
    },
    {
      name: 'global:session:teardown',
      testDir: './teardown',
      testMatch: /session\.teardown\.ts$/,
    },

    // ---------------------- PR TESTS ----------------------
    {
      name: 'pr',
      testMatch: /.^/,
      dependencies: [
        'pr:db-cluster',
        'pr:db-cluster-details',
        'pr:multinamespaces',
        'pr:no-match',
        'pr:settings',
        'pr:db-restore',
        ...(process.env.IGNORE_RBAC_TESTS &&
        process.env.IGNORE_RBAC_TESTS !== 'false'
          ? []
          : ['pr:rbac']),
      ],
    },
    ...dbClusterProject,
    ...dbClusterDetailsProject,
    ...dbRestoreProject,
    ...multinamespacesProject,
    ...noMatchProject,
    ...settingsProject,
    ...rbacProject,

    // ---------------------- RELEASE TESTS ----------------------
    // release project
    {
      name: 'release',
      dependencies: ['release:session'],
    },
    // release:session:session project
    {
      name: 'release:session',
      testDir: './release/session',
      dependencies: ['global:session:setup'],
    },

    // -----------------------------------
    // e2e:rbac project
    // {
    //   name: 'rbac-setup',
    //   testDir: './setup',
    //   testMatch: /rbac.setup\.ts/,
    //   use: {
    //     storageState: STORAGE_STATE_FILE,
    //   },
    //   dependencies: ['setup'],
    // },
    // {
    //   name: 'rbac',
    //   use: {
    //     browserName: 'chromium',
    //     channel: 'chrome',
    //     storageState: STORAGE_STATE_FILE,
    //   },
    //   testDir: './pr/rbac',
    //   dependencies: ['setup', 'rbac-setup'],
    // },
    // {
    //   name: 'rbac-teardown',
    //   testDir: './teardown',
    //   testMatch: /rbac\.teardown\.ts/,
    //   use: {
    //     storageState: STORAGE_STATE_FILE,
    //   },
    //   dependencies: ['rbac'],
    // },

    // PR project
    // {
    //   name: 'pr',
    //   use: {
    //     storageState: STORAGE_STATE_FILE,
    //   },
    //   testDir: 'pr',
    //   testIgnore: ['pr/rbac/**/*'],
    //   dependencies: [
    //     'setup',
    //     ...(process.env.IGNORE_RBAC_TESTS &&
    //     process.env.IGNORE_RBAC_TESTS !== 'false'
    //       ? []
    //       : ['rbac', 'rbac-teardown']),
    //   ],
    // },
    // {
    //   name: 'release-rbac-setup',
    //   testDir: './setup',
    //   testMatch: /rbac.setup\.ts/,
    //   use: {
    //     storageState: STORAGE_STATE_FILE,
    //   },
    //   dependencies: ['setup'],
    // },
    // {
    //   name: 'release-rbac',
    //   use: {
    //     browserName: 'chromium',
    //     channel: 'chrome',
    //     storageState: STORAGE_STATE_FILE,
    //   },
    //   testDir: './release/rbac',
    //   dependencies: ['setup', 'release-rbac-setup'],
    // },
    // {
    //   name: 'release-rbac-teardown',
    //   testDir: './teardown',
    //   testMatch: /rbac\.teardown\.ts/,
    //   use: {
    //     storageState: STORAGE_STATE_FILE,
    //   },
    //   dependencies: ['release-rbac'],
    // },
    // {
    //   name: 'release',
    //   use: {
    //     storageState: STORAGE_STATE_FILE,
    //     actionTimeout: 10000,
    //   },
    //   testDir: 'release',
    //   testIgnore: ['release/rbac/*', 'release/session/*'],
    //   dependencies: [
    //     'setup',
    //     ...(process.env.IGNORE_RBAC_TESTS &&
    //     process.env.IGNORE_RBAC_TESTS !== 'false'
    //       ? []
    //       : ['release-rbac', 'release-rbac-teardown']),
    //   ],
    // },

    // Upgrade project
    // {
    //   name: 'upgrade',
    //   use: {
    //     storageState: STORAGE_STATE_FILE,
    //     video: 'retain-on-failure',
    //     actionTimeout: 10000,
    //   },
    //   testDir: 'upgrade',
    //   dependencies: ['setup'],
    // },
  ],
});
