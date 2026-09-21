import { DurableTaskResponse } from '@hatchet/protoc/v1/dispatcher';
import { HatchetClient } from '../client';
import { TenantClient } from '../features/tenant';
import { DurableEvictionManager } from './eviction/eviction-manager';
import { InternalWorker } from './worker-internal';

function gate() {
  let release = () => {};
  const promise = new Promise<void>((resolve) => {
    release = resolve;
  });
  return { promise, release: () => release() };
}

describe('durable worker shutdown', () => {
  let client: HatchetClient;
  let worker: InternalWorker;
  let manager: DurableEvictionManager;
  let acknowledgement: ReturnType<typeof gate>;
  let requests: string[];
  let cancelled: string[];
  let signalListeners: Map<NodeJS.Signals, NodeJS.SignalsListener[]>;

  beforeEach(async () => {
    jest.useFakeTimers();
    jest.spyOn(TenantClient.prototype, 'get').mockRejectedValue(new Error('No test engine'));
    const claims = Buffer.from(
      JSON.stringify({
        sub: 'test-tenant',
        server_url: 'http://localhost:1',
        grpc_broadcast_address: 'localhost:1',
      })
    ).toString('base64url');
    client = HatchetClient.init({
      token: `test.${claims}.test`,
      tls_config: { tls_strategy: 'none' },
      logger: () => ({
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        green: jest.fn(),
        util: jest.fn(),
      }),
    });
    acknowledgement = gate();
    requests = [];
    cancelled = [];
    jest
      .spyOn(client.durableListener.client, 'durableTask')
      .mockImplementation(async function* (outgoing) {
        for await (const request of outgoing) {
          if (request.evictInvocation) {
            requests.push(request.evictInvocation.durableTaskExternalId ?? '');
            await acknowledgement.promise;
            yield DurableTaskResponse.fromPartial({ evictionAck: request.evictInvocation });
          }
        }
      });
    signalListeners = new Map(
      (['SIGTERM', 'SIGINT'] as const).map((signal) => [signal, process.listeners(signal)])
    );
    worker = new InternalWorker(client, {
      name: 'shutdown-test',
      handleKill: false,
      durableSlots: 2,
    });
    manager = new DurableEvictionManager({
      durableSlots: 2,
      cancelLocal: (key) => cancelled.push(key),
      requestEvictionWithAck: (_key, run) =>
        client.durableListener.sendEvictInvocation(
          run.taskRunExternalId,
          run.invocationCount,
          run.evictionReason
        ),
      logger: worker.logger,
    });
    worker.evictionManager = manager;
    await client.durableListener.start('test-worker');
  });

  afterEach(async () => {
    acknowledgement.release();
    await client.durableListener.stop();
    for (const [signal, previous] of signalListeners) {
      for (const listener of process.listeners(signal)) {
        if (!previous.includes(listener)) {
          process.removeListener(signal, listener);
        }
      }
    }
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  it('resolves concurrent eviction requests from one acknowledgement', async () => {
    const evictions = Promise.allSettled([
      client.durableListener.sendEvictInvocation('waiting-task', 1),
      client.durableListener.sendEvictInvocation('waiting-task', 1),
    ]);
    acknowledgement.release();
    await jest.advanceTimersByTimeAsync(30_000);

    expect(await evictions).toEqual([
      { status: 'fulfilled', value: undefined },
      { status: 'fulfilled', value: undefined },
    ]);
    expect(requests).toEqual(['waiting-task']);
  });

  it('rejects all eviction callers on timeout and permits a later request', async () => {
    const evictions = Promise.allSettled([
      client.durableListener.sendEvictInvocation('waiting-task', 1),
      client.durableListener.sendEvictInvocation('waiting-task', 1),
    ]);
    await jest.advanceTimersByTimeAsync(30_000);
    expect(await evictions).toEqual([
      { status: 'rejected', reason: expect.any(Error) },
      { status: 'rejected', reason: expect.any(Error) },
    ]);

    acknowledgement.release();
    await jest.advanceTimersByTimeAsync(0);
    await expect(
      client.durableListener.sendEvictInvocation('waiting-task', 1)
    ).resolves.toBeUndefined();
    expect(requests).toEqual(['waiting-task', 'waiting-task']);
  });

  it('rejects all eviction callers when the listener stops', async () => {
    const evictions = Promise.allSettled([
      client.durableListener.sendEvictInvocation('waiting-task', 1),
      client.durableListener.sendEvictInvocation('waiting-task', 1),
    ]);
    await client.durableListener.stop();
    await jest.advanceTimersByTimeAsync(30_000);

    expect(await evictions).toEqual([
      { status: 'rejected', reason: new Error('DurableListener stopped') },
      { status: 'rejected', reason: new Error('DurableListener stopped') },
    ]);
  });

  it.each(['signal-first', 'stop-first'])('drains waiting runs once: %s', async (order) => {
    const stopListener = jest.spyOn(client.durableListener, 'stop');
    for (const task of ['first-task', 'second-task']) {
      manager.registerRun(`${task}/0`, task, 1, undefined);
      manager.markWaiting(`${task}/0`, 'sleep', 'test-sleep');
    }
    const signalShutdowns = () =>
      [...signalListeners].flatMap(([signal, previous]) =>
        process
          .listeners(signal)
          .filter((listener) => !previous.includes(listener))
          .map((listener) => listener(signal))
      );
    const shutdowns =
      order === 'stop-first'
        ? [worker.stop(), ...signalShutdowns()]
        : [...signalShutdowns(), worker.stop()];
    const stopped = Promise.all(shutdowns);
    await jest.advanceTimersByTimeAsync(0);
    expect(cancelled).toEqual([]);
    expect(stopListener).not.toHaveBeenCalled();

    acknowledgement.release();
    await jest.advanceTimersByTimeAsync(60_000);
    await stopped;
    await worker.stop();

    expect(requests).toEqual(['first-task', 'second-task']);
    expect(cancelled).toEqual(['first-task/0', 'second-task/0']);
    expect(manager.cache.getAllWaiting()).toEqual([]);
    expect(stopListener).toHaveBeenCalledTimes(1);
  });
});
