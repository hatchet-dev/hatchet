import { createAction } from '@hatchet/clients/dispatcher/action';
import { ActionType } from '@hatchet/protoc/dispatcher';
import { TaskRunTerminatedError } from '@hatchet/util/errors/task-run-terminated-error';
import { Logger } from '@hatchet/util/logger';
import { Context, DurableContext } from './context';
import type { ContextRuntime, DurableTransport } from './runtime';
import { DurableEvictionManager } from './eviction/eviction-manager';

function action(overrides: Partial<Parameters<typeof createAction>[0]> = {}) {
  return createAction({
    tenantId: 'tenant-id',
    workflowRunId: 'workflow-run-id',
    getGroupKeyRunId: '',
    jobId: 'job-id',
    jobName: 'my-task',
    jobRunId: 'job-run-id',
    taskId: 'task-id',
    taskRunExternalId: 'task-run-id',
    actionId: 'my-workflow:my-task',
    actionType: ActionType.START_STEP_RUN,
    actionPayload: JSON.stringify({ input: { n: 1 } }),
    taskName: 'my-task',
    retryCount: 2,
    priority: 1,
    ...overrides,
  });
}

type MockFns<T> = {
  [K in keyof T]: T[K] extends (...args: any[]) => infer R ? jest.Mock<R, any[]> : T[K];
};

type FakeRuntime = ContextRuntime & MockFns<ContextRuntime> & { logs: string[] };

function fakeRuntime(): FakeRuntime {
  const logs: string[] = [];
  const logger = {
    debug: (m: string) => logs.push(`debug:${m}`),
    info: (m: string) => logs.push(`info:${m}`),
    green: (m: string) => logs.push(`green:${m}`),
    warn: (m: string) => logs.push(`warn:${m}`),
    error: (m: string) => logs.push(`error:${m}`),
  };
  return {
    logs,
    namespace: 'ns_',
    logger: jest.fn<any, any[]>(() => logger),
    cancelRun: jest.fn<Promise<void>, any[]>(async () => {}),
    cancelBatch: jest.fn<Promise<void>, any[]>(async () => {}),
    putLog: jest.fn<Promise<void>, any[]>(async () => {}),
    refreshTimeout: jest.fn<Promise<void>, any[]>(async () => {}),
    releaseSlot: jest.fn<Promise<void>, any[]>(async () => {}),
    putStream: jest.fn<Promise<void>, any[]>(async () => {}),
    runWorkflow: jest.fn<Promise<any>, any[]>(async () => ({})),
    runWorkflows: jest.fn<Promise<any[]>, any[]>(async () => []),
    workerId: jest.fn<string | undefined, any[]>(() => 'worker-1'),
    hasWorkflow: jest.fn<boolean, any[]>((name: string) => name === 'ns_child'),
    workerLabels: jest.fn<Record<string, string>, any[]>(() => ({ region: 'eu' })),
    upsertWorkerLabels: jest.fn<Promise<any>, any[]>(async (labels) => labels),
  } as unknown as FakeRuntime;
}

