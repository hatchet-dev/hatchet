import { defineConfig } from 'tsup';

// One entry per published subpath (see the exports map in package.json). The SDK stays
// external: it is a peer dependency, so the consumer's copy is the one that runs. So do the
// optional peers the Node and Vercel adapters load at runtime (`ws`, `@vercel/functions`)
// and the Node built-ins those two entries alone import; `.` and `./cloudflare` reach
// neither, which scripts/check-edge-entry.mjs proves.
export default defineConfig({
  entry: {
    index: 'src/index.ts',
    'adapters/cloudflare': 'src/adapters/cloudflare.ts',
    'adapters/vercel': 'src/adapters/vercel.ts',
    'adapters/node': 'src/adapters/node.ts',
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
  external: [/^@hatchet-dev\/typescript-sdk/, 'zod', 'ws', '@vercel/functions', /^node:/],
});
