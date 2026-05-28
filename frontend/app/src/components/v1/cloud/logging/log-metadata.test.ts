import { formatLogMetadata, hasLogMetadata } from './log-metadata';
import * as assert from 'node:assert';
import { test } from 'node:test';

test('hasLogMetadata returns true for non-empty metadata objects', () => {
  assert.strictEqual(hasLogMetadata({ chunkIndex: 16 }), true);
});

test('hasLogMetadata returns false for missing or empty metadata', () => {
  assert.strictEqual(hasLogMetadata(undefined), false);
  assert.strictEqual(hasLogMetadata({}), false);
});

test('formatLogMetadata pretty-prints metadata fields', () => {
  const formatted = formatLogMetadata({
    chunkIndex: 16,
    elapsedMs: 595,
    evt: 'staff_comp_snapshot_upsert_complete',
  });

  assert.strictEqual(
    formatted,
    '{\n  "chunkIndex": 16,\n  "elapsedMs": 595,\n  "evt": "staff_comp_snapshot_upsert_complete"\n}',
  );
});
