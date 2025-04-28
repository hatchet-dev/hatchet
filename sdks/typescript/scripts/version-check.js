const fs = require('fs');
const path = require('path');
const semver = require('semver');

// Get the current package version
const currentVersion = require('../package.json').version;

// Try to find the previous version from node_modules
const previousVersionPath = path.join(
  process.cwd(),
  'node_modules',
  '@hatchet-dev',
  'typescript-sdk',
  'package.json'
);
let previousVersion = null;

try {
  if (fs.existsSync(previousVersionPath)) {
    previousVersion = require(previousVersionPath).version;
  }
} catch (error) {
  // If we can't read the previous version, that's okay
}

// If we have a previous version and it's less than 1.4.0, show the warning
if (previousVersion && semver.lt(previousVersion, '1.4.0') && semver.gte(currentVersion, '1.4.0')) {
  console.warn(
    '\x1b[33m%s\x1b[0m',
    'WARNING: This release fixes a critical bug which makes the runNoWait methods async. You will need to await this method to access the runRef.'
  );
}
