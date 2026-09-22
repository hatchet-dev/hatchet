#!/usr/bin/env node
/* eslint-disable no-console */
// Runs the core client scenario (scenario.mjs) against a live engine from the built package,
// the way a user's project imports it: a scratch package whose node_modules links
// @hatchet-dev/typescript-sdk to dist/ (as scripts/check-exports.mjs does) and undici to the
// SDK's dev dependency. Prints the scenario's result as one JSON line on stdout.
//
// Run after `pnpm run tsc:build`, with HATCHET_CLIENT_TOKEN and HATCHET_CLIENT_HOST_PORT set:
//   node e2e-core/run.mjs
// pkg/testing/e2e/tscore drives it from `go test -tags e2e` against the harness engine.
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
