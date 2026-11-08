import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// In development Vite serves the front-end on 5103 and forwards /api to the
// Go server on 8103. In production the Go binary serves the built files.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5103,
    strictPort: true,
    proxy: { '/api': 'http://localhost:8103' },
  },
  build: { target: 'es2022' },
  test: { include: ['src/**/*.test.ts'] },
});
