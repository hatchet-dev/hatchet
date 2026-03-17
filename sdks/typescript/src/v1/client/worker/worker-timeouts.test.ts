import { resolveExecutionTimeout, resolveScheduleTimeout } from './worker-internal';

describe('resolveExecutionTimeout', () => {
  it('uses task.executionTimeout when set', () => {
    expect(resolveExecutionTimeout({ executionTimeout: '30s' })).toBe('30s');
  });

  it('falls back to deprecated task.timeout', () => {
    expect(resolveExecutionTimeout({ timeout: '45s' })).toBe('45s');
  });

  it('prefers executionTimeout over deprecated timeout', () => {
    expect(resolveExecutionTimeout({ executionTimeout: '30s', timeout: '45s' })).toBe('30s');
  });

  it('falls back to workflow taskDefaults.executionTimeout', () => {
    expect(resolveExecutionTimeout({}, { executionTimeout: '2m' })).toBe('2m');
  });

  it('task-level timeout beats workflow defaults', () => {
    expect(resolveExecutionTimeout({ timeout: '45s' }, { executionTimeout: '2m' })).toBe('45s');
  });

  it('task-level executionTimeout beats workflow defaults', () => {
    expect(resolveExecutionTimeout({ executionTimeout: '30s' }, { executionTimeout: '2m' })).toBe(
      '30s'
    );
  });

  it('defaults to 60s when nothing is set', () => {
    expect(resolveExecutionTimeout({})).toBe('60s');
  });

  it('defaults to 60s when workflow defaults are empty', () => {
    expect(resolveExecutionTimeout({}, {})).toBe('60s');
  });

  it('handles DurationObject for executionTimeout', () => {
    expect(resolveExecutionTimeout({ executionTimeout: { minutes: 5 } })).toBe('5m');
  });

  it('handles DurationObject for deprecated timeout', () => {
    expect(resolveExecutionTimeout({ timeout: { hours: 1, minutes: 30 } })).toBe('1h30m');
  });

  it('handles DurationObject in workflow defaults', () => {
    expect(resolveExecutionTimeout({}, { executionTimeout: { seconds: 90 } })).toBe('90s');
  });
});

describe('resolveScheduleTimeout', () => {
  it('uses task.scheduleTimeout when set', () => {
    expect(resolveScheduleTimeout({ scheduleTimeout: '10m' })).toBe('10m');
  });

  it('falls back to workflow taskDefaults.scheduleTimeout', () => {
    expect(resolveScheduleTimeout({}, { scheduleTimeout: '15m' })).toBe('15m');
  });

  it('task-level beats workflow defaults', () => {
    expect(resolveScheduleTimeout({ scheduleTimeout: '10m' }, { scheduleTimeout: '15m' })).toBe(
      '10m'
    );
  });

  it('returns undefined when nothing is set', () => {
    expect(resolveScheduleTimeout({})).toBeUndefined();
  });

  it('returns undefined when workflow defaults are empty', () => {
    expect(resolveScheduleTimeout({}, {})).toBeUndefined();
  });

  it('handles DurationObject for scheduleTimeout', () => {
    expect(resolveScheduleTimeout({ scheduleTimeout: { minutes: 10 } })).toBe('10m');
  });
});
