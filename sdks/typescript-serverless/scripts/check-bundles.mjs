#!/usr/bin/env node

// Verifies that what one published entry records about a declaration is seen by the others.
// Each entry is bundled on its own (`splitting: false` in tsup.config.ts), so state kept in a
// module would exist once per entry, and a task declared through `.` would look different to
// `./testing` or an adapter. Declares a pure task through the root entry and checks that the
// healthcheck built by the testing entry delivers it over a POST rather than the socket.
//
// Run after `pnpm run build` (the `check:edge` script does both).
import { resolve, dirname } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const entry = (path) => import(pathToFileURL(resolve(root, path)).href);

const { hatchet } = await entry('dist/index.js');
const { createTestOperator } = await entry('dist/testing/index.js');

const pure = hatchet.task({ name: 'pure', hatchetClient: false, fn: async (input) => input });
const social = hatchet.task({ name: 'social', fn: async (input) => input });
const operator = createTestOperator({
  workflows: [pure, social],
  secret: 'check-bundles-signing-secret-0123456789',
});

const healthcheck = await operator.healthcheck();
const socketTasks = healthcheck.tasks.map((task) => task.action);

if (socketTasks.includes('pure:pure') || !socketTasks.includes('social:social')) {
  console.error(
    `the pure-task mark is not shared across the published entries: the healthcheck lists ${JSON.stringify(socketTasks)} as socket tasks`
  );
  process.exit(1);
}

console.log('bundle check: the pure-task mark is shared across the published entries');
