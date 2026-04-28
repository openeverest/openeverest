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
    globals: true,
    environment: 'jsdom',
    setupFiles: 'src/setupTests.ts',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      reportsDirectory: './coverage',
      exclude: [
        'node_modules/**',
        'src/setupTests.ts',
        '.e2e/**',
        '**/*.test.{ts,tsx}',
        '**/*.spec.{ts,tsx}',
        'src/**/*.d.ts',
      ],
    },
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
