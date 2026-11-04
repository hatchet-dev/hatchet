#!/usr/bin/env node
// Assembles the @hatchet-dev/cli wrapper and one package per platform from a
// GoReleaser dist directory. Versions are filled in here so the templates in
// this directory are not published themselves.
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(scriptDir, '../../..');

// npm os/cpu differ from Go's GOOS/GOARCH: windows is win32, amd64 is x64.
const platforms = [
  { goos: 'darwin', goarch: 'arm64', os: 'darwin', cpu: 'arm64', name: 'cli-darwin-arm64' },
  { goos: 'darwin', goarch: 'amd64', os: 'darwin', cpu: 'x64', name: 'cli-darwin-x64' },
  { goos: 'linux', goarch: 'arm64', os: 'linux', cpu: 'arm64', name: 'cli-linux-arm64' },
  { goos: 'linux', goarch: 'amd64', os: 'linux', cpu: 'x64', name: 'cli-linux-x64' },
  { goos: 'windows', goarch: 'amd64', os: 'win32', cpu: 'x64', name: 'cli-win32-x64' },
];

const versionPattern =
  /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/;

function fail(message) {
  console.error(message);
  process.exit(1);
}

function usage() {
  fail('usage: node build.mjs <version> <goreleaser-dist> [out-dir]');
}

// GoReleaser records absolute paths from the build machine. After the dist
// directory is downloaded on another runner those paths do not exist, so fall
// back to the path relative to the dist directory.
function locateBinary(artifact, distDir) {
  const recorded = String(artifact.path ?? '').replaceAll('\\', '/');
  if (recorded && fs.existsSync(recorded)) {
    return recorded;
  }
  // GoReleaser stores paths relative to its workdir ("dist/...") or absolute
  // paths from the build machine. The publish job downloads that directory
  // under another name, so keep only the part inside dist.
  let rel = recorded;
  const base = path.basename(distDir);
  if (rel.startsWith(`${base}/`)) {
    rel = rel.slice(base.length + 1);
  } else if (rel.startsWith('dist/')) {
    rel = rel.slice('dist/'.length);
  } else {
    const idx = rel.lastIndexOf('/dist/');
    if (idx >= 0) {
      rel = rel.slice(idx + '/dist/'.length);
    }
  }
  const candidate = path.join(distDir, rel);
  if (rel && fs.existsSync(candidate)) {
    return candidate;
  }
  fail(
    `binary for ${artifact.goos}/${artifact.goarch} not found (artifact path ${artifact.path})`,
  );
}

function isHatchetBinary(artifact) {
  if (artifact.type !== 'Binary') {
    return false;
  }
  const extra = artifact.extra ?? {};
  return (
    extra.ID === 'hatchet' ||
    extra.Binary === 'hatchet' ||
    artifact.name === 'hatchet' ||
    artifact.name === 'hatchet.exe'
  );
}

const versionArg = process.argv[2];
const distDir = process.argv[3];
const outDir = path.resolve(process.argv[4] ?? path.join(scriptDir, 'npm-out'));

if (!versionArg || !distDir) {
  usage();
}

const version = versionArg.replace(/^v/, '');
if (!versionPattern.test(version)) {
  fail(`invalid npm version ${JSON.stringify(versionArg)}`);
}

const dist = path.resolve(distDir);
const artifactsPath = path.join(dist, 'artifacts.json');
if (!fs.existsSync(artifactsPath)) {
  fail(`no artifacts.json in ${dist}`);
}

const artifacts = JSON.parse(fs.readFileSync(artifactsPath, 'utf8'));
const binaries = artifacts.filter(isHatchetBinary);
const byTarget = new Map();

for (const artifact of binaries) {
  const key = `${artifact.goos}/${artifact.goarch}`;
  if (byTarget.has(key)) {
    fail(`multiple binaries for ${key} in ${artifactsPath}`);
  }
  byTarget.set(key, artifact);
}

for (const spec of platforms) {
  if (!byTarget.has(`${spec.goos}/${spec.goarch}`)) {
    fail(`missing ${spec.goos}/${spec.goarch} binary in ${artifactsPath}`);
  }
}

for (const key of byTarget.keys()) {
  const [goos, goarch] = key.split('/');
  if (!platforms.some((spec) => spec.goos === goos && spec.goarch === goarch)) {
    fail(`no npm package mapping for ${key}; add it to build.mjs`);
  }
}

fs.rmSync(outDir, { recursive: true, force: true });
fs.mkdirSync(outDir, { recursive: true });

const license = fs.readFileSync(path.join(repoRoot, 'LICENSE'));
const template = fs.readFileSync(path.join(scriptDir, 'platform/package.json.tmpl'), 'utf8');

for (const spec of platforms) {
  const artifact = byTarget.get(`${spec.goos}/${spec.goarch}`);
  const binName = spec.os === 'win32' ? 'hatchet.exe' : 'hatchet';
  const pkgDir = path.join(outDir, spec.name);
  const binDir = path.join(pkgDir, 'bin');
  fs.mkdirSync(binDir, { recursive: true });
  fs.copyFileSync(locateBinary(artifact, dist), path.join(binDir, binName));
  fs.chmodSync(path.join(binDir, binName), 0o755);
  fs.writeFileSync(path.join(pkgDir, 'LICENSE'), license);

  const description = `Hatchet CLI binary for ${spec.os} ${spec.cpu}`;
  const rendered = template
    .replaceAll('__NAME__', spec.name)
    .replaceAll('__VERSION__', version)
    .replaceAll('__DESCRIPTION__', description)
    .replaceAll('__OS__', spec.os)
    .replaceAll('__CPU__', spec.cpu);
  JSON.parse(rendered);
  fs.writeFileSync(path.join(pkgDir, 'package.json'), rendered.endsWith('\n') ? rendered : `${rendered}\n`);
  fs.writeFileSync(
    path.join(pkgDir, 'README.md'),
    `# @hatchet-dev/${spec.name}\n\nPlatform-specific binary for the [Hatchet CLI](https://www.npmjs.com/package/@hatchet-dev/cli). Install \`@hatchet-dev/cli\` instead of this package.\n`,
  );
}

const wrapper = JSON.parse(fs.readFileSync(path.join(scriptDir, 'cli/package.json'), 'utf8'));
wrapper.version = version;
const expectedDeps = platforms.map((spec) => `@hatchet-dev/${spec.name}`);
const actualDeps = Object.keys(wrapper.optionalDependencies ?? {}).sort();
if (actualDeps.join() !== [...expectedDeps].sort().join()) {
  fail(
    `wrapper optionalDependencies must be ${expectedDeps.join(', ')}, got ${actualDeps.join(', ')}`,
  );
}
for (const dep of expectedDeps) {
  wrapper.optionalDependencies[dep] = version;
}

const wrapperDir = path.join(outDir, 'cli');
fs.mkdirSync(path.join(wrapperDir, 'bin'), { recursive: true });
fs.copyFileSync(path.join(scriptDir, 'cli/bin/hatchet.js'), path.join(wrapperDir, 'bin/hatchet.js'));
fs.chmodSync(path.join(wrapperDir, 'bin/hatchet.js'), 0o755);
fs.copyFileSync(path.join(scriptDir, 'cli/README.md'), path.join(wrapperDir, 'README.md'));
fs.writeFileSync(path.join(wrapperDir, 'LICENSE'), license);
fs.writeFileSync(path.join(wrapperDir, 'package.json'), `${JSON.stringify(wrapper, null, 2)}\n`);

console.log(`assembled @hatchet-dev/cli@${version} in ${outDir}`);
