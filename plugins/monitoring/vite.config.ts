import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';

export default defineConfig({
  plugins: [react()],
  build: {
    lib: {
      entry: 'src/main.tsx',
      formats: ['es'],
      fileName: () => 'main.js',
    },
    // No externals — React is passed at runtime via the register() API,
    // so there are no bare `import 'react'` statements in the bundle.
  },
  server: {
    port: 3001,
    cors: true,
  },
});
