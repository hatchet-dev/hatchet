import assert from 'node:assert/strict';
import { test } from 'node:test';
import { ZodError } from 'zod';
import { Sandbox, Files, FilesystemError } from '../dist/index.js';

const offline = {
  config: {},
  workflows: {
    list: async () => {
      throw new Error('network reached');
    },
  },
};

async function rejectsZod(promise, pattern) {
  await assert.rejects(promise, (e) => e instanceof ZodError && pattern.test(e.message));
}

test('sandbox options are validated before any network call', async () => {
  await rejectsZod(Sandbox.create(offline, { resources: { memory: '1073741824' } }), /512Mi/);
  await rejectsZod(Sandbox.create(offline, { resources: {} }), /at least one/);
  await rejectsZod(Sandbox.create(offline, { resources: { vcpu: 1.5 } }), /vcpu/);
  await rejectsZod(Sandbox.create(offline, { id: 'Bad_ID' }), /lowercase/);
  await rejectsZod(Sandbox.create(offline, { idleTimeout: 5 }), /Unrecognized key/);
  await rejectsZod(Sandbox.create(offline, { inheritParentMetadata: true }), /needs from/);
  await rejectsZod(Sandbox.connect(offline, ''), /Too small/);
  await rejectsZod(Sandbox.delete(offline, { metadata: {} }), /sandboxId or a metadata/);
});

test('file operations validate input and parse provider output', async () => {
  let sent;
  const files = new Files(async (fn, input) => {
    sent = { fn, input };
    return {
      entry: {
        name: 'a',
        path: 'a',
        type: 'file',
        size: 1,
        mode: 420,
        permissions: '-rw-r--r--',
        modifiedTime: '2026-11-11T00:00:00Z',
      },
    };
  });
  await rejectsZod(files.download('http://example.com/x', 'a'), /https/);
  await rejectsZod(files.read(''), /Too small/);
  await rejectsZod(files.list('.', { depth: 0 }), /depth/);
  await rejectsZod(files.writeFiles([{ path: 'a', data: 42 }]), /data/);

  const info = await files.getInfo('a');
  assert.ok(info.modifiedTime instanceof Date);

  await files.download('https://example.com/x', 'a', { timeoutMs: 5 });
  assert.equal(sent.fn, '$fs.download');
  assert.deepEqual(sent.input, { url: 'https://example.com/x', path: 'a', timeoutMs: 5 });

  const noEntry = new Files(async () => ({}));
  await rejectsZod(noEntry.rename('a', 'b'), /entry/);
  assert.equal(await noEntry.exists('a'), false);

  const failing = new Files(async () => ({ error: { code: 'not_found', message: 'nope' } }));
  await assert.rejects(
    failing.exists('a'),
    (e) => e instanceof FilesystemError && e.code === 'not_found'
  );
});