describe('Context on a ContextRuntime', () => {
  it('routes engine-facing operations through the runtime', async () => {
    const runtime = fakeRuntime();
    const ctx = new Context<{ n: number }>(action(), runtime);

    expect(ctx.input).toEqual({ n: 1 });
    expect(ctx.runtime).toBe(runtime);
    expect(ctx.v1).toBeUndefined();

    expect(ctx.worker.id()).toBe('worker-1');
    expect(ctx.worker.labels()).toEqual({ region: 'eu' });
    expect(ctx.worker.hasWorkflow('ns_child')).toBe(true);
    await ctx.worker.upsertLabels({ region: 'us' });
    expect(runtime.upsertWorkerLabels).toHaveBeenCalledWith({ region: 'us' });

    await ctx.logger.warn('careful', { extra: { k: 'v' } });
    expect(runtime.putLog).toHaveBeenCalledWith('task-run-id', 'careful', 'WARN', 2, { k: 'v' });
    expect(runtime.logs).toContain('warn:careful');

    await ctx.refreshTimeout('30s');
    expect(runtime.refreshTimeout).toHaveBeenCalledWith('task-run-id', '30s');

    await ctx.releaseSlot();
    expect(runtime.releaseSlot).toHaveBeenCalledWith('task-run-id');

    await ctx.putStream('chunk');
    await ctx.putStream('chunk-2');
    expect(runtime.putStream).toHaveBeenNthCalledWith(1, 'task-run-id', 'chunk', 0);
    expect(runtime.putStream).toHaveBeenNthCalledWith(2, 'task-run-id', 'chunk-2', 1);

    await ctx.cancel();
    expect(runtime.cancelRun).toHaveBeenCalledWith('task-run-id');
    expect(ctx.cancelled).toBe(true);
  });

  it('cancels every member of a batch through the runtime', async () => {
    const runtime = fakeRuntime();
    const ctx = new Context(
      action({
        batchId: 'batch-1',
        actionType: ActionType.START_BATCH,
        actionPayload: JSON.stringify({ 'run-a': { payload: {} }, 'run-b': { payload: {} } }),
      }),
      runtime
    );

    await ctx.cancel();

    expect(runtime.cancelBatch).toHaveBeenCalledWith({
      workerId: 'worker-1',
      jobId: 'job-id',
      actionId: 'my-workflow:my-task',
      batchId: 'batch-1',
      memberIds: ['run-a', 'run-b'],
    });
    expect(runtime.cancelRun).not.toHaveBeenCalled();
  });

  it('spawns children through the runtime with the namespace applied', async () => {
    const runtime = fakeRuntime();
    const ctx = new Context(action(), runtime);

    await ctx.runNoWaitChild('Child', { x: 1 }, { key: 'k' });

    expect(runtime.runWorkflow).toHaveBeenCalledWith(
      'Child',
      { x: 1 },
      expect.objectContaining({
        parentId: 'workflow-run-id',
        parentTaskRunExternalId: 'task-run-id',
        childIndex: 0,
        childKey: 'k',
        desiredWorkerId: undefined,
      })
    );
  });

  it('still accepts a client and worker', () => {
    const logger = { error: jest.fn() };
    const client = { config: { logger: () => logger, log_level: 'INFO', namespace: 'ns_' } } as any;
    const worker = { workerId: 'w', labels: { a: 1 }, workflow_registry: [{ name: 'x' }] } as any;

    const ctx = new Context(action(), client, worker);

    expect(ctx.v1).toBe(client);
    expect(ctx.runtime.namespace).toBe('ns_');
    expect(ctx.worker.id()).toBe('w');
    expect(ctx.worker.labels()).toEqual({ a: 1 });
    expect(ctx.worker.hasWorkflow('x')).toBe(true);
  });
});

