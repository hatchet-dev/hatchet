/**
 * The invocation's boundaries against an operator that misbehaves or goes away: every
 * lifecycle transition settles the invocation promise, inbound references are checked
 * against the socket's task and invocation, frames and pending state are bounded, and a
 * malformed payload fails the invocation instead of escaping the socket listener.
 *
 * The socket here delivers synchronously, so a throw out of the endpoint's message listener
 * surfaces in the test instead of in a microtask.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ActionType, AssignedAction } from '../../generated/proto/dispatcher';
import type {
  DurableEventLogEntryRef,
  DurableTaskResponse,
} from '../../generated/proto/v1/dispatcher';
import type { ServerlessDurableFrame } from '../../generated/proto/v1/serverless';
import { hatchet, type TaskWorkflowDeclaration } from '../../index';
import { createSocketPair } from '../../testing/socket-pair';
import { buildRegistry } from '../registry';
import { RunStreamClosedError, RunWatcher } from '../runs';
import { decodeFrame, encodeFrame } from './frames';
import { runDurableInvocation, type DurableInvocationOptions } from './invocation';
import type { DurableSocket } from './socket';
import { SocketStreams } from './streams';
import { FrameTransport } from './transport';

const quiet = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };
const encoder = new TextEncoder();

const sleepOnly = hatchet.durableTask({
  name: 'sleep-only',
  fn: async (_input: {}, ctx) => {
    await ctx.sleepFor('1s');
    return { slept: true };
  },
});

function fakeSocket() {
  const sent: ServerlessDurableFrame[] = [];
  const messageListeners: Array<(text: string) => void> = [];
  const closeListeners: Array<(code: number, reason: string) => void> = [];
  let closed: { code: number; reason: string } | undefined;

  const socket: DurableSocket = {
    send: (text) => {
      if (closed) throw new Error('socket is closed');
      sent.push(decodeFrame(text));
    },
    close: (code = 1005, reason = '') => {
      if (closed) return;
      closed = { code, reason };
      queueMicrotask(() => {
        for (const listener of closeListeners) listener(code, reason);
      });
    },
    onMessage: (listener) => {
      messageListeners.push(listener);
    },
    onClose: (listener) => {
      closeListeners.push(listener);
    },
  };

  return {
    socket,
    sent,
    get closed() {
      return closed;
    },
    /** Delivers a text frame to the endpoint synchronously. */
    deliver(text: string) {
      for (const listener of messageListeners) listener(text);
    },
    /** The operator closes the socket. */
    closeFromOperator(code: number, reason = '') {
      closed = { code, reason };
      for (const listener of closeListeners) listener(code, reason);
    },
  };
}

function start(
  task: TaskWorkflowDeclaration<any, any>,
  options: Partial<DurableInvocationOptions> & { inlineWaitBudgetMs?: number } = {}
) {
  const { inlineWaitBudgetMs = 5000, ...rest } = options;
  const fake = fakeSocket();
  const registry = buildRegistry([task], undefined, () => {});
  const taskId = crypto.randomUUID();
  const { name } = task.definition;
  let settled = false;

  const run = runDurableInvocation({
    socket: fake.socket,
    registry,
    console: quiet,
    expected: { taskRunExternalId: taskId, invocationCount: 1 },
    ...rest,
  });
  run.then(
    () => {
      settled = true;
    },
    () => {
      settled = true;
    }
  );

  const ref = (nodeId: number, over: Partial<DurableEventLogEntryRef> = {}) => ({
    durableTaskExternalId: taskId,
    invocationCount: 1,
    branchId: 0,
    nodeId,
    ...over,
  });

  return {
    fake,
    run,
    taskId,
    ref,
    isSettled: () => settled,
    last: () => fake.sent[fake.sent.length - 1],
    sendFirst() {
      const action = AssignedAction.fromPartial({
        tenantId: crypto.randomUUID(),
        workflowRunId: crypto.randomUUID(),
        jobId: crypto.randomUUID(),
        jobName: name,
        jobRunId: crypto.randomUUID(),
        taskId: crypto.randomUUID(),
        taskRunExternalId: taskId,
        actionId: `${name}:${name}`,
        actionType: ActionType.START_STEP_RUN,
        actionPayload: JSON.stringify({ input: {}, parents: {}, triggered_by: 'manual' }),
        taskName: name,
        retryCount: 0,
        priority: 1,
        durableTaskInvocationCount: 1,
      });

      fake.deliver(encodeFrame({ first: { action, invocationCount: 1, inlineWaitBudgetMs } }));
    },
    respond(response: DurableTaskResponse) {
      fake.deliver(encodeFrame({ response }));
    },
  };
}

async function settle(): Promise<void> {
  await vi.advanceTimersByTimeAsync(0);
}

describe('invocation lifecycle settles', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    expect(vi.getTimerCount()).toBe(0);
    vi.useRealTimers();
  });

  it('settles the invocation when the socket closes before the first frame', async () => {
    const h = start(sleepOnly);
    await settle();

    h.fake.closeFromOperator(1001, 'going away');
    await settle();

    expect(h.isSettled()).toBe(true);
    expect(h.fake.sent).toEqual([]);
  });

  it('settles the invocation when the socket closes while the eviction ack is pending', async () => {
    const h = start(sleepOnly, { inlineWaitBudgetMs: 0 });
    h.sendFirst();
    await settle();
    expect(h.last().request?.waitFor).toBeDefined();

    h.respond({ waitForAck: { ref: h.ref(1) } });
    await settle();
    expect(h.last().request?.evictInvocation).toBeDefined();

    h.fake.closeFromOperator(1001, 'going away');
    await settle();

    expect(h.isSettled()).toBe(true);
    expect(h.last().request?.evictInvocation).toBeDefined();
  });
});

