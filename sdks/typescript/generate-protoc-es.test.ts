import { spawnSync } from 'child_process';
import {
  chmodSync,
  cpSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';

/**
 * A repository-shaped scratch tree for the generation script: `sdks/typescript` with the
 * script, this package's `node_modules`, and an existing output directory; `api-contracts` and
 * `hack/proto/vendor` are created by each test as it needs them. The compiler is a script that
 * records its arguments and, unless told to fail, writes a marker into the output directory.
 */
function scratchRepo() {
  const root = mkdtempSync(join(tmpdir(), 'hatchet-gen-'));
  const sdk = join(root, 'sdks', 'typescript');
  mkdirSync(sdk, { recursive: true });
  cpSync(join(__dirname, 'generate-protoc-es.sh'), join(sdk, 'generate-protoc-es.sh'));
  symlinkSync(join(__dirname, 'node_modules'), join(sdk, 'node_modules'), 'dir');

  const out = join(sdk, 'src', 'protoc-es');
  mkdirSync(out, { recursive: true });
  writeFileSync(join(out, 'existing.ts'), '// committed bindings\n');

  const argvFile = join(root, 'argv.json');
  const compiler = join(root, 'fake-protoc');
  writeFileSync(
    compiler,
    [
      '#!/usr/bin/env node',
      `require('fs').writeFileSync(${JSON.stringify(argvFile)}, JSON.stringify(process.argv.slice(2)));`,
      'if (process.env.FAKE_PROTOC_FAIL) process.exit(1);',
      "const outDir = process.argv.find((a) => a.startsWith('--es_out=')).slice('--es_out='.length);",
      "require('fs').writeFileSync(require('path').join(outDir, 'generated.ts'), '// generated\\n');",
      '',
    ].join('\n')
  );
  chmodSync(compiler, 0o755);

  return {
    root,
    sdk,
    out,
    compiler,
    argv: () => JSON.parse(readFileSync(argvFile, 'utf8')) as string[],
    withInputs(protoNames: string[]) {
      const contracts = join(root, 'api-contracts');
      mkdirSync(contracts, { recursive: true });
      for (const name of protoNames) {
        writeFileSync(join(contracts, name), 'syntax = "proto3";\n');
      }
      mkdirSync(join(root, 'hack', 'proto', 'vendor'), { recursive: true });
    },
    run(env: Record<string, string> = {}) {
      return spawnSync('bash', [join(sdk, 'generate-protoc-es.sh')], {
        env: { ...process.env, PROTOC: compiler, ...env },
        encoding: 'utf8',
      });
    },
  };
}

describe('generate-protoc-es.sh', () => {
  let repo: ReturnType<typeof scratchRepo>;

  beforeEach(() => {
    repo = scratchRepo();
  });

  afterEach(() => rmSync(repo.root, { recursive: true, force: true }));

  it('passes every source as a path under the proto directory', () => {
    repo.withInputs(['ordinary.proto', '--descriptor_set_out=option-output.proto']);

    const result = repo.run();

    expect(result.status).toBe(0);
    const sources = repo.argv().filter((arg) => arg.endsWith('.proto'));
    expect(sources).toEqual([
      '../../api-contracts/--descriptor_set_out=option-output.proto',
      '../../api-contracts/ordinary.proto',
      'google/rpc/status.proto',
    ]);
  });

  it('replaces the output only after a successful generation', () => {
    repo.withInputs(['ordinary.proto']);

    expect(repo.run().status).toBe(0);

    expect(readdirSync(repo.out)).toEqual(['generated.ts']);
    expect(readdirSync(join(repo.sdk, 'src'))).toEqual(['protoc-es']);
  });

  it('keeps the existing output when the compiler fails', () => {
    repo.withInputs(['ordinary.proto']);

    expect(repo.run({ FAKE_PROTOC_FAIL: '1' }).status).not.toBe(0);

    expect(readdirSync(repo.out)).toEqual(['existing.ts']);
    expect(readdirSync(join(repo.sdk, 'src'))).toEqual(['protoc-es']);
  });

  it('keeps the existing output when the proto directory is missing', () => {
    const result = repo.run();

    expect(result.status).not.toBe(0);
    expect(result.stderr).toMatch(/proto source directory not found/);
    expect(readdirSync(repo.out)).toEqual(['existing.ts']);
  });

  it('keeps the existing output when the compiler is missing', () => {
    repo.withInputs(['ordinary.proto']);

    const result = repo.run({ PROTOC: join(repo.root, 'no-such-protoc') });

    expect(result.status).not.toBe(0);
    expect(result.stderr).toMatch(/protoc not found/);
    expect(readdirSync(repo.out)).toEqual(['existing.ts']);
  });

  it('refuses a proto file name containing a newline', () => {
    repo.withInputs(['ordinary.proto', 'line\nbreak.proto']);

    const result = repo.run();

    expect(result.status).not.toBe(0);
    expect(result.stderr).toMatch(/contains a newline/);
    expect(existsSync(join(repo.out, 'existing.ts'))).toBe(true);
  });
});
