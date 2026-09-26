import { defineConfig } from 'vite';
import type { Plugin } from 'vite';
import react from '@vitejs/plugin-react-swc';

// Same build contract as a real ui-lib plugin (see plugin-inspector/plugin-hub).
const HOST_PROVIDED = ['react', 'react-dom', 'react/jsx-runtime'];

const failOnHostRequire = (): Plugin => ({
  name: 'fail-on-host-require',
  apply: 'build',
  renderChunk(code, chunk) {
    for (const id of HOST_PROVIDED) {
      if (code.includes(`__require("${id}")`)) {
        this.error(`${chunk.fileName} calls require("${id}"); alias that CommonJS dependency to ESM`);
      }
    }
    return null;
  },
});

export default defineConfig(({ command }) => ({
  plugins: [react(), failOnHostRequire()],
  resolve: {
    dedupe: ['@mui/material', '@emotion/react', '@emotion/styled', '@emotion/cache'],
  },
  define:
    command === 'build'
      ? { 'process.env.NODE_ENV': JSON.stringify('production') }
      : undefined,
  build: {
    lib: {
      entry: 'src/main.tsx',
      formats: ['es'],
      fileName: () => 'main.js',
    },
    rollupOptions: {
      external: HOST_PROVIDED,
    },
  },
}));
