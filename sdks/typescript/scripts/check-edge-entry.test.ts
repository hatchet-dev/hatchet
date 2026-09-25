import { spawnSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const script = resolve(__dirname, 'check-edge-entry.mjs');

/** Runs the checker on a module holding `source`, as if it were an SDK module in the bundle. */
function check(source: string): { status: number | null; stderr: string } {
  const dir = mkdtempSync(join(tmpdir(), 'check-edge-entry-'));
  try {
    const entry = join(dir, 'entry.js');
    writeFileSync(entry, source);
    const result = spawnSync(process.execPath, [script, entry], { encoding: 'utf8' });
    return { status: result.status, stderr: result.stderr };
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
}

describe('check-edge-entry', () => {
  it('passes a module that uses none of the Node-only globals', () => {
    const result = check(`
      // setImmediate(() => {}) in a comment is fine, as is AbortSignal.any in prose.
      /* process.exit() described here is fine too */
      export const wait = (ms) => new Promise((r) => setTimeout(r, ms));
      export const url = 'https://example.com//path';
    `);
    expect(result.stderr).toBe('');
    expect(result.status).toBe(0);
  });

  it.each([
    ['setImmediate', 'export const tick = () => setImmediate(() => {});', /setImmediate/],
    ['process.nextTick', 'export const tick = () => process.nextTick(() => {});', /process or Buffer/],
    ['Buffer', "export const bytes = Buffer.from('x');", /process or Buffer/],
    ['.unref(', 'export const timer = setTimeout(() => {}, 1); timer.unref();', /\.unref\(\)/],
    ['AbortSignal.timeout', 'export const signal = AbortSignal.timeout(5);', /AbortSignal\.timeout or AbortSignal\.any/],
    ['AbortSignal.any', 'export const signal = AbortSignal.any([]);', /AbortSignal\.timeout or AbortSignal\.any/],
  ])('fails a module that uses %s', (_name, source, message) => {
    const result = check(source);
    expect(result.status).toBe(1);
    expect(result.stderr).toMatch(message);
  });

  it.each([
    ['typeof setImmediate', "export const tick = typeof setImmediate === 'function' ? (f) => setImmediate(f) : (f) => setTimeout(f, 0);"],
    ['an optional .unref?.()', 'export const timer = setTimeout(() => {}, 1); timer.unref?.();'],
    ['typeof AbortSignal.any', "export const has = typeof AbortSignal.any === 'function';"],
    ['globalThis.Buffer', "export const bytes = globalThis.Buffer ? globalThis.Buffer.from('x') : new Uint8Array();"],
  ])('allows a feature-detected %s', (_name, source) => {
    const result = check(source);
    expect(result.stderr).toBe('');
    expect(result.status).toBe(0);
  });
});