describe('DurableContext on a DurableTransport', () => {
  function fakeTransport() {
    let nodeId = 0;
    const transport = {
      sendEvent: jest.fn<Promise<any>, any[]>(
        async (id: string, invocationCount: number, event: any) => {
          nodeId += 1;
          const base = { invocationCount, durableTaskExternalId: id, branchId: 1, nodeId };
          if (event.kind === 'memo') {
            return { ackType: 'memo', ...base, memoAlreadyExisted: false };
          }
          if (event.kind === 'runChildren') {
            return {
              ackType: 'run',
              ...base,
              runEntries: [{ nodeId, branchId: 1, workflowRunExternalId: 'child-run' }],
            };
          }
          return { ackType: 'waitFor', ...base };
        }
      ),
      waitForCallback: jest.fn<Promise<any>, any[]>(async (id, _invocation, _branch, node) => ({
        durableTaskExternalId: id,
        nodeId: node,
        payload: { CREATE: { key: [{ sleep_duration: '2s' }] } },
        isFailure: false,
        errorMessage: undefined,
      })),
      consumeCallbackWithoutBlocking: jest.fn<void, any[]>(),
      sendMemoCompletedNotification: jest.fn<Promise<void>, any[]>(async () => {}),
      cleanupTaskState: jest.fn<void, any[]>(),
      sendEvictInvocation: jest.fn<Promise<void>, any[]>(async () => {}),
    } as unknown as DurableTransport & MockFns<DurableTransport>;
    return transport;
  }

  it('sends waitFor events over the transport and resolves with the callback payload', async () => {
    const runtime = fakeRuntime();
    const transport = fakeTransport();
    const ctx = new DurableContext(action({ durableTaskInvocationCount: 3 }), runtime, transport, {
      engineVersion: 'v0.81.0',
    });

    expect(ctx.supportsEviction).toBe(true);
    expect(ctx.durableListener).toBe(transport);
    expect(ctx.invocationCount).toBe(3);

    const result = await ctx.sleepFor('10s');

    expect(result).toEqual({ durationMs: 2000 });
    expect(transport.sendEvent).toHaveBeenCalledWith(
      'task-run-id',
      3,
      expect.objectContaining({ kind: 'waitFor' })
    );
    expect(transport.waitForCallback).toHaveBeenCalledWith('task-run-id', 3, 1, 1, {
      signal: ctx.abortController.signal,
    });
  });

  it('memoizes now() through the transport with a WebCrypto memo key', async () => {
    const runtime = fakeRuntime();
    const transport = fakeTransport();
    const ctx = new DurableContext(action(), runtime, transport, { engineVersion: 'v0.81.0' });

    const now = await ctx.now();

    expect(now).toBeInstanceOf(Date);
    const [, , event] = transport.sendEvent.mock.calls[0] as any;
    expect(event.kind).toBe('memo');
    expect(event.memoKey).toBeInstanceOf(Uint8Array);
    expect(event.memoKey).toHaveLength(32);
    expect(transport.consumeCallbackWithoutBlocking).toHaveBeenCalledWith('task-run-id', 1, 1, 1);
    expect(transport.sendMemoCompletedNotification).toHaveBeenCalledWith(
      'task-run-id',
      1,
      1,
      1,
      event.memoKey,
      expect.any(Uint8Array)
    );
  });

  it('spawns children over the transport with the namespace applied', async () => {
    const runtime = fakeRuntime();
    const transport = fakeTransport();
    transport.waitForCallback.mockResolvedValue({
      durableTaskExternalId: 'task-run-id',
      nodeId: 1,
      payload: { ok: true },
      isFailure: false,
      errorMessage: undefined,
    });
    const ctx = new DurableContext(action(), runtime, transport, { engineVersion: 'v0.81.0' });

    const result = await ctx.spawnChild('Child', { x: 1 });

    expect(result).toEqual({ ok: true });
    const [, , event] = transport.sendEvent.mock.calls[0] as any;
    expect(event.kind).toBe('runChildren');
    expect(event.triggerOpts[0]).toMatchObject({
      name: 'ns_child',
      input: JSON.stringify({ x: 1 }),
      parentId: 'workflow-run-id',
      parentTaskRunExternalId: 'task-run-id',
      childIndex: 0,
    });
    expect(runtime.runWorkflow).not.toHaveBeenCalled();
  });

  it('fails clearly when the engine predates eviction and the transport has no legacy fallback', async () => {
    const ctx = new DurableContext(action(), fakeRuntime(), fakeTransport(), {
      engineVersion: 'v0.70.0',
    });

    await expect(ctx.waitFor({ sleepFor: '1s' })).rejects.toThrow(
      'does not support durable eviction and the durable transport has no legacy fallback'
    );
  });
});

