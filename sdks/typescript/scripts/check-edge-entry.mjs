#!/usr/bin/env node
/* eslint-disable no-console */
// Verifies that an entry point imports nothing from Node. Bundles the entry for a
// workerd-like browser target and fails on any `node:` specifier or Node builtin reached
// from it, transitively, on any package named with `--forbid`, and on any SDK module in the
// bundle that touches a Node-only global (which a bundler cannot see, since nothing imports
// them): `process`, `Buffer`, `setImmediate`, a timer's `.unref()`, `AbortSignal.timeout`
// and `AbortSignal.any`. A use the module feature-detects (`typeof setImmediate`,
// `timer.unref?.()`, `globalThis.Buffer`) is allowed.
//
// Run after `pnpm run tsc:build`:
//   node scripts/check-edge-entry.mjs                       # dist/edge/index.js
//   node scripts/check-edge-entry.mjs dist/core/index.js --forbid axios,nice-grpc
// The `check:edge` and `check:core` scripts build first and then run the two entries.
import { builtinModules } from 'node:module';
import { existsSync, readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { build } from 'esbuild';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');

const args = process.argv.slice(2);
const forbidden = new Set();
let entryArg = 'dist/edge/index.js';
for (let i = 0; i < args.length; i += 1) {
  if (args[i] === '--forbid') {
    for (const pkg of (args[i + 1] ?? '').split(',')) if (pkg) forbidden.add(pkg);
    i += 1;
  } else {
    entryArg = args[i];
  }
}

const entry = resolve(root, entryArg);
const label = entryArg.replace(/^dist\//, '').replace(/\/index\.js$/, '');

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
const sdkInputs = bundled.filter((f) => !f.includes('node_modules/'));
const packageInputs = [...new Set(bundled.filter((f) => f.includes('node_modules/')).map((f) => {
  const m = f.match(/node_modules\/(?:\.pnpm\/[^/]+\/node_modules\/)?((?:@[^/]+\/)?[^/]+)/);
  return m ? m[1] : f;
}))].sort();

console.log(`${label} entry: bundled ${sdkInputs.length} SDK module(s) and ${packageInputs.length} package(s)`);
if (packageInputs.length) console.log(`packages: ${packageInputs.join(', ')}`);

let failed = false;

if (violations.size > 0) {
  failed = true;
  console.error(`\n${label} entry reaches Node builtins:`);
  for (const [specifier, importers] of [...violations.entries()].sort()) {
    console.error(`  ${specifier}`);
    for (const importer of [...importers].sort()) console.error(`    from ${importer}`);
  }
}

const forbiddenHits = packageInputs.filter((pkg) => forbidden.has(pkg));
if (forbiddenHits.length > 0) {
  failed = true;
  console.error(`\n${label} entry bundles forbidden package(s): ${forbiddenHits.join(', ')}`);
}

// These are globals in Node that workerd and browsers lack, or that some runtimes the entry
// targets lack (`AbortSignal.any`, `AbortSignal.timeout`); a module that reads them fails at
// runtime without ever importing anything. A module that feature-detects one may use it: a
// `globalThis.Buffer` access (which the generated bindings guard) and an optional
// `.unref?.()` call are their own guards, and a `typeof` check of the global anywhere in the
// module allows its use in that module.
const nodeGlobals = [
  { what: 'the process or Buffer globals', pattern: /(?<!globalThis\.)\b(?:process|Buffer)\s*\./ },
  { what: 'setImmediate', pattern: /(?<![.\w$])setImmediate\s*\(/, guard: /typeof\s+setImmediate\b/ },
  { what: "a timer's .unref()", pattern: /(?<!\?)\.unref\s*\(/ },
  {
    what: 'AbortSignal.timeout or AbortSignal.any',
    pattern: /\bAbortSignal\s*\.\s*(?:timeout|any)\b/,
    guard: /typeof\s+AbortSignal\s*\.\s*(?:timeout|any)\b/,
  },
];
// Comments describe these globals without touching them, so they are stripped first.
const stripComments = (source) => source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:\\])\/\/.*$/gm, '$1');
const sources = new Map(sdkInputs.map((f) => [f, stripComments(readFileSync(resolve(root, f), 'utf8'))]));
for (const { what, pattern, guard } of nodeGlobals) {
  const hits = sdkInputs.filter((f) => pattern.test(sources.get(f)) && !(guard && guard.test(sources.get(f))));
  if (hits.length > 0) {
    failed = true;
    console.error(`\n${label} entry bundles SDK module(s) that use ${what}:`);
    for (const f of hits) console.error(`  ${f}`);
  }
}

if (failed) process.exit(1);

console.log(`${label} entry is free of Node builtins${forbidden.size ? ' and forbidden packages' : ''}`);
