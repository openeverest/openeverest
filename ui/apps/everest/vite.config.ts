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

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [tsconfigPaths({ root: '.' }), react()],
  server: {
    watch: {
      ignored: path.resolve(__dirname, '.e2e/**/*.*'),
    },
    proxy: {
      '/v1': `http://127.0.0.1:${process.env.API_PORT || '8080'}`,
    },
    open: true,
  },
  build: {
    assetsDir: 'static',
  },
  test: {
    name: 'unit',
    globals: true,
    environment: 'jsdom',
    setupFiles: 'src/setupTests.ts',
    dir: 'src',
    include: ['**/*.test.ts', '**/*.test.tsx', '**/*.spec.ts', '**/*.spec.tsx'],
    exclude: ['**/*.browser.test.{ts,tsx}'],
    isolate: true,
    fileParallelism: true,
    maxWorkers: process.env.CI ? '50%' : undefined,
    reporters: ['verbose'],
  },
  // During prod the libs will be built, so no need to point to src
  ...(process.env.NODE_ENV !== 'production' && {
    resolve: {
      alias: {
        '@percona/ui-lib': path.resolve(__dirname, '../../packages/ui-lib/src'),
        '@percona/design': path.resolve(__dirname, '../../packages/design/src'),
        '@percona/utils': path.resolve(__dirname, '../../packages/utils/src'),
        '@percona/types': path.resolve(__dirname, '../../packages/types/src'),
      },
    },
  }),
});
