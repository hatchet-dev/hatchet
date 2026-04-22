import { sentryVitePlugin } from '@sentry/vite-plugin';
import react from '@vitejs/plugin-react';
import path from 'path';
import { defineConfig } from 'vite';

const apiProxyTarget =
  process.env.VITE_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';
const controlPlaneApiProxyTarget =
  process.env.VITE_CONTROL_PLANE_API_PROXY_TARGET ?? 'http://127.0.0.1:8081';

export default defineConfig({
  base: './',
  plugins: [
    react(),
    sentryVitePlugin({
      org: 'hatchet',
      project: 'frontend-react',
    }),
    {
      name: 'inject-go-template',
      apply: 'build',
      transformIndexHtml: {
        // When building for production, inject Go template directives that the
        // hatchet-staticfileserver renders at request time to support deployment
        // under any URL subpath (e.g. /hatchet/).
        //
        // See cmd/hatchet-staticfileserver/staticfileserver/server.go for details.
        order: 'post',
        handler(html) {
          return {
            html: html,
            tags: [
              {
                tag: 'base',
                attrs: { href: '{{ .BasePath }}' },
                injectTo: 'head',
              },
              {
                tag: 'script',
                children: 'window.__CONFIG__ = { BASE_PATH: "{{ .BasePath }}" };',
                injectTo: 'head',
              },
            ],
          };
        },
      },
    },
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
      },
      // The frontend uses relative `/api/v1/...` paths, so proxy `/api` to the API server.
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
});

function injectGoTemplate(html: string) {

}
