import viteConfig from './vite.config';
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin';
import { playwright } from '@vitest/browser-playwright';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

const dirname = path.dirname(fileURLToPath(import.meta.url));

// Runs every story as a test in Chromium: it must render, and its play
// function, if any, must pass. Unit tests stay on `node --test`.
export default defineConfig({
  // Only the import aliases: the app's plugins and dev proxy don't apply here.
  resolve: { alias: viteConfig.resolve?.alias },
  // Found only once stories start rendering, which makes Vite reload the
  // browser mid-run and drop the test files that were loading.
  optimizeDeps: { include: ['react/jsx-dev-runtime'] },
  plugins: [storybookTest({ configDir: path.join(dirname, '.storybook') })],
  test: {
    name: 'storybook',
    browser: {
      enabled: true,
      headless: true,
      provider: playwright(),
      instances: [{ browser: 'chromium' }],
    },
  },
});
