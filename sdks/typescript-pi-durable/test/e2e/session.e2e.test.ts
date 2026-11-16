// The core flow against a real Hatchet engine: an input runs a tool in a sandbox, a fork branches the conversation
// and its workspace, an abort kills the running command, and a killed worker hands the session to another one.
// Run with PI_E2E=1 and a client configured for the engine (HATCHET_CLIENT_TOKEN and friends).
import '@hatchet-dev/sandbox';
import { type ChildProcess, spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { readFileSync } from 'node:fs';
import * as fs from 'node:fs/promises';
import * as os from 'node:os';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import { afterAll, beforeAll, describe, expect, it } from 'vitest';
import { type ConversationSummary, SessionClient, type SessionView } from '../../src/index.js';
import { fakeSandboxProvider } from './fake-provider.js';

const enabled = process.env.PI_E2E === '1';
const HERE = path.dirname(fileURLToPath(import.meta.url));

/** A task name of its own, so runs left over from an earlier test run never pick up these sessions. */
const TASK = `pi-e2e-${randomUUID().slice(0, 8)}`;

describe.skipIf(!enabled)('pi-durable on Hatchet', () => {
  let hatchet: HatchetClient;
  let provider: ReturnType<typeof fakeSandboxProvider>;
  let providerWorker: Awaited<ReturnType<HatchetClient['worker']>>;
  let sessions: SessionClient;
  let dataDir: string;
  const workers: ChildProcess[] = [];

  /** Starts a session worker in its own process, and resolves once it has registered. */
  function startWorker(): Promise<ChildProcess> {
    // node itself, with tsx as a loader: the tsx binary runs the worker in a child process that a kill would miss.
    const child = spawn(process.execPath, ['--import', 'tsx', path.join(HERE, 'worker.ts')], {
      env: { ...process.env, PI_E2E_TASK: TASK, PI_E2E_DATA: dataDir },
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    workers.push(child);
    const prefix = `[worker ${child.pid}]`;
    child.stderr!.on('data', (chunk) => process.stderr.write(`${prefix} ${chunk}`));
    return new Promise((resolve, reject) => {
      child.stdout!.on('data', (chunk: Buffer) => {
        if (process.env.PI_E2E_VERBOSE) process.stdout.write(`${prefix} ${chunk}`);
        if (chunk.toString().includes('PI_E2E_READY')) resolve(child);
      });
      child.on('exit', (code) =>
        reject(new Error(`${prefix} exited with ${code} before it was ready`))
      );
    });
  }

  /** Reads the session's views until one matches. */
  async function waitFor(
    sessionId: string,
    predicate: (view: SessionView) => boolean,
    timeoutMs = 60_000
  ): Promise<SessionView> {
    const abort = new AbortController();
    const timer = setTimeout(() => abort.abort(), timeoutMs);
    let last: SessionView | undefined;
    try {
      for await (const view of sessions.views(sessionId, abort.signal)) {
        last = view;
        if (predicate(view)) return view;
      }
    } finally {
      clearTimeout(timer);
      abort.abort();
    }
    throw new Error(`no matching view within ${timeoutMs} ms; last view: ${JSON.stringify(last)}`);
  }

  async function until(check: () => boolean, timeoutMs = 60_000) {
    const deadline = Date.now() + timeoutMs;
    while (!check()) {
      if (Date.now() > deadline) throw new Error(`condition not met within ${timeoutMs} ms`);
      await new Promise((resolve) => setTimeout(resolve, 100));
    }
  }

  const conversation = (view: SessionView, id: number) =>
    view.conversations.find((c) => c.id === id);
  const answered = (c: ConversationSummary | undefined, text: string) =>
    c?.items.some((item) => item.kind === 'assistant' && item.text === text) ?? false;
  const workspaceFile = (c: ConversationSummary, file: string) =>
    readFileSync(path.join(provider.workspaceOf(c.sandboxId!), file), 'utf8');

  beforeAll(async () => {
    dataDir = await fs.mkdtemp(path.join(os.tmpdir(), 'pi-e2e-'));
    hatchet = HatchetClient.init();
    provider = fakeSandboxProvider(hatchet, path.join(dataDir, 'provider'));
    providerWorker = await hatchet.worker(`${TASK}-provider`, {
      workflows: [provider.task],
      slots: 50,
    });
    void providerWorker.start();
    await providerWorker.waitUntilReady(30_000);
    sessions = new SessionClient(hatchet, { taskName: TASK });
    await startWorker();
  }, 120_000);

  afterAll(async () => {
    for (const child of workers) child.kill('SIGKILL');
    await providerWorker?.stop();
    if (dataDir) await fs.rm(dataDir, { recursive: true, force: true });
  });

  let sessionId: string;

  it('runs a tool call in the conversation sandbox', async () => {
    sessionId = await sessions.create();
    await sessions.send(sessionId, {
      type: 'input',
      conversationId: 1,
      text: 'bash: echo hello > out.txt && cat out.txt',
    });

    const view = await waitFor(sessionId, (v) => answered(conversation(v, 1), 'result: hello'));
    const root = conversation(view, 1)!;
    expect(root.sandboxId).toBeDefined();
    expect(root.snapshotId).toBeDefined();
    expect(root.items.find((item) => item.kind === 'tool')).toMatchObject({
      name: 'bash',
      isError: false,
    });
    expect(workspaceFile(root, 'out.txt')).toBe('hello\n');
  });

  it('forks the conversation with its workspace', async () => {
    const before = await waitFor(sessionId, () => true);
    const root = conversation(before, 1)!;
    const answer = root.items.filter((item) => item.kind === 'assistant').at(-1)!;
    await sessions.send(sessionId, { type: 'fork', conversationId: 1, entryId: answer.id });

    const forked = await waitFor(sessionId, (v) => v.conversations.length === 2);
    const fork = forked.conversations.find((c) => c.parent)!;
    expect(fork.parent).toEqual({ conversationId: 1, entryId: answer.id });
    expect(fork.sandboxId).toBeDefined();
    expect(fork.sandboxId).not.toBe(root.sandboxId);
    expect(workspaceFile(fork, 'out.txt')).toBe('hello\n');

    await sessions.send(sessionId, {
      type: 'input',
      conversationId: fork.id,
      text: 'bash: echo branch > out.txt && cat out.txt',
    });
    await waitFor(sessionId, (v) => answered(conversation(v, fork.id), 'result: branch'));
    expect(workspaceFile(fork, 'out.txt')).toBe('branch\n');
    expect(workspaceFile(root, 'out.txt')).toBe('hello\n');
  });

  it('aborts a running command and kills it in the sandbox', async () => {
    const started = provider.log.commands.length;
    const sleeping = () =>
      provider.log.commands.slice(started).find((c) => c.cmd.includes('sleep 30'));
    await sessions.send(sessionId, {
      type: 'input',
      conversationId: 1,
      text: 'bash: sleep 30 && echo late',
    });
    await until(() => sleeping() !== undefined);

    const abortedAt = Date.now();
    await sessions.send(sessionId, { type: 'abort', conversationId: 1 });
    const view = await waitFor(sessionId, (v) => conversation(v, 1)?.busy === false, 20_000);
    expect(Date.now() - abortedAt).toBeLessThan(20_000);
    expect(answered(conversation(view, 1), 'result: late')).toBe(false);
    await until(() => sleeping()?.killed === true, 10_000);
  });

  it('moves the session to another worker when its worker dies', async () => {
    const crashSession = await sessions.create();
    const started = provider.log.commands.length;
    await sessions.send(crashSession, {
      type: 'input',
      conversationId: 1,
      text: 'bash: sleep 20 && echo survived',
    });
    const sleeping = () =>
      provider.log.commands.slice(started).find((c) => c.cmd.includes('sleep 20'));
    await until(() => sleeping() !== undefined);
    const running = await waitFor(crashSession, (v) => v.host.state === 'running');

    // Only the first worker existed when the run started, so it owns the session.
    const [owner] = workers;
    await startWorker();
    // The command must still be running, or the session would have nothing to recover.
    expect(sleeping()?.exitCode).toBeUndefined();
    owner!.kill('SIGKILL');

    // The new owner does not run the command again, since it may have partly run: pi-durable records the call as
    // interrupted, and the agent carries on from that result.
    const interrupted = (v: SessionView) =>
      conversation(v, 1)?.items.some(
        (item) => item.kind === 'tool' && item.isError && item.output.includes('was interrupted')
      ) ?? false;
    const view = await waitFor(
      crashSession,
      (v) =>
        interrupted(v) &&
        conversation(v, 1)!.items.at(-1)?.kind === 'assistant' &&
        !conversation(v, 1)!.busy,
      180_000
    );
    // A retry of the killed run, or a queued wake run of the session, takes over: either way another run.
    expect(view.host.runId !== running.host.runId || view.host.attempt > running.host.attempt).toBe(
      true
    );
    expect(answered(conversation(view, 1), 'result: survived')).toBe(false);
  }, 240_000);
});
