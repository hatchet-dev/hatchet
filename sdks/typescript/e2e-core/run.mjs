#!/usr/bin/env node
/* eslint-disable no-console */
// The scenario (scenario.mjs) must import the SDK the way a user's project does, through the
// package's exports map from a real node_modules, so that a broken `/core` or `/edge` entry
// fails here and not only in a user's bundle. A scratch package linking dist/ provides that
// (as scripts/check-exports.mjs does); undici is linked from the SDK's dev dependencies since
// the scenario needs an HTTP/1.1-only agent. The result is the last stdout line, as JSON,
// which is what pkg/testing/e2e/tscore parses.
//
// Run after `pnpm run tsc:build`, with HATCHET_CLIENT_TOKEN and HATCHET_CLIENT_HOST_PORT set:
//   node e2e-core/run.mjs
import { cpSync, existsSync, mkdirSync, mkdtempSync, rmSync, symlinkSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const sdk = resolve(here, '..');
const dist = join(sdk, 'dist');

if (!existsSync(join(dist, 'core', 'index.js'))) {
  console.error(`dist is not built: ${dist}/core/index.js is missing. Run \`pnpm run tsc:build\` first.`);
  process.exit(1);
}

const scratch = mkdtempSync(join(tmpdir(), 'hatchet-e2e-core-'));
try {
  cpSync(join(sdk, 'package.json'), join(dist, 'package.json'));
  mkdirSync(join(scratch, 'node_modules', '@hatchet-dev'), { recursive: true });
  symlinkSync(dist, join(scratch, 'node_modules', '@hatchet-dev', 'typescript-sdk'), 'dir');
  symlinkSync(join(sdk, 'node_modules', 'undici'), join(scratch, 'node_modules', 'undici'), 'dir');
  writeFileSync(join(scratch, 'package.json'), '{ "type": "module" }\n');
  cpSync(join(here, 'scenario.mjs'), join(scratch, 'scenario.mjs'));

  const { run } = await import(pathToFileURL(join(scratch, 'scenario.mjs')).href);
  const result = await run(process.env);
  console.log(JSON.stringify(result));
} finally {
  rmSync(scratch, { recursive: true, force: true });
}
