#!/usr/bin/env node
/* eslint-disable no-console */
// Verifies the package's `exports` map against the built `dist/`: every documented import
// path resolves from both a native ESM module and a CommonJS module, every directory that
// ships an `index.js` has an explicit entry (a wildcard cannot map a bare directory to its
// index), and every entry's target exists.
//
// The published package root is `dist/` (`publish:ci` copies package.json there and publishes
// from it), so the map is checked from a scratch package whose node_modules links to `dist/`.
// Run after `pnpm run tsc:build` (the `check:exports` script does both).
import { createRequire } from 'node:module';
import {
  cpSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const PACKAGE = '@hatchet-dev/typescript-sdk';

// Import paths users and the serverless package rely on, with and without the `.js` suffix.
const DOCUMENTED_SUBPATHS = [
  '.',
  './v1',
  './edge',
  './edge/index.js',
  './core',
  './clients/hatchet-client',
  './v1/client/client.js',
  './protoc/dispatcher/dispatcher',
];

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const dist = join(root, 'dist');

if (!existsSync(join(dist, 'index.js'))) {
  console.error(`dist is not built: ${dist}/index.js is missing. Run \`pnpm run tsc:build\` first.`);
  process.exit(1);
}

const pkg = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
const exportsMap = pkg.exports ?? {};
const explicitSubpaths = Object.keys(exportsMap).filter((k) => !k.includes('*'));
const failures = [];

function indexDirectories(dir, prefix = '.') {
  const out = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const sub = `${prefix}/${entry.name}`;
    if (existsSync(join(dir, entry.name, 'index.js'))) out.push(sub);
    out.push(...indexDirectories(join(dir, entry.name), sub));
  }
  return out;
}

for (const sub of indexDirectories(dist)) {
  if (!(sub in exportsMap)) {
    failures.push(`${sub} ships an index.js but has no explicit "exports" entry`);
  }
}

for (const sub of explicitSubpaths) {
  const target = exportsMap[sub];
  const files = typeof target === 'string' ? [target] : Object.values(target);
  for (const file of files) {
    if (file !== './package.json' && !existsSync(join(dist, file))) {
      failures.push(`"${sub}" target ${file} does not exist in dist`);
    }
  }
}

const scratch = mkdtempSync(join(tmpdir(), 'hatchet-check-exports-'));
try {
  // The published layout: package.json next to the build output, reached through node_modules.
  cpSync(join(root, 'package.json'), join(dist, 'package.json'));
  mkdirSync(join(scratch, 'node_modules', '@hatchet-dev'), { recursive: true });
  symlinkSync(dist, join(scratch, 'node_modules', '@hatchet-dev', 'typescript-sdk'), 'dir');

  const esmDir = join(scratch, 'esm');
  const cjsDir = join(scratch, 'cjs');
  mkdirSync(esmDir);
  mkdirSync(cjsDir);
  writeFileSync(join(esmDir, 'package.json'), '{ "type": "module" }\n');
  writeFileSync(join(cjsDir, 'package.json'), '{ "type": "commonjs" }\n');
  const cjsRequire = createRequire(join(cjsDir, 'probe.js'));

  const subpaths = [...new Set([...DOCUMENTED_SUBPATHS, ...explicitSubpaths])];
  for (const [i, sub] of subpaths.entries()) {
    const specifier = sub === '.' ? PACKAGE : `${PACKAGE}/${sub.slice(2)}`;

    const probe = join(esmDir, `probe-${i}.mjs`);
    const attributes = sub === './package.json' ? ' with { type: "json" }' : '';
    writeFileSync(probe, `import ${JSON.stringify(specifier)}${attributes};\n`);
    try {
      await import(pathToFileURL(probe).href);
    } catch (e) {
      failures.push(`ESM import of ${specifier} failed: ${e.message}`);
    }

    try {
      cjsRequire(specifier);
    } catch (e) {
      failures.push(`CJS require of ${specifier} failed: ${e.message}`);
    }
  }
} finally {
  rmSync(scratch, { recursive: true, force: true });
}

if (failures.length) {
  console.error('exports map check failed:');
  for (const f of failures) console.error(`  ${f}`);
  process.exit(1);
}

console.log(
  `exports map resolves ${DOCUMENTED_SUBPATHS.length} documented and ${explicitSubpaths.length} explicit subpaths from ESM and CJS`
);
