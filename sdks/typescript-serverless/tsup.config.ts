import { defineConfig } from 'tsup';

// Three entries, each published under its own subpath (see the exports map in
// package.json). The SDK stays external: it is a peer dependency, so the consumer's copy
// is the one that runs.
export default defineConfig({
  entry: {
    index: 'src/index.ts',
    'adapters/cloudflare': 'src/adapters/cloudflare.ts',
    'testing/index': 'src/testing/index.ts',
  },
  format: ['esm', 'cjs'],
  dts: true,
  sourcemap: true,
  clean: true,
  splitting: false,
  treeshake: true,
  target: 'es2022',
  platform: 'neutral',
  external: [/^@hatchet-dev\/typescript-sdk/, 'zod'],
});
