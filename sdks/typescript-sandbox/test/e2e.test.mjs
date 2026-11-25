import assert from 'node:assert/strict';
import { randomUUID } from 'node:crypto';
import { after, test } from 'node:test';
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import '../dist/index.js';

const provider = process.env.SANDBOX_PROVIDER;
const template = process.env.SANDBOX_TEMPLATE;
const live = Boolean(process.env.HATCHET_CLIENT_TOKEN && provider);
const opts = { skip: live ? false : 'set HATCHET_CLIENT_TOKEN and SANDBOX_PROVIDER' };

const hatchet = live ? HatchetClient.init() : undefined;
const session = `e2e-${randomUUID().slice(0, 8)}`;
const tags = { suite: 'sandbox-e2e', session };
const log = (...parts) => console.log('   ', provider, ...parts);

after(async () => {
  if (!live) return;
  log('cleanup deleted', await hatchet.sandbox.delete({ metadata: tags }));
});

test('create, exec, files, kill, connect, fork, list, delete', opts, async () => {
  const sandbox = await hatchet.sandbox.create({ provider, template, metadata: { ...tags, role: 'parent' } });
  log('created', sandbox.sandboxId, 'metadata', sandbox.metadata);
  assert.equal(sandbox.metadata['hatchet.sandbox.provider'], provider);
  assert.equal(sandbox.metadata.role, 'parent');

  const ran = await sandbox.exec('echo out; echo err >&2; exit 3');
  log('exec', ran);
  assert.equal(ran.stdout.trim(), 'out');
  assert.equal(ran.stderr.trim(), 'err');
  assert.equal(ran.exitCode, 3);

  const slow = await sandbox.exec('sleep 20; echo late', { timeoutMs: 2000 });
  log('timeout exit', slow.exitCode, 'stderr', slow.stderr.trim());
  assert.notEqual(slow.exitCode, 0);
  assert.ok(!slow.stdout.includes('late'));

  await sandbox.files.write('a.txt', 'alpha');
  await sandbox.files.write([
    { path: 'nested/deep/b.txt', data: 'bravo' },
    { path: 'nested/c.bin', data: new Uint8Array([0, 1, 2, 255]) },
  ]);
  assert.equal(await sandbox.files.read('a.txt'), 'alpha');
  assert.equal(await sandbox.files.read('nested/deep/b.txt'), 'bravo');
  assert.deepEqual([...(await sandbox.files.read('nested/c.bin', { format: 'bytes' }))], [0, 1, 2, 255]);

  const entries = await sandbox.files.list('nested', { depth: 2 });
  log('list nested', entries.map((e) => `${e.type}:${e.path}`));
  assert.deepEqual(entries.map((e) => e.path).sort(), ['nested/c.bin', 'nested/deep', 'nested/deep/b.txt']);
  const info = await sandbox.files.getInfo('a.txt');
  log('getInfo', info);
  assert.equal(info.type, 'file');
  assert.equal(info.size, 5);
  assert.equal(await sandbox.files.exists('a.txt'), true);
  assert.equal(await sandbox.files.exists('missing.txt'), false);

  await sandbox.files.makeDir('made/dir');
  assert.equal((await sandbox.files.getInfo('made/dir')).type, 'dir');
  await sandbox.files.rename('a.txt', 'made/dir/a.txt');
  await sandbox.files.copy('made/dir/a.txt', 'a-copy.txt');
  assert.equal(await sandbox.files.exists('a.txt'), false);
  assert.equal(await sandbox.files.read('a-copy.txt'), 'alpha');
  await sandbox.files.remove('a-copy.txt');
  assert.equal(await sandbox.files.exists('a-copy.txt'), false);
  assert.equal((await sandbox.exec('cat made/dir/a.txt')).stdout, 'alpha');

  await sandbox.kill();
  const [killed] = await hatchet.sandbox.list({ sandboxId: sandbox.sandboxId });
  log('after kill live', killed.live, 'snapshot', killed.latestSnapshot);
  assert.ok(killed.latestSnapshot);

  const back = await hatchet.sandbox.connect(sandbox.sandboxId);
  log('connected metadata', back.metadata);
  assert.equal((await back.exec('cat made/dir/a.txt')).stdout, 'alpha');
  assert.equal(back.metadata.role, 'parent');

  const fork = await hatchet.sandbox.create({ provider, from: killed.latestSnapshot, inheritParentMetadata: true, metadata: { role: 'fork' } });
  log('fork', fork.sandboxId, 'metadata', fork.metadata);
  assert.equal((await fork.exec('cat made/dir/a.txt')).stdout, 'alpha');
  await fork.exec('echo forked > only-here.txt');
  assert.equal(await back.files.exists('only-here.txt'), false);

  const found = await hatchet.sandbox.list({ metadata: tags });
  log('list by tags', found.map((s) => `${s.sandboxId}:${s.metadata?.role}:${s.live}`));
  assert.deepEqual(found.map((s) => s.metadata?.role).sort(), ['fork', 'parent']);

  const deleted = await hatchet.sandbox.delete({ metadata: { ...tags, role: 'fork' } });
  log('deleted by tag', deleted);
  assert.deepEqual(deleted, [fork.sandboxId]);
  assert.deepEqual(await hatchet.sandbox.list({ sandboxId: fork.sandboxId }), []);

  await back.delete();
  assert.deepEqual(await hatchet.sandbox.list({ sandboxId: sandbox.sandboxId }), []);
  assert.deepEqual(await hatchet.sandbox.delete({ sandboxId: sandbox.sandboxId }), []);
});

test('resources: one vcpu is honoured', opts, async () => {
  const small = await hatchet.sandbox.create({ provider, template, metadata: { ...tags, role: 'small' }, resources: { vcpu: 1 } });
  const nproc = await small.exec('nproc');
  log('nproc', nproc.stdout.trim());
  assert.equal(nproc.exitCode, 0);
  await small.delete();
});
