import { defineConfig } from 'vite';
import viteReact from '@vitejs/plugin-react';
import { TanStackRouterVite } from '@tanstack/router-plugin/vite';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const __dirname = dirname(fileURLToPath(import.meta.url));

// https://vitejs.dev/config/
export default defineConfig({
   plugins: [TanStackRouterVite({ autoCodeSplitting: true }), viteReact()],
   // The Cognito SDK reads Node's `global`; the production build handles it,
   // but the dev server's pre-bundled dependencies need it defined.
   optimizeDeps: {
      esbuildOptions: { define: { global: 'globalThis' } },
   },
   resolve: {
      alias: {
         '@': resolve(__dirname, './src'),
      },
   },
   test: {
      environment: 'jsdom',
      globals: true,
      setupFiles: ['./src/test-setup.ts'],
      css: false,
   },
});
