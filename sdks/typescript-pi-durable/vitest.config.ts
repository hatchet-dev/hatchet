import { defineConfig } from 'vitest/config';

export default defineConfig({
  resolve: {
    alias: { '@hatchet-dev/pi-durable': new URL('./src/index.ts', import.meta.url).pathname },
  },
  test: {
    include: ['test/**/*.test.ts'],
    environment: 'node',
    testTimeout: 120_000,
    hookTimeout: 120_000,
  },
});
