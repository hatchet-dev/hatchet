export function bumpMinorVersion(version: string): string {
  const startsWithV = version.startsWith('v');

  if (startsWithV) {
    // eslint-disable-next-line no-param-reassign
    version = version.slice(1);
  }

  const parts = version.split('.');
  if (parts.length !== 3) {
    throw new Error(`Invalid semantic version: ${version}`);
  }

  const [major, minor] = parts.map(Number);

  const newMinor = minor + 1;
  const newVersion = `${major}.${newMinor}.0`;

  if (startsWithV) {
    return `v${newVersion}`;
  }
  return newVersion;
}
