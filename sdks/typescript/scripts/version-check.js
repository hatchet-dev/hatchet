/* eslint-disable no-console */
const fs = require('fs');
const path = require('path');
const semver = require('semver');

const WARNINGS = {
  '1.4.0':
    'Breaking Changes in v1.4.0: This release fixes a critical bug which makes the runNoWait methods async. You will need to await this method to access the runRef.',
};

try {
  // Get the current package version
  // eslint-disable-next-line global-require
  const currentVersion = require('../package.json').version;

  // Look for the package.json in various possible locations
  const possiblePaths = [
    // npm
    path.join(process.cwd(), 'package.json'),
    // pnpm
    path.join(process.cwd(), '..', 'package.json'),
    // yarn
    path.join(process.cwd(), '..', '..', 'package.json'),
    // monorepo setup
    path.join(process.cwd(), '..', '..', '..', 'package.json'),
  ];

  let parentPackagePath = null;
  for (const possiblePath of possiblePaths) {
    if (fs.existsSync(possiblePath)) {
      parentPackagePath = possiblePath;
      break;
    }
  }

  if (parentPackagePath) {
    const parentPackage = JSON.parse(fs.readFileSync(parentPackagePath, 'utf8'));
    const dependencies = {
      ...parentPackage.dependencies,
      ...parentPackage.devDependencies,
    };

    const installedVersion = dependencies['@hatchet-dev/typescript-sdk'];

    // If there's no installed version, this is a first-time install
    if (!installedVersion) {
      // Show all warnings for the current version
      for (const [version, warning] of Object.entries(WARNINGS)) {
        if (semver.gte(currentVersion, version)) {
          console.warn('\x1b[33m%s\x1b[0m', warning);
        }
      }
    } else {
      // Check for specific version warnings
      for (const [version, warning] of Object.entries(WARNINGS)) {
        if (semver.gte(currentVersion, version) && semver.lt(installedVersion, version)) {
          console.warn('\x1b[33m%s\x1b[0m', warning);
        }
      }
    }
  }
} catch (error) {
  // Silently fail - this is just a warning system
  // console.error(error);
}
