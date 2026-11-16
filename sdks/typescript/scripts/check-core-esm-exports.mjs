#!/usr/bin/env node
// Verifies that every export of the built core entry is visible to an ESM importer under Node:
// the entry is CommonJS, and Node derives its named exports statically, so a re-export written
// in a form that detection misses (a getter over a default import) vanishes for `import { X }`.
import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';
import { resolve } from 'node:path';

const entry = resolve('dist/core/index.js');
const cjs = createRequire(import.meta.url)(entry);
const esm = await import(pathToFileURL(entry).href);
const missing = Object.keys(cjs).filter((name) => name !== '__esModule' && !(name in esm));

if (missing.length > 0) {
  console.error(`core entry exports hidden from ESM importers: ${missing.join(', ')}`);
  process.exit(1);
}

console.log(`core entry: ${Object.keys(cjs).length} exports visible to ESM importers`);
