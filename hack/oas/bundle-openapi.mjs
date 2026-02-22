#!/usr/bin/env node
/**
 * Bundles the OpenAPI spec using only @redocly/openapi-core (no Redoc UI).
 * Use this in Docker/CI to avoid pulling in styled-components/React from the CLI.
 *
 * Usage: from repo root or from a dir that has ./openapi/openapi.yaml:
 *   node bundle-openapi.mjs
 * Output: ./bin/oas/openapi.yaml
 */
import { bundle, loadConfig } from '@redocly/openapi-core';
import yaml from 'yaml';
import fs from 'fs';
import path from 'path';

const ref = './openapi/openapi.yaml';
const outPath = './bin/oas/openapi.yaml';

const config = await loadConfig();
const result = await bundle({ ref, config });
if (result.problems?.errors?.length) {
  console.error('Bundle had errors:', result.problems.errors);
  process.exit(1);
}

const outDir = path.dirname(outPath);
fs.mkdirSync(outDir, { recursive: true });
fs.writeFileSync(outPath, yaml.stringify(result.bundle.parsed), 'utf8');
console.log('Bundled to', outPath);
