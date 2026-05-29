// Copyright (C) 2026 The OpenEverest Contributors
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

import {defineConfig} from '@playwright/test';
import path from 'path';
import {dirname} from 'path';
import {fileURLToPath} from 'url';
import {API_CI_TOKEN, API_TEST_TOKEN} from '@root/constants';
import {TIMEOUTS} from "./constants";

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
  // Some tests are time-consuming
  timeout: TIMEOUTS.ThirtyMinutes,
  // testDir: path.join(__dirname, 'tests'),
  outputDir: path.join(__dirname, 'test-results'),
  testMatch: /.ˆ/,
  // testMatch: /.*\.spec\.(js|ts)x?/,
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  workers: 5,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: [
    ['github'],
    ['list'],
    ['html', {open: 'never', outputFolder: './playwright-report'}],
    ['json', {outputFile: './playwright-report/report.json'}],
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
    timezoneId: 'UTC',
  },

  /* Configure projects for major browsers */
  projects: [
    // ---------------------- global setup and teardown ----------------------
    // global:auth:ci
    {
      name: 'global:auth:ci:setup',
      testDir: 'tests/setup/auth',
      testMatch: /ci\.setup\.ts/,
      teardown: 'global:auth:ci:teardown',
    },
    {
      name: 'global:auth:ci:teardown',
      testDir: 'tests/teardown/auth',
      testMatch: /ci\.teardown\.ts/,
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        },
      },
    },
    // global:auth:test
    {
      name: 'global:auth:test:setup',
      testDir: 'tests/setup/auth',
      testMatch: /test\.setup\.ts/,
    },
    // global:pmm
    {
      name: 'global:pmm:api-key:setup',
      testDir: 'tests/setup',
      testMatch: /pmm\.setup\.ts/,
      use: {
        ignoreHTTPSErrors: true,
      },
    },
    // global:monitoring-config
    {
      name: 'global:monitoring-config:setup',
      testDir: 'tests/setup',
      testMatch: /monitoring-config\.setup\.ts/,
      dependencies: [
        'global:auth:ci:setup',
        'global:pmm:api-key:setup',
      ],
      teardown: 'global:monitoring-config:teardown',
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        },
      },
    },
    {
      name: 'global:monitoring-config:teardown',
      testDir: 'tests/teardown',
      testMatch: /monitoring-config\.teardown\.ts/,
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        },
      },
    },
    // ---------------------- TESTS ----------------------
    // api-tests project
    {
      name: 'api-tests',
      dependencies: [
        'auth',
        'backup-storage',
        'kubernetes',
        'monitoring-config-v2',
        'settings',
        'version',
      ],
    },
    // -------------------- General instances tests ----------------
    // auth tests
    {
      name: 'auth',
      testDir: 'tests',
      testMatch: /auth\.spec\.ts/,
      dependencies: ['global:auth:test:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_TEST_TOKEN]}`,
        }
      },
    },
    // backup-storage-v2 tests
    {
      name: 'backup-storage',
      testDir: 'tests',
      testMatch: /backup-storage\.spec\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // kubernetes tests
    {
      name: 'kubernetes',
      testDir: 'tests',
      testMatch: /kubernetes\.spec\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // monitoring-config-v2 tests
    {
      name: 'monitoring-config-v2',
      testDir: 'tests',
      testMatch: /monitoring-config-v2\.spec\.ts/,
      dependencies: [
        'global:auth:ci:setup',
        'global:pmm:api-key:setup',
      ],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // settings tests
    {
      name: 'settings',
      testDir: 'tests',
      testMatch: /settings\.spec\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // version tests
    {
      name: 'version',
      testDir: 'tests',
      testMatch: /version\.spec\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
  ]
});
