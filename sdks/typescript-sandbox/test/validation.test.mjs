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
  const isNotFound = (e) => e instanceof FilesystemError && e.code === 'not_found';
  await assert.rejects(failing.exists('a'), isNotFound);
  await assert.rejects(failing.rename('a', 'b'), isNotFound);
});

function fakeClient() {
  const calls = { started: [], cancelled: [] };
  const turn = (output) => ({ 'hatchet-sandbox': { output, sandboxId: 'sb' } });
  let finish;
  const client = {
    config: {},
    admin: {
      logger: { warn: () => {} },
      runWorkflow: async (name, payload) => {
        calls.started.push({ name, payload });
        return {
          getWorkflowRunId: async () => 'run-1',
          result: () =>
            new Promise((resolve) => {
              finish = () => resolve(turn({ stdout: 'out', stderr: '', exitCode: 0 }));
            }),
        };
      },
    },
    workflows: { list: async () => ({ rows: [{ name: 'hatchet-sandbox' }] }) },
    workflow: () => ({ run: async () => turn({}) }),
    runs: {
      cancel: async ({ ids }) => {
        calls.cancelled.push(...ids);
      },
    },
  };
  return { client, calls, finish: () => finish() };
}

test('exec with a signal resolves with the command output', async () => {
  const { client, calls, finish } = fakeClient();
  const sandbox = await Sandbox.connect(client, 'sb');
  const pending = sandbox.exec('echo out', { signal: new AbortController().signal });
  await new Promise((resolve) => setImmediate(resolve));
  finish();
  assert.equal((await pending).stdout, 'out');
  assert.equal(calls.started[0].name, 'hatchet-sandbox');
  assert.equal(calls.started[0].payload.fn, '$command');
  assert.equal('signal' in calls.started[0].payload.input, false);
  assert.deepEqual(calls.cancelled, []);
});

test('exec with a signal cancels its run when the signal aborts', async () => {
  const { client, calls } = fakeClient();
  const sandbox = await Sandbox.connect(client, 'sb');
  const abort = new AbortController();
  const pending = sandbox.exec('sleep 60', { signal: abort.signal });
  await new Promise((resolve) => setImmediate(resolve));
  abort.abort();
  await assert.rejects(pending, (e) => e.name === 'AbortError');
  await new Promise((resolve) => setImmediate(resolve));
  assert.deepEqual(calls.cancelled, ['run-1']);
});

test('exec with an aborted signal starts nothing', async () => {
  const { client, calls } = fakeClient();
  const sandbox = await Sandbox.connect(client, 'sb');
  await assert.rejects(
    sandbox.exec('true', { signal: AbortSignal.abort() }),
    (e) => e.name === 'AbortError'
  );
  assert.deepEqual(calls.started, []);
});
