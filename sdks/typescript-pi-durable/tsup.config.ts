import { defineConfig } from 'tsup';

// ESM only: pi-durable, pi-ai and chord publish ES modules alone.
export default defineConfig({
  entry: { index: 'src/index.ts' },
  format: ['esm'],
  dts: true,
  sourcemap: true,
  clean: true,
  splitting: false,
  treeshake: true,
  target: 'es2022',
  platform: 'node',
  external: [/^@hatchet-dev\//, /^@earendil-works\//, /^node:/, 'zod'],
});
