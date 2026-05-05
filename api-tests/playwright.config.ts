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
        'database-engines',
        'kubernetes',
        'loadbalancer-config',
        'monitoring-config',
        'settings',
        'version',
        'pg',
        'psmdb',
        'pxc',
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
    // backup-storage tests
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
    // database-engines tests
    {
      name: 'database-engines',
      testDir: 'tests',
      testMatch: /database-engines\.spec\.ts/,
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
    // lbc tests
    {
      name: 'loadbalancer-config',
      testDir: 'tests',
      testMatch: /loadbalancer-config\.spec\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // monitoring-config tests
    {
      name: 'monitoring-config',
      testDir: 'tests',
      testMatch: /monitoring-config\.spec\.ts/,
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
    // ------------------------ PG tests ----------------------------
    // pg:backup-storage:setup
    {
      name: 'pg:backup-storage:setup',
      testDir: 'tests/pg/setup',
      testMatch: /backup-storage\.setup\.ts/,
      teardown: 'pg:backup-storage:teardown',
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // pg:backup-storage:teardown
    {
      name: 'pg:backup-storage:teardown',
      testDir: 'tests/pg/teardown',
      testMatch: /backup-storage\.teardown\.ts/,
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // pg project
    {
      name: 'pg',
      dependencies: [
        'pg:backup',
        'pg:cluster',
        'pg:data-importer',
        'pg:monitoring',
        'pg:restore',
      ],
    },
    // pg:backup tests
    // pg:backup:db:setup
    {
      name: 'pg:backup:db:setup',
      testDir: 'tests/pg/setup',
      testMatch: /backup-db\.setup\.ts/,
      teardown: 'pg:backup:db:teardown',
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // pg:backup:db:teardown
    {
      name: 'pg:backup:db:teardown',
      testDir: 'tests/pg/teardown',
      testMatch: /backup-db\.teardown\.ts/,
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // pg:backup
    {
      name: 'pg:backup',
      testDir: 'tests/pg',
      testMatch: /backup\.spec\.ts/,
      dependencies: [
        'global:auth:ci:setup',
        'pg:backup-storage:setup',
        'pg:backup:db:setup',
      ],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // pg:cluster tests
    {
      name: 'pg:cluster',
      testDir: 'tests/pg',
      testMatch: /cluster\.spec\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // pg:data-importer tests
    {
      name: 'pg:data-importer',
      testDir: 'tests/pg',
      testMatch: /data-importer\.spec\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // pg:monitoring tests
    {
      name: 'pg:monitoring',
      testDir: 'tests',
      testMatch: /monitoring\.spec\.ts/,
      dependencies: [
        'global:auth:ci:setup',
        'global:monitoring-config:setup',
      ],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // pg:restore tests
    // pg:restore:db:setup
    {
      name: 'pg:restore:db:setup',
      testDir: 'tests/pg/setup',
      testMatch: /restore-db\.setup\.ts/,
      teardown: 'pg:restore:db:teardown',
      dependencies: [
        'global:auth:ci:setup',
        'pg:backup-storage:setup',
      ],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // pg:restore:db:teardown
    {
      name: 'pg:restore:db:teardown',
      testDir: 'tests/pg/teardown',
      testMatch: /restore-db\.teardown\.ts/,
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // pg:restore
    {
      name: 'pg:restore',
      testDir: 'tests/pg',
      testMatch: /restore\.spec\.ts/,
      dependencies: [
        'global:auth:ci:setup',
        'pg:backup-storage:setup',
        'pg:restore:db:setup',
      ],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // ------------------------ PSMDB tests --------------------------------
    // psmdb project
    {
      name: 'psmdb',
      dependencies: [
        'psmdb:cluster',
      ],
    },
    // psmdb:cluster tests
    {
      name: 'psmdb:cluster',
      testDir: 'tests/psmdb',
      testMatch: /cluster\.spec\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
    // ------------------------- PXC tests ---------------------------------
    // pxc project
    {
      name: 'pxc',
      dependencies: [
        'pxc:cluster',
      ],
    },
    // pxc:cluster tests
    {
      name: 'pxc:cluster',
      testDir: 'tests/pxc',
      testMatch: /cluster\.spec\.ts/,
      dependencies: ['global:auth:ci:setup'],
      use: {
        extraHTTPHeaders: {
          'Authorization': `Bearer ${process.env[API_CI_TOKEN]}`,
        }
      },
    },
  ]
});
