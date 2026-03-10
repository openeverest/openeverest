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
