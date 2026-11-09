import { formatErrorForDisplay } from './utils';
import assert from 'node:assert/strict';
import { test } from 'node:test';

test('a JSON error is pretty-printed with real line breaks in its stack', () => {
  const error = JSON.stringify({
    message: 'boom',
    stack: 'Error: boom\n    at run (worker.ts:1:1)',
  });

  assert.deepEqual(formatErrorForDisplay(error), {
    language: 'json',
    code: [
      '{',
      '  "message": "boom",',
      '  "stack": "Error: boom',
      '    at run (worker.ts:1:1)"',
      '}',
    ].join('\n'),
  });
});

test('an escaped backslash before n is not treated as a line break', () => {
  const error = JSON.stringify({ path: 'C:\\new\\file' });

  assert.equal(
    formatErrorForDisplay(error).code,
    '{\n  "path": "C:\\\\new\\\\file"\n}',
  );
});

test('a plain-text error is returned unchanged', () => {
  const error = 'Traceback (most recent call last):\n  File "worker.py"';

  assert.deepEqual(formatErrorForDisplay(error), {
    language: 'text',
    code: error,
  });
});

test('a JSON primitive is treated as plain text', () => {
  assert.deepEqual(formatErrorForDisplay('"timed out"'), {
    language: 'text',
    code: '"timed out"',
  });
});
