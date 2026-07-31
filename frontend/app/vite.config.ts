import { sentryVitePlugin } from '@sentry/vite-plugin';
import react from '@vitejs/plugin-react';
import path from 'path';
import { defineConfig } from 'vite';

const apiProxyTarget =
  process.env.VITE_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';
const controlPlaneApiProxyTarget =
  process.env.VITE_CONTROL_PLANE_API_PROXY_TARGET ?? 'http://127.0.0.1:8081';

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
    allowedHosts: [
      'local.hatchet-tools.com',
      'local-oss.hatchet-tools.com',
      'local-cloud.hatchet-tools.com',
      'app.dev.hatchet-tools.com',
      'app.localtest.me',
      'localhost',
      '127.0.0.1',
    ],
    proxy: {
      '/api/v1/control-plane': {
        target: controlPlaneApiProxyTarget,
        changeOrigin: true,
        // With no control plane running (OSS dev/e2e), a dead upstream would
        // surface as a 500, which the frontend treats as a transient error
        // and retries. Production OSS returns 404 for unregistered
        // control-plane routes, so mimic that: unreachable upstream means
        // "no control plane".
        configure: (proxy) => {
          proxy.on('error', (_err, _req, res) => {
            if ('writeHead' in res && !res.headersSent) {
              res.writeHead(404, { 'Content-Type': 'application/json' });
              res.end(
                JSON.stringify({
                  errors: [{ description: 'control plane not available' }],
                }),
              );
            }
          });
        },
      },
      // The frontend uses relative `/api/v1/...` paths, so proxy `/api` to the API server.
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
});
