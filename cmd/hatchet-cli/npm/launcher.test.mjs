import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';

function launch(t, source, args = []) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'hatchet-launcher-'));
  const binDir = path.join(dir, 'node_modules', '@hatchet-dev',
    `cli-${process.platform}-${process.arch}`, 'bin');
  fs.mkdirSync(binDir, { recursive: true });
  fs.copyFileSync(new URL('./cli/bin/hatchet.js', import.meta.url), path.join(dir, 'hatchet.cjs'));
  fs.writeFileSync(path.join(binDir, 'hatchet'), `#!/usr/bin/env node\n${source}`, { mode: 0o755 });
  const child = spawn(process.execPath, [path.join(dir, 'hatchet.cjs'), ...args]);
  t.after(() => {
    child.kill('SIGKILL');
    fs.rmSync(dir, { recursive: true, force: true });
  });
  return child;
}

const unix = { skip: process.platform === 'win32', timeout: 10000 };

test('passes arguments and preserves exit status', unix, async (t) => {
  const child = launch(t, 'console.log(JSON.stringify(process.argv.slice(2))); process.exit(23);', ['a b', '--version']);
  let output = '';
  child.stdout.on('data', (data) => { output += data; });
  assert.deepEqual(await once(child, 'close'), [23, null]);
  assert.deepEqual(JSON.parse(output), ['a b', '--version']);
});

for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP']) {
  test(`forwards ${signal} and waits for child cleanup`, unix, async (t) => {
    const child = launch(t, `
      process.on('${signal}', () => {
        setTimeout(() => { console.log('cleaned up'); process.exit(0); }, 50);
      });
      console.log(process.pid);
      setInterval(() => {}, 1000);
    `);
    let output = '';
    child.stdout.on('data', (data) => { output += data; });
    const closed = once(child, 'close');
    await once(child.stdout, 'data');
    const pid = Number(output.trim());
    t.after(() => { try { process.kill(pid, 'SIGKILL'); } catch {} });
    child.kill(signal);
    assert.deepEqual(await closed, [0, null]);
    assert.match(output, /cleaned up/);
    assert.throws(() => process.kill(pid, 0), { code: 'ESRCH' });
  });
}

test('mirrors child signal termination', unix, async (t) => {
  const child = launch(t, "process.kill(process.pid, 'SIGTERM');");
  assert.deepEqual(await once(child, 'close'), [null, 'SIGTERM']);
});
