import { test } from 'node:test';
import assert from 'node:assert/strict';
import { formatDuration, parseDuration } from '../src/duration.ts';

test('parses hours and minutes', () => {
  assert.equal(parseDuration('1h30m'), 5_400_000);
});

test('parses seconds and days', () => {
  assert.equal(parseDuration('90s'), 90_000);
  assert.equal(parseDuration('2d'), 172_800_000);
});

test('parses milliseconds', () => {
  assert.equal(parseDuration('500ms'), 500);
  assert.equal(parseDuration('1s250ms'), 1_250);
});

test('rejects garbage', () => {
  assert.throws(() => parseDuration('soon'));
  assert.throws(() => parseDuration('10 minutes'));
});

test('round-trips through formatDuration', () => {
  for (const d of ['1h30m', '2d', '45s', '1m5s']) {
    assert.equal(formatDuration(parseDuration(d)), d);
  }
});
