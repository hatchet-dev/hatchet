import { DurableTaskResponse } from '@hatchet/protoc/v1/dispatcher';
import { ActionListener } from '@hatchet/clients/dispatcher/action-listener';
import { HatchetClient } from '../client';
import { TenantClient } from '../features/tenant';
import { DurableEvictionManager } from './eviction/eviction-manager';
import { InternalWorker } from './worker-internal';
import { HealthServer } from './health-server';

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
        sub: 'shutdown-serialization-subject',
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

  function registerOpenActionStream() {
    const streamClosed = gate();
    const listener = new ActionListener(client.dispatcher, 'registered-worker');
    const stopStream = jest.spyOn(listener, 'stopStream').mockImplementation(() => {
      streamClosed.release();
    });
    const stopHeartbeat = jest.spyOn(listener.heartbeat, 'stop');
    const unsubscribe = jest
      .spyOn(listener.client, 'unsubscribe')
      .mockResolvedValue({ tenantId: client.tenantId, workerId: listener.workerId });
    jest.spyOn(listener, 'actions').mockImplementation(async function* () {
      await streamClosed.promise;
      yield* [];
    });
    jest.spyOn(client.dispatcher, 'getActionListener').mockResolvedValue(listener);
    jest.spyOn(client.workers, 'pause').mockResolvedValue(undefined as never);
    worker.action_registry['waiting-task'] = () => {};
    return { stopStream, stopHeartbeat, unsubscribe };
  }

  it('sets the worker id once the listener is registered', async () => {
    registerOpenActionStream();

    const starting = worker.start();
    await jest.advanceTimersByTimeAsync(0);

    expect(worker.workerId).toBe('registered-worker');

    await worker.stop();
    await starting;
  });

  it('closes the action stream once across repeated stops without exiting the process', async () => {
    const { stopStream, unsubscribe } = registerOpenActionStream();
    const exit = jest.spyOn(process, 'exit').mockImplementation(() => undefined as never);

    const starting = worker.start();
    await jest.advanceTimersByTimeAsync(0);
    expect(stopStream).not.toHaveBeenCalled();

    await worker.stop();
    await worker.stop();
    await starting;

    expect(worker.killing).toBe(true);
    expect(unsubscribe).toHaveBeenCalledTimes(1);
    expect(exit).not.toHaveBeenCalled();
  });

  it('keeps the action stream and heartbeat running until in-flight tasks finish, then unregisters', async () => {
    const { stopStream, stopHeartbeat, unsubscribe } = registerOpenActionStream();
    const inFlightTask = gate();
    worker.futures['in-flight-task/0'] = { promise: inFlightTask.promise } as any;

    const starting = worker.start();
    await jest.advanceTimersByTimeAsync(0);

    const stopping = worker.stop();
    await jest.advanceTimersByTimeAsync(60_000);

    expect(stopStream).not.toHaveBeenCalled();
    expect(stopHeartbeat).not.toHaveBeenCalled();
    expect(unsubscribe).not.toHaveBeenCalled();

    inFlightTask.release();
    await stopping;
    await starting;

    expect(stopStream).toHaveBeenCalledTimes(1);
    expect(stopHeartbeat).toHaveBeenCalledTimes(1);
    expect(unsubscribe).toHaveBeenCalledTimes(1);
  });

  it('stops the action stream without stopping the heartbeat', () => {
    const listener = new ActionListener(client.dispatcher, 'registered-worker');
    const stopHeartbeat = jest.spyOn(listener.heartbeat, 'stop');
    listener.abortController = new AbortController();

    listener.stopStream();

    expect(listener.done).toBe(true);
    expect(listener.abortController.signal.aborted).toBe(true);
    expect(stopHeartbeat).not.toHaveBeenCalled();
  });

  it('waits for in-flight tasks to finish before stop resolves', async () => {
    registerOpenActionStream();
    const inFlightTask = gate();
    worker.futures['in-flight-task/0'] = { promise: inFlightTask.promise } as any;

    const starting = worker.start();
    await jest.advanceTimersByTimeAsync(0);

    let stopped = false;
    const stopping = worker.stop().then(() => {
      stopped = true;
    });
    await jest.advanceTimersByTimeAsync(60_000);
    expect(stopped).toBe(false);

    inFlightTask.release();
    await stopping;
    await starting;

    expect(stopped).toBe(true);
  });

  it('unregisters a listener acquired after shutdown completes', async () => {
    const registration = gate();
    const listener = new ActionListener(client.dispatcher, 'late-worker');
    const unregister = jest
      .spyOn(listener, 'unregister')
      .mockResolvedValue({ tenantId: client.tenantId, workerId: listener.workerId });
    const actions = jest.spyOn(listener, 'actions').mockImplementation(async function* () {
      yield* [];
    });
    const getListener = jest
      .spyOn(client.dispatcher, 'getActionListener')
      .mockImplementation(async () => {
        await registration.promise;
        return listener;
      });
    worker.action_registry['waiting-task'] = () => {};

    const starting = worker.start();
    await jest.advanceTimersByTimeAsync(0);
    expect(getListener).toHaveBeenCalledTimes(1);
    await worker.stop();
    expect(unregister).not.toHaveBeenCalled();

    registration.release();
    await starting;
    await worker.stop();

    expect(unregister).toHaveBeenCalledTimes(1);
    expect(actions).not.toHaveBeenCalled();
  });

  it.each(['before-start', 'workflow-registration'])(
    'does not acquire a listener after stop: %s',
    async (phase) => {
      const registration = gate();
      const getListener = jest
        .spyOn(client.dispatcher, 'getActionListener')
        .mockRejectedValue(new Error('Unexpected listener registration'));
      worker.action_registry['waiting-task'] = () => {};
      worker.registeredWorkflowPromises.push(registration.promise);

      if (phase === 'before-start') {
        await worker.stop();
      }
      const starting = worker.start();
      await worker.stop();
      registration.release();
      await starting;

      expect(getListener).not.toHaveBeenCalled();
    }
  );

  it('closes a health server whose startup finishes after shutdown', async () => {
    const healthReady = gate();
    let healthListening = false;
    jest.spyOn(HealthServer.prototype, 'start').mockImplementation(async () => {
      await healthReady.promise;
      healthListening = true;
    });
    jest.spyOn(HealthServer.prototype, 'stop').mockImplementation(async () => {
      healthListening = false;
    });
    client.config.healthcheck = { enabled: true, port: 8001 };
    const startingWorker = new InternalWorker(client, {
      name: 'health-startup',
      handleKill: false,
    });
    const starting = startingWorker.start();
    await startingWorker.stop();

    healthReady.release();
    await starting;

    expect(healthListening).toBe(false);
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