describe('DurableContext eviction during wait registration', () => {
  class NoopLogger extends Logger {
    debug() {}
    info() {}
    green() {}
    warn() {}
    error() {}
    util() {}
  }

  interface Deferred<T> {
    promise: Promise<T>;
    resolve: (v: T) => void;
    reject: (e: any) => void;
  }

  function deferred<T>(): Deferred<T> {
    let resolve!: (v: T) => void;
    let reject!: (e: any) => void;
    const promise = new Promise<T>((res, rej) => {
      resolve = res;
      reject = rej;
    });
    return { promise, resolve, reject };
  }

  // sendEvent stays pending until resolved or until cleanupTaskState rejects
  // it, matching the real listener's pending ack handling.
  function pendingSendTransport() {
    const pendingSends = new Set<Deferred<any>>();
    const transport = {
      sendEvent: jest.fn<Promise<any>, any[]>(() => {
        const d = deferred<any>();
        d.promise.catch(() => {}).finally(() => pendingSends.delete(d));
        pendingSends.add(d);
        return d.promise;
      }),
      waitForCallback: jest.fn<Promise<any>, any[]>(async () => ({
        durableTaskExternalId: 'task-run-id',
        nodeId: 1,
        payload: {},
        isFailure: false,
        errorMessage: undefined,
      })),
      consumeCallbackWithoutBlocking: jest.fn<void, any[]>(),
      sendMemoCompletedNotification: jest.fn<Promise<void>, any[]>(async () => {}),
      cleanupTaskState: jest.fn<void, any[]>(() => {
        pendingSends.forEach((d) =>
          d.reject(new TaskRunTerminatedError('evicted', 'task state cleaned up'))
        );
        pendingSends.clear();
      }),
      sendEvictInvocation: jest.fn<Promise<void>, any[]>(async () => {}),
    } as unknown as DurableTransport & MockFns<DurableTransport>;
    return { transport, pendingSends };
  }

  function setup(transport: DurableTransport) {
    const ref: { ctx?: DurableContext<any, any> } = {};
    const cancelLocal = jest.fn((_key: any, _invocationCount: number) => {
      ref.ctx!.abortController.abort(new TaskRunTerminatedError('evicted'));
      transport.cleanupTaskState('task-run-id', 3);
    });
    const requestEvictionWithAck = jest.fn(async (_key: any, rec: any) => {
      await transport.sendEvictInvocation(
        rec.taskRunExternalId,
        rec.invocationCount,
        rec.evictionReason
      );
    });
    const manager = new DurableEvictionManager({
      durableSlots: 10,
      cancelLocal,
      requestEvictionWithAck,
      config: { checkIntervalMs: 3_600_000 },
      logger: new NoopLogger(),
    });
    const act = action({ durableTaskInvocationCount: 3 });
    ref.ctx = new DurableContext(act, fakeRuntime(), transport, {
      engineVersion: 'v0.81.0',
      evictionManager: manager,
    });
    manager.registerRun(act.key, 'task-run-id', 3, {
      ttl: '30s',
      allowCapacityEviction: true,
      priority: 0,
    });
    return { ctx: ref.ctx, manager, cancelLocal, act };
  }

  it('is visible to shutdown eviction while its waitFor registration is in flight', async () => {
    const { transport, pendingSends } = pendingSendTransport();
    const { ctx, manager, act } = setup(transport);

    const waiting = ctx.waitFor({ sleepFor: '1s' });
    // sendEvent goes out on the serialized-send microtask.
    await new Promise((r) => setImmediate(r));

    // The ack has not landed, but the run must already read as waiting;
    // previously the mark only happened after the sendEvent round-trip.
    expect(manager.cache.get(act.key)?._waitCount).toBe(1);
    expect(pendingSends.size).toBe(1);

    // Resolve the registration so the wait completes and the mark clears.
    pendingSends.forEach((d) =>
      d.resolve({
        ackType: 'waitFor',
        durableTaskExternalId: 'task-run-id',
        invocationCount: 3,
        branchId: 1,
        nodeId: 7,
      })
    );
    await waiting;
    expect(manager.cache.get(act.key)?._waitCount ?? 0).toBe(0);
  });

  it('evicts a run whose waitFor registration is in flight when shutdown starts', async () => {
    const { transport } = pendingSendTransport();
    const { ctx, manager, cancelLocal, act } = setup(transport);

    const waiting = ctx.waitFor({ sleepFor: '1s' });
    await new Promise((r) => setImmediate(r));

    const evicted = await manager.evictAllWaiting();

    expect(evicted).toBe(1);
    expect(transport.sendEvictInvocation).toHaveBeenCalledWith(
      'task-run-id',
      3,
      expect.any(String)
    );
    expect(cancelLocal).toHaveBeenCalledWith(act.key, 3);
    await expect(waiting).rejects.toBeInstanceOf(TaskRunTerminatedError);
  });

  it('updates the recorded resource id once the registration ack lands', async () => {
    const { transport, pendingSends } = pendingSendTransport();
    const { ctx, manager, act } = setup(transport);

    let observed: { waitCount?: number; resourceId?: string } = {};
    transport.waitForCallback.mockImplementation(async () => {
      const rec = manager.cache.get(act.key);
      observed = { waitCount: rec?._waitCount, resourceId: rec?.waitResourceId };
      return {
        durableTaskExternalId: 'task-run-id',
        nodeId: 7,
        payload: {},
        isFailure: false,
        errorMessage: undefined,
      };
    });

    const waiting = ctx.waitFor({ sleepFor: '1s' });
    await new Promise((r) => setImmediate(r));
    // sleepFor conditions render a declared readableDataKey, so it is recorded
    // before the registration round-trip starts.
    expect(manager.cache.get(act.key)?.waitResourceId).toBe('sleep-1s');
    expect(pendingSends.size).toBe(1);

    pendingSends.forEach((d) =>
      d.resolve({
        ackType: 'waitFor',
        durableTaskExternalId: 'task-run-id',
        invocationCount: 3,
        branchId: 1,
        nodeId: 7,
      })
    );
    await waiting;

    // The post-ack update refreshes metadata without double-counting the wait.
    expect(observed).toEqual({ waitCount: 1, resourceId: 'sleep-1s' });
    expect(transport.waitForCallback).toHaveBeenCalledWith('task-run-id', 3, 1, 7, {
      signal: ctx.abortController.signal,
    });
  });
});
