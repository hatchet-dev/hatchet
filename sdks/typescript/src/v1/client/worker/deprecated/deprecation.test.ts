import { parseSemver, semverLessThan } from './deprecation';

describe('parseSemver', () => {
  it('parses a standard version with v prefix', () => {
    expect(parseSemver('v0.78.23')).toEqual([0, 78, 23]);
  });

  it('parses a version without v prefix', () => {
    expect(parseSemver('1.2.3')).toEqual([1, 2, 3]);
  });

  it('strips pre-release suffix', () => {
    expect(parseSemver('v0.1.0-alpha.0')).toEqual([0, 1, 0]);
    expect(parseSemver('v10.20.30-rc.1')).toEqual([10, 20, 30]);
  });

  it('returns [0,0,0] for empty string', () => {
    expect(parseSemver('')).toEqual([0, 0, 0]);
  });

  it('returns [0,0,0] for malformed input', () => {
    expect(parseSemver('v1.2')).toEqual([0, 0, 0]);
    expect(parseSemver('not-a-version')).toEqual([0, 0, 0]);
  });
});

describe('semverLessThan', () => {
  it('returns true when a < b (patch)', () => {
    expect(semverLessThan('v0.78.22', 'v0.78.23')).toBe(true);
  });

  it('returns false when a == b', () => {
    expect(semverLessThan('v0.78.23', 'v0.78.23')).toBe(false);
  });

  it('returns false when a > b (patch)', () => {
    expect(semverLessThan('v0.78.24', 'v0.78.23')).toBe(false);
  });

  it('compares minor versions correctly', () => {
    expect(semverLessThan('v0.77.99', 'v0.78.0')).toBe(true);
    expect(semverLessThan('v0.79.0', 'v0.78.99')).toBe(false);
  });

  it('compares major versions correctly', () => {
    expect(semverLessThan('v0.78.23', 'v1.0.0')).toBe(true);
    expect(semverLessThan('v1.0.0', 'v0.99.99')).toBe(false);
  });

  it('handles pre-release versions', () => {
    expect(semverLessThan('v0.1.0-alpha.0', 'v0.78.23')).toBe(true);
  });

  it('treats empty string as 0.0.0', () => {
    expect(semverLessThan('', 'v0.78.23')).toBe(true);
    expect(semverLessThan('v0.78.23', '')).toBe(false);
  });
});
