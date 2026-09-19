#!/usr/bin/env node

// Verifies that the runtime entries of the package (`.` and `./cloudflare`) import nothing
// from Node and nothing from the SDK outside its edge entry. Bundles each built entry for a
// workerd-like browser target, the way wrangler does, and fails on any `node:` specifier,
// Node builtin or non-edge SDK module reached transitively.
//
// Run after `pnpm run build` (the `check:edge` script does both). The linked SDK is bundled
// from its dist so the check covers what the SDK's edge entry reaches too.
import { builtinModules } from 'node:module';
import { existsSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { build } from 'esbuild';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const entries = process.argv.slice(2);

if (entries.length === 0) {
  entries.push('dist/index.js', 'dist/adapters/cloudflare.js');
}

// Optional peers the SDK's declaration classes load lazily (`mcpTool()`); never reached on a
// serverless code path.
const lazyOptionalPeers = [
  '@openai/agents',
  '@anthropic-ai/claude-agent-sdk',
  '@modelcontextprotocol/sdk',
];

const builtins = new Set(builtinModules.flatMap((m) => [m, `node:${m}`]));

function sdkModuleOf(path) {
  const m = path.match(/@hatchet-dev\/typescript-sdk\/(.+)$/);
  return m ? m[1] : undefined;
}

async function check(entry) {
  const entryPath = resolve(root, entry);

  if (!existsSync(entryPath)) {
    console.error(`entry not built: ${entryPath} is missing. Run \`pnpm run build\` first.`);
    return false;
  }

  const violations = new Map();

  const record = (specifier, importer) => {
    const from = importer ? importer.replace(`${root}/`, '') : '<entry>';
    if (!violations.has(specifier)) violations.set(specifier, new Set());
    violations.get(specifier).add(from);
  };

  const detector = {
    name: 'edge-violation-detector',
    setup(pluginBuild) {
      pluginBuild.onResolve({ filter: /.*/ }, (args) => {
        const isBuiltin = args.path.startsWith('node:') || builtins.has(args.path);
        const sdkModule = sdkModuleOf(args.path);
        const isNonEdgeSdk =
          args.path === '@hatchet-dev/typescript-sdk' ||
          (sdkModule !== undefined && !sdkModule.startsWith('edge'));

        if (!isBuiltin && !isNonEdgeSdk) return undefined;

        record(args.path, args.importer);
        // Resolve to an empty stub so the bundle continues and every violation is collected.
        return { path: args.path, namespace: 'edge-violation-stub' };
      });
      pluginBuild.onLoad({ filter: /.*/, namespace: 'edge-violation-stub' }, () => ({
        contents: 'export default {};',
        loader: 'js',
      }));
    },
  };

  const result = await build({
    entryPoints: [entryPath],
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
    plugins: [detector],
  });

  const bundled = Object.keys(result.metafile.inputs);
  const packages = [
    ...new Set(
      bundled
        .filter((f) => f.includes('node_modules/'))
        .map((f) => {
          const m = f.match(/node_modules\/(?:\.pnpm\/[^/]+\/node_modules\/)?((?:@[^/]+\/)?[^/]+)/);
          return m ? m[1] : f;
        })
    ),
  ].sort();

  console.log(`${entry}: bundled ${bundled.length} module(s); packages: ${packages.join(', ')}`);

  if (violations.size > 0) {
    console.error(`\n${entry} reaches Node builtins or the SDK's non-edge entry:`);
    for (const [specifier, importers] of [...violations.entries()].sort()) {
      console.error(`  ${specifier}`);
      for (const importer of [...importers].sort()) console.error(`    from ${importer}`);
    }
    return false;
  }

  return true;
}

let ok = true;

for (const entry of entries) {
  ok = (await check(entry)) && ok;
}

if (!ok) {
  process.exit(1);
}

console.log('runtime entries are free of Node builtins and of the SDK non-edge entry');
