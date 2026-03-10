import react from '@vitejs/plugin-react-swc';
import * as path from 'path';
import tsconfigPaths from 'vite-tsconfig-paths';
import { defineConfig } from 'vitest/config';
import { playwright } from '@vitest/browser-playwright';

const isCI = process.env.CI === 'true';

export default defineConfig({
  plugins: [tsconfigPaths({ root: '.' }), react()],
  optimizeDeps: {
    include: ['@testing-library/jest-dom/matchers'],
  },
  test: {
    name: 'browser',
    globals: true,
    include: ['src/**/*.browser.test.{ts,tsx}'],
    setupFiles: 'src/setupBrowserTests.ts',
    isolate: true,
    maxWorkers: isCI ? '50%' : undefined,
    browser: {
      enabled: true,
      // Work around pnpm type duplication of vitest instances in monorepo.
      provider: playwright() as never,
      headless: isCI,
      instances: [{ browser: 'chromium' as const }],
      fileParallelism: true,
    },
    reporters: isCI ? ['dot', 'github-actions', 'junit'] : ['verbose'],
    outputFile: isCI
      ? { junit: './test-results/vitest-browser-junit.xml' }
      : undefined,
  },
  resolve: {
    alias: {
      '@percona/ui-lib': path.resolve(__dirname, '../../packages/ui-lib/src'),
      '@percona/design': path.resolve(__dirname, '../../packages/design/src'),
      '@percona/utils': path.resolve(__dirname, '../../packages/utils/src'),
      '@percona/types': path.resolve(__dirname, '../../packages/types/src'),
    },
  },
});
