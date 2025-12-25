import { sentryVitePlugin } from '@sentry/vite-plugin';
import path from 'path';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [
    react(),
    sentryVitePlugin({
      org: 'hatchet',
      project: 'frontend-react',
    }),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    sourcemap: true,
  },
  server: {
    /**
     * For E2E we run on `app.localtest.me` so cookies can be set with Domain=localtest.me.
     * `localtest.me` resolves to 127.0.0.1 without requiring /etc/hosts changes.
     */
    host: true,
    port: 5173,
    /**
     * When running behind Caddy TLS (see `Caddyfile`), the browser origin is https://...:443
     * but Vite will default the HMR websocket port to the dev server port (5173), which breaks
     * HMR due to mixed-content / unreachable wss://...:5173.
     *
     * We allow the dev script to opt into the correct public HMR connection settings via env vars.
     */
    hmr:
      process.env.VITE_HMR_CLIENT_PORT ||
      process.env.VITE_HMR_HOST ||
      process.env.VITE_HMR_PROTOCOL
        ? {
            clientPort: process.env.VITE_HMR_CLIENT_PORT
              ? Number(process.env.VITE_HMR_CLIENT_PORT)
              : undefined,
            host: process.env.VITE_HMR_HOST,
            protocol: process.env.VITE_HMR_PROTOCOL as 'ws' | 'wss' | undefined,
          }
        : undefined,
    allowedHosts: [
      'app.dev.hatchet-tools.com',
      'app.localtest.me',
      'localhost',
      '127.0.0.1',
    ],
    proxy: {
      // The frontend uses relative `/api/v1/...` paths, so proxy `/api` to the API server.
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
});
