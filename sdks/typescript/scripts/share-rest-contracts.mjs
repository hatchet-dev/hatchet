#!/usr/bin/env node
/* eslint-disable no-console */
// Points the fetch REST client at the axios client's data contracts. swagger-typescript-api
// writes a `data-contracts.ts` next to every client it generates, and the two copies would be
// distinct declarations: a string enum from one is not assignable to the same enum from the
// other, so a caller mixing `V1TaskStatus` from the core entry with a status returned by the
// fetch client would not type-check. The axios client's copy is the one every type import in
// the SDK names, so the fetch client's copy is dropped and its import rewritten.
//
// Run by `pnpm run generate-api-fetch`, right after the generator.
import { readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const generated = join(root, 'src', 'clients', 'rest', 'generated-fetch');
const api = join(generated, 'Api.ts');

const contractsImport = /from (['"])\.\/data-contracts\1/g;
const source = readFileSync(api, 'utf8');
if (!contractsImport.test(source)) {
  console.error(`${api} does not import ./data-contracts; the generator's output changed shape`);
  process.exit(1);
}

writeFileSync(api, source.replace(contractsImport, "from '../generated/data-contracts'"));
rmSync(join(generated, 'data-contracts.ts'), { force: true });
console.log('generated-fetch/Api.ts imports ../generated/data-contracts');
