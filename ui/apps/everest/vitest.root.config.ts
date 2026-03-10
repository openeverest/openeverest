import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    projects: ['vite.config.ts', 'vitest.browser.config.ts'],
  },
});
