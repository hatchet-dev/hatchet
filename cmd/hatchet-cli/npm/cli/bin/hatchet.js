#!/usr/bin/env node

const { spawnSync } = require('child_process');

// npm package names use win32/x64, not the Go GOOS/GOARCH values.
const platforms = {
  'darwin-arm64': '@hatchet-dev/cli-darwin-arm64',
  'darwin-x64': '@hatchet-dev/cli-darwin-x64',
  'linux-arm64': '@hatchet-dev/cli-linux-arm64',
  'linux-x64': '@hatchet-dev/cli-linux-x64',
  'win32-x64': '@hatchet-dev/cli-win32-x64',
};

const key = `${process.platform}-${process.arch}`;
const pkg = platforms[key];

function supportedPlatforms() {
  return Object.keys(platforms)
    .map((name) => `  ${name}`)
    .join('\n');
}

if (!pkg) {
  console.error(
    `@hatchet-dev/cli does not support ${process.platform}/${process.arch}.`,
  );
  console.error('Supported platforms:');
  console.error(supportedPlatforms());
  process.exit(1);
}

const binName = process.platform === 'win32' ? 'hatchet.exe' : 'hatchet';
let bin;
try {
  bin = require.resolve(`${pkg}/bin/${binName}`);
} catch {
  console.error(`Could not find the ${pkg} package.`);
  console.error(
    'It is an optional dependency and may have been skipped (for example by --no-optional).',
  );
  console.error('Reinstall with: npm i -g @hatchet-dev/cli');
  console.error('Supported platforms:');
  console.error(supportedPlatforms());
  process.exit(1);
}

const result = spawnSync(bin, process.argv.slice(2), { stdio: 'inherit' });
if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
if (result.signal) {
  process.kill(process.pid, result.signal);
}
process.exit(result.status === null ? 1 : result.status);