describe('completion identity', () => {
  function transportHarness() {
    const fake = fakeSocket();
    const transport = new FrameTransport({
      socket: fake.socket,
      durableTaskExternalId: 'task-a',
      invocationCount: 1,
      inlineWaitBudgetMs: 60_000,
      onBudgetElapsed: () => {},
      onEngineError: () => {},
      onServerEvict: () => {},
      console: quiet,
    });
    fake.socket.onMessage((text) => transport.handleFrame(decodeFrame(text)));

    const completion = (ref: DurableEventLogEntryRef, payload = '{}') =>
      fake.deliver(
        encodeFrame({
          response: { entryCompleted: { ref, payload: encoder.encode(payload), isFailure: false } },
        })
      );
    const flush = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

    return { fake, transport, completion, flush };
  }

  it('ignores a completion whose reference names another task or invocation', async () => {
    const { transport, completion, flush } = transportHarness();
    let resolved: unknown;
    void transport.waitForCallback('task-a', 1, 0, 1).then((result) => {
      resolved = result;
    });

    completion({ durableTaskExternalId: 'task-b', invocationCount: 1, branchId: 0, nodeId: 1 });
    await flush();
    expect(resolved).toBeUndefined();

    completion({ durableTaskExternalId: 'task-a', invocationCount: 2, branchId: 0, nodeId: 1 });
    await flush();
    expect(resolved).toBeUndefined();

    completion(
      { durableTaskExternalId: 'task-a', invocationCount: 1, branchId: 0, nodeId: 1 },
      '{"from":"task-a"}'
    );
    await flush();
    expect(resolved).toMatchObject({ payload: { from: 'task-a' } });

    transport.dispose();
  });

  it('rejects an ack whose reference names another task', async () => {
    const { fake, transport, flush } = transportHarness();
    const pending = transport.sendEvent('task-a', 1, {
      kind: 'memo',
      memoKey: encoder.encode('k'),
      payload: encoder.encode('{}'),
    });
    pending.catch(() => {});
    await flush();
    expect(fake.sent[0]?.request?.memo).toBeDefined();

    fake.deliver(
      encodeFrame({
        response: {
          memoAck: {
            ref: { durableTaskExternalId: 'task-b', invocationCount: 1, branchId: 0, nodeId: 1 },
            memoAlreadyExisted: false,
          },
        },
      })
    );

    await expect(pending).rejects.toThrow(/task-b/);
    transport.dispose();
  });
});

describe('frame and pending-state budgets', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    expect(vi.getTimerCount()).toBe(0);
    vi.useRealTimers();
  });

  it('fails the invocation on a frame larger than maxFrameBytes', async () => {
    const h = start(sleepOnly, { maxFrameBytes: 1024 });
    h.sendFirst();
    await settle();

    h.fake.deliver('x'.repeat(2048));
    await settle();

    expect(h.last().done).toMatchObject({ error: expect.stringMatching(/frame/), retry: true });
    expect(h.fake.closed).toBeDefined();
    expect(h.isSettled()).toBe(true);
  });

  it('fails the invocation when unrequested completions exceed maxPendingCompletions', async () => {
    const h = start(sleepOnly, { maxPendingCompletions: 4 });
    h.sendFirst();
    await settle();
    h.respond({ waitForAck: { ref: h.ref(1) } });
    await settle();

    for (let node = 100; node < 108; node++) {
      h.respond({
        entryCompleted: { ref: h.ref(node), payload: encoder.encode('{}'), isFailure: false },
      });
    }
    await settle();

    expect(h.last().done).toMatchObject({
      error: expect.stringMatching(/completions/),
      retry: true,
    });
    expect(h.isSettled()).toBe(true);
  });
});

describe('malformed completions', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    expect(vi.getTimerCount()).toBe(0);
    vi.useRealTimers();
  });

  it('fails the invocation instead of throwing when a completion payload is not JSON', async () => {
    const h = start(sleepOnly);
    h.sendFirst();
    await settle();
    h.respond({ waitForAck: { ref: h.ref(1) } });
    await settle();

    expect(() =>
      h.respond({
        entryCompleted: { ref: h.ref(1), payload: encoder.encode('not json'), isFailure: false },
      })
    ).not.toThrow();
    await settle();

    expect(h.last().done).toMatchObject({
      error: expect.stringMatching(/payload/),
      retry: true,
    });
    expect(h.isSettled()).toBe(true);
  });
});

describe('run watcher shutdown', () => {
  const flush = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

  it('close() rejects outstanding waits', async () => {
    const pair = createSocketPair();
    const streams = new SocketStreams({ socket: pair.endpoint, console: quiet });
    const watcher = new RunWatcher(streams);
    let outcome: unknown;

    watcher.awaitRun('run-1').then(
      () => {
        outcome = 'resolved';
      },
      (err) => {
        outcome = err;
      }
    );
    watcher.close();
    await flush();

    expect(outcome).toBeInstanceOf(RunStreamClosedError);
  });

  it('closeAll() rejects the waits of a watcher on the streams', async () => {
    const pair = createSocketPair();
    const streams = new SocketStreams({ socket: pair.endpoint, console: quiet });
    const watcher = new RunWatcher(streams);
    let outcome: unknown;

    watcher.awaitRun('run-1').then(
      () => {
        outcome = 'resolved';
      },
      (err) => {
        outcome = err;
      }
    );
    streams.closeAll();
    await flush();

    expect(outcome).toBeInstanceOf(RunStreamClosedError);
  });
});
