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
    // pr project
    {
      name: 'pr',
      dependencies: [
        'pr:db-cluster',
        'pr:db-cluster-details',
        'pr:multinamespaces',
        'pr:no-match',
        'pr:settings',
      ],
    },
    // pr:db-cluster tests
    {
      name: 'pr:db-cluster',
      dependencies: [
        'pr:db-cluster:db-overview',
        'pr:db-cluster:db-list',
        'pr:db-cluster:db-wizard',
      ],
    },
    // pr:db-cluster:db-overview tests
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
    // pr:db-cluster:db-list tests
    {
      name: 'pr:db-cluster:db-list',
      testDir: './pr/db-cluster',
      testMatch: /db-clusters-list\.e2e\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        storageState: CI_USER_STORAGE_STATE_FILE,
      },
    },
    // pr:db-cluster:db-wizard tests
    {
      name: 'pr:db-cluster:db-wizard',
      dependencies: [
        'pr:db-cluster:db-wizard:create',
        'pr:db-cluster:db-wizard:create:sharding:psmdb',
        'pr:db-cluster:db-wizard:errors',
      ],
    },
    // pr:db-cluster:db-wizard:create tests
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
    // pr:db-cluster:db-wizard:errors tests
    {
      name: 'pr:db-cluster:db-wizard:errors',
      testDir: './pr/db-cluster/db-wizard/create-db-cluster',
      testMatch: /errors-handling\.e2e\.ts/,
      dependencies: ['global:backup-storage:setup'],
      use: {
        storageState: CI_USER_STORAGE_STATE_FILE,
      },
    },
    // pr:db-cluster:db-wizard:create:sharding:psmdb tests
    {
      name: 'pr:db-cluster:db-wizard:create:sharding:psmdb',
      testDir: './pr/db-cluster/db-wizard/create-db-cluster',
      testMatch: /sharding\.e2e\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        storageState: CI_USER_STORAGE_STATE_FILE,
      },
    },

    // pr:db-cluster-details tests
    {
      name: 'pr:db-cluster-details',
      dependencies: [
        'pr:db-cluster-details:components',
        'pr:db-cluster-details:edit-db-cluster',
        'pr:db-cluster:db-wizard',
      ],
    },
    // pr:db-cluster-details:components:setup tests
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
    // pr:db-cluster-details:components:teardown tests
    {
      name: 'pr:db-cluster-details:components:teardown',
      testDir: './pr/db-cluster-details/components',
      testMatch: /components\.teardown\.ts/,
    },
    // pr:db-cluster-details:components tests
    {
      name: 'pr:db-cluster-details:components',
      testDir: './pr/db-cluster-details/components',
      testMatch: /components\.e2e\.ts/,
      dependencies: ['pr:db-cluster-details:components:setup'],
      use: {
        storageState: CI_USER_STORAGE_STATE_FILE,
      },
    },
    // pr:db-cluster-details:edit-db-cluster tests
    {
      name: 'pr:db-cluster-details:edit-db-cluster',
      dependencies: [
        'pr:db-cluster-details:edit-db-cluster:db-version-upgrade',
        // 'pr:db-cluster-details:edit-db-cluster:',
        // 'pr:db-cluster-details:edit-db-cluster:',
      ],
    },
    // pr:db-cluster-details:edit-db-cluster:db-version-upgrade tests
    {
      name: 'pr:db-cluster-details:edit-db-cluster:db-version-upgrade',
      testDir: './pr/db-cluster-details/edit-db-cluster',
      testMatch: /db-version-upgrade\.e2e\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        storageState: CI_USER_STORAGE_STATE_FILE,
      },
    },

    // pr:multinamespaces tests
    {
      name: 'pr:multinamespaces',
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

    // pr:no-match tests
    {
      name: 'pr:no-match',
      testDir: './pr/no-match',
      dependencies: ['global:auth:ci:setup'],
      use: {
        storageState: CI_USER_STORAGE_STATE_FILE,
      },
    },

    // pr:rbac tests
    {
      name: 'pr:rbac',
      testDir: './pr/rbac',
      dependencies: ['pr:rbac:backups'],
    },
    // pr:rbac:backups tests
    {
      name: 'pr:rbac:backups',
      testDir: './pr/rbac',
      testMatch: /backups\.e2e\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        storageState: CI_USER_STORAGE_STATE_FILE,
      },
    },

    // pr:settings tests
    {
      name: 'pr:settings',
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
