import { escapeControls } from './escape';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

describe('escapeControls', () => {
  it('shows backspaces and ANSI escapes as text', () => {
    assert.equal(
      escapeControls('\b\b\x1b[32mHatchet support'),
      '\\x08\\x08\\x1b[32mHatchet support',
    );
  });

  it('keeps tabs, newlines and printable text', () => {
    assert.equal(escapeControls('a\tb\nc é ✓'), 'a\tb\nc é ✓');
  });

  it('escapes DEL and NUL', () => {
    assert.equal(escapeControls('\x7f\x00'), '\\x7f\\x00');
  });
});
