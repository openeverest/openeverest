import { defineConfig } from '@playwright/test';

const PORT = Number(process.env.PORT ?? 4173);
const HOST_LABEL = process.env.HOST_LABEL ?? 'host-default';

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  retries: 0,
  timeout: 30_000,
  reporter: [['list'], ['json', { outputFile: `results/${HOST_LABEL}.json` }]],
  use: {
    baseURL: `http://localhost:${PORT}`,
    browserName: 'chromium',
    viewport: { width: 1280, height: 800 },
    trace: 'retain-on-failure',
  },
  webServer: {
    command: 'node server.mjs',
    url: `http://localhost:${PORT}/`,
    reuseExistingServer: false,
    env: { PORT: String(PORT), HOST_DIST: process.env.HOST_DIST ?? '../apps/everest/dist' },
  },
});
