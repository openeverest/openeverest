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

import { defineConfig } from 'vitest/config';
import * as path from 'path';

export default defineConfig({
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: 'src/setupTests.ts',
    dir: 'src',
    isolate: true,
    fileParallelism: true,
    maxWorkers: process.env.CI ? '50%' : undefined,
    reporters: ['verbose'],
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
