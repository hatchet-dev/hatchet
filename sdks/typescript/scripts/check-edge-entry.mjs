#!/usr/bin/env node
/* eslint-disable no-console */
// Verifies the edge entry point (`@hatchet-dev/typescript-sdk/edge`) imports nothing
// from Node. Bundles dist/edge/index.js for a workerd-like browser target and fails on
// any `node:` specifier or Node builtin reached from it, transitively.
//
// Run after `pnpm run tsc:build` (the `check:edge` script does both). An alternative
// entry can be given as the first argument to inspect another module, for example
// `node scripts/check-edge-entry.mjs dist/index.js` to see what the root reaches.
import { builtinModules } from 'node:module';
import { existsSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { build } from 'esbuild';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const entry = resolve(root, process.argv[2] ?? 'dist/edge/index.js');

if (!existsSync(entry)) {
  console.error(`entry not built: ${entry} is missing. Run \`pnpm run tsc:build\` first.`);
  process.exit(1);
}

// Optional peers the declaration classes load lazily (`mcpTool()`); never reached on a
// serverless code path and not part of the edge contract.
const lazyOptionalPeers = ['@openai/agents', '@anthropic-ai/claude-agent-sdk', '@modelcontextprotocol/sdk'];

const builtins = new Set(builtinModules.flatMap((m) => [m, `node:${m}`]));
const violations = new Map();

const nodeBuiltinDetector = {
  name: 'node-builtin-detector',
  setup(pluginBuild) {
    pluginBuild.onResolve({ filter: /.*/ }, (args) => {
      const isBuiltin = args.path.startsWith('node:') || builtins.has(args.path);
      if (!isBuiltin) return undefined;
      const importer = args.importer ? args.importer.replace(`${root}/`, '') : '<entry>';
      if (!violations.has(args.path)) violations.set(args.path, new Set());
      violations.get(args.path).add(importer);
      // Resolve to an empty stub so the bundle continues and every violation is collected.
      return { path: args.path, namespace: 'node-builtin-stub' };
    });
    pluginBuild.onLoad({ filter: /.*/, namespace: 'node-builtin-stub' }, () => ({
      contents: 'export default {};',
      loader: 'js',
    }));
  },
};

const result = await build({
  entryPoints: [entry],
  bundle: true,
  write: false,
  platform: 'browser',
  conditions: ['workerd', 'worker', 'browser'],
  mainFields: ['browser', 'module', 'main'],
  format: 'esm',
  target: 'es2022',
  logLevel: 'silent',
  metafile: true,
  external: lazyOptionalPeers,
  plugins: [nodeBuiltinDetector],
});

const bundled = Object.keys(result.metafile.inputs);
const sdkInputs = bundled.filter((f) => f.startsWith('dist/'));
const packageInputs = [...new Set(bundled.filter((f) => f.includes('node_modules/')).map((f) => {
  const m = f.match(/node_modules\/(?:\.pnpm\/[^/]+\/node_modules\/)?((?:@[^/]+\/)?[^/]+)/);
  return m ? m[1] : f;
}))].sort();

console.log(`edge entry: bundled ${sdkInputs.length} SDK module(s) and ${packageInputs.length} package(s)`);
if (packageInputs.length) console.log(`packages: ${packageInputs.join(', ')}`);

if (violations.size > 0) {
  console.error('\nedge entry reaches Node builtins:');
  for (const [specifier, importers] of [...violations.entries()].sort()) {
    console.error(`  ${specifier}`);
    for (const importer of [...importers].sort()) console.error(`    from ${importer}`);
  }
  process.exit(1);
}

console.log('edge entry is free of Node builtins');
