import {
  getResourceLimitStatus,
  getUsageLimitStatus,
} from './resource-limit-status';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

describe('getResourceLimitStatus', () => {
  it('marks a limit exhausted when usage meets the cap', () => {
    assert.equal(
      getResourceLimitStatus({ value: 3, limitValue: 3 }),
      'exhausted',
    );
  });

  it('warns when usage meets the alarm threshold', () => {
    assert.equal(
      getResourceLimitStatus({ value: 8, alarmValue: 8, limitValue: 10 }),
      'warn',
    );
  });
});

describe('getUsageLimitStatus', () => {
  it('ignores unlimited and zero-grant features', () => {
    assert.equal(
      getUsageLimitStatus({ usage: 4, includedUsage: 1, unlimited: true }),
      'ok',
    );
    assert.equal(
      getUsageLimitStatus({ usage: 4, includedUsage: 0, unlimited: false }),
      'ok',
    );
  });

  it('marks included usage exhausted at the cap', () => {
    assert.equal(
      getUsageLimitStatus({
        usage: 5,
        includedUsage: 5,
        unlimited: false,
      }),
      'exhausted',
    );
  });

  it('warns when usage is above 75% of the included amount', () => {
    assert.equal(
      getUsageLimitStatus({
        usage: 76,
        includedUsage: 100,
        unlimited: false,
      }),
      'warn',
    );
    assert.equal(
      getUsageLimitStatus({
        usage: 75,
        includedUsage: 100,
        unlimited: false,
      }),
      'ok',
    );
  });
});
