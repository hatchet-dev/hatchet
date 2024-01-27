import { bumpMinorVersion } from './semver';

describe('bumpMinorVersion', () => {
  it('should increment the minor version by 1', () => {
    const version = '1.2.3';
    const expected = '1.3.0';
    const result = bumpMinorVersion(version);
    expect(result).toEqual(expected);
  });

  it('should handle a version with a single digit minor version', () => {
    const version = '2.0.0';
    const expected = '2.1.0';
    const result = bumpMinorVersion(version);
    expect(result).toEqual(expected);
  });

  it('should handle a version with a two-digit minor version', () => {
    const version = '3.10.0';
    const expected = '3.11.0';
    const result = bumpMinorVersion(version);
    expect(result).toEqual(expected);
  });

  it('should handle a version with a three-digit minor version', () => {
    const version = '4.100.0';
    const expected = '4.101.0';
    const result = bumpMinorVersion(version);
    expect(result).toEqual(expected);
  });

  it('should handle with v prefix', () => {
    const version = 'v1.2.3';
    const expected = 'v1.3.0';
    const result = bumpMinorVersion(version);
    expect(result).toEqual(expected);
  });
});
