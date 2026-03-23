import { durationToString, durationToMs, Duration } from './duration';

describe('durationToString', () => {
  it('passes through a duration string as-is', () => {
    expect(durationToString('1h30m5s')).toBe('1h30m5s');
    expect(durationToString('10m')).toBe('10m');
    expect(durationToString('30s')).toBe('30s');
  });

  it('converts a DurationObject to a string', () => {
    expect(durationToString({ hours: 1, minutes: 30, seconds: 5 })).toBe('1h30m5s');
    expect(durationToString({ minutes: 10 })).toBe('10m');
    expect(durationToString({ seconds: 45 })).toBe('45s');
    expect(durationToString({ hours: 2 })).toBe('2h');
  });

  it('returns "0s" for an empty DurationObject', () => {
    expect(durationToString({})).toBe('0s');
  });

  it('converts milliseconds to a string', () => {
    expect(durationToString(0)).toBe('0s');
    expect(durationToString(5000)).toBe('5s');
    expect(durationToString(60_000)).toBe('1m');
    expect(durationToString(3_600_000)).toBe('1h');
    expect(durationToString(5_405_000)).toBe('1h30m5s');
  });

  it('truncates sub-second remainders from milliseconds', () => {
    expect(durationToString(1500)).toBe('1s');
    expect(durationToString(999)).toBe('0s');
  });
});

describe('durationToMs', () => {
  it('parses a seconds-only string', () => {
    expect(durationToMs('30s')).toBe(30_000);
  });

  it('parses a minutes-only string', () => {
    expect(durationToMs('10m')).toBe(600_000);
  });

  it('parses an hours-only string', () => {
    expect(durationToMs('2h')).toBe(7_200_000);
  });

  it('parses a multi-unit string', () => {
    expect(durationToMs('1h30m5s')).toBe(5_405_000);
  });

  it('converts a DurationObject', () => {
    expect(durationToMs({ hours: 1, minutes: 30, seconds: 5 })).toBe(5_405_000);
    expect(durationToMs({ seconds: 10 })).toBe(10_000);
    expect(durationToMs({})).toBe(0);
  });

  it('returns a number (milliseconds) as-is', () => {
    expect(durationToMs(42_000)).toBe(42_000);
    expect(durationToMs(0)).toBe(0);
  });

  it('throws on an invalid string', () => {
    expect(() => durationToMs('bad' as Duration)).toThrow(/Invalid duration string/);
  });
});

describe('round-trip: durationToMs → durationToString', () => {
  const cases: [Duration, number, string][] = [
    ['1h30m5s', 5_405_000, '1h30m5s'],
    ['10m', 600_000, '10m'],
    ['30s', 30_000, '30s'],
    [{ hours: 2, minutes: 15 }, 8_100_000, '2h15m'],
    [60_000, 60_000, '1m'],
  ];

  it.each(cases)('input %j → %d ms → "%s"', (input, expectedMs, expectedStr) => {
    const ms = durationToMs(input);
    expect(ms).toBe(expectedMs);
    expect(durationToString(ms)).toBe(expectedStr);
  });
});
