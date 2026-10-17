import { DurableTaskResponse } from '@hatchet/protoc/v1/dispatcher';
import { ActionType, StepActionEventType } from '@hatchet/protoc/dispatcher';
import {
  ActionListener,
  ActionKey,
  createAction,
} from '@hatchet/clients/dispatcher/action-listener';
import { HatchetClient } from '../client';
import { TenantClient } from '../features/tenant';
import { DurableEvictionManager } from './eviction/eviction-manager';
import HatchetPromise from '@util/hatchet-promise/hatchet-promise';
import { InternalWorker } from './worker-internal';
import { HealthServer } from './health-server';
import { DurableContext } from './context';
import { MinEngineVersion } from './engine-version';

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
  let waitForAcknowledgement: ReturnType<typeof gate>;
  let requests: string[];
  let waitForRequests: string[];
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
    waitForAcknowledgement = gate();
    requests = [];
    waitForRequests = [];
    cancelled = [];
    // Evictions are answered in order once `acknowledgement` opens; a waitFor
    // registration is answered once `waitForAcknowledgement` opens, without
    // blocking the requests behind it, as the engine handles them concurrently.
    jest
      .spyOn(client.durableListener.client, 'durableTask')
      .mockImplementation(async function* (outgoing) {
        const responses: DurableTaskResponse[] = [];
        let wake: (() => void) | undefined;
        let requestsDone = false;
        const respond = (response: DurableTaskResponse) => {
          responses.push(response);
          wake?.();
        };
        void (async () => {
          for await (const request of outgoing) {
            if (request.evictInvocation) {
              requests.push(request.evictInvocation.durableTaskExternalId ?? '');
              await acknowledgement.promise;
              respond(DurableTaskResponse.fromPartial({ evictionAck: request.evictInvocation }));
            }
            if (request.triggerRuns) {
              // Child batches are acked at once and every child completes immediately.
              const { durableTaskExternalId, invocationCount } = request.triggerRuns;
              const runEntries = (request.triggerRuns.triggerOpts || []).map((_, i) => ({
                nodeId: 10 + i,
                branchId: 0,
                workflowRunExternalId: `child-${i}`,
              }));
              respond(
                DurableTaskResponse.fromPartial({
                  triggerRunsAck: { durableTaskExternalId, invocationCount, runEntries },
                })
              );
              for (const entry of runEntries) {
                respond(
                  DurableTaskResponse.fromPartial({
                    entryCompleted: {
                      ref: {
                        durableTaskExternalId,
                        invocationCount,
                        nodeId: entry.nodeId,
                        branchId: 0,
                      },
                      payload: new TextEncoder().encode('{}'),
                      isFailure: false,
                    },
                  })
                );
              }
            }
            if (request.waitFor) {
              const { durableTaskExternalId, invocationCount, label } = request.waitFor;
              waitForRequests.push(durableTaskExternalId ?? '');
              // A wait labelled `never-acked` models a registration the engine never answers.
              if (label === 'never-acked') continue;
              void waitForAcknowledgement.promise.then(() =>
                respond(
                  DurableTaskResponse.fromPartial({
                    waitForAck: {
                      ref: { durableTaskExternalId, invocationCount, branchId: 0, nodeId: 1 },
                    },
                  })
                )
              );
            }
          }
          requestsDone = true;
          wake?.();
        })();
        while (!requestsDone || responses.length > 0) {
          while (responses.length > 0) {
            yield responses.shift()!;
          }
          if (requestsDone) break;
          await new Promise<void>((resolve) => {
            wake = resolve;
          });
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
    waitForAcknowledgement.release();
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

  // Mirrors handleStartStepRun: the future leaves `worker.futures` as soon as its cancelable
  // wrapper settles, which on cancellation is before the underlying work finishes.
  function registerRunningTask(key: ActionKey) {
    const work = gate();
    const future = new HatchetPromise<void>(work.promise);
    future.promise
      .catch(() => undefined)
      .finally(() => {
        delete worker.futures[key];
      });
    worker.futures[key] = future;
    return { future, finish: work.release };
  }

  // Mirrors a dispatched durable action: the worker wires its own eviction manager
  // (with the real local-cancel path) and runs `fn` as the task body.
  function startDurableTask(
    taskRunExternalId: string,
    fn: (ctx: DurableContext<any, any>) => Promise<unknown>
  ) {
    const actionId = 'shutdown-test:durable-task';
    worker.engineVersion = MinEngineVersion.DURABLE_EVICTION;
    worker.durable_action_set.add(actionId);
    worker.action_registry[actionId] = fn as any;
    worker.evictionManager = undefined;
    const sendStepActionEvent = jest
      .spyOn(client.dispatcher, 'sendStepActionEvent')
      .mockResolvedValue(undefined as never);
    const action = createAction({
      tenantId: 'tenant',
      workflowRunId: 'workflow-run',
      getGroupKeyRunId: '',
      jobId: 'job',
      jobName: 'shutdown-test',
      jobRunId: 'job-run',
      taskId: 'task',
      taskRunExternalId,
      actionId,
      actionType: ActionType.START_STEP_RUN,
      actionPayload: '{}',
      taskName: 'durable-task',
      retryCount: 0,
      priority: 1,
      durableTaskInvocationCount: 1,
    });
    const running = worker.handleStartStepRun(action);
    const reportedEventTypes = () =>
      sendStepActionEvent.mock.calls.map(([event]) => event.eventType);
    return { running, reportedEventTypes };
  }

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
    const inFlightTask = registerRunningTask('in-flight-task/0' as ActionKey);

    const starting = worker.start();
    await jest.advanceTimersByTimeAsync(0);

    const stopping = worker.stop();
    await jest.advanceTimersByTimeAsync(60_000);

    expect(stopStream).not.toHaveBeenCalled();
    expect(stopHeartbeat).not.toHaveBeenCalled();
    expect(unsubscribe).not.toHaveBeenCalled();

    inFlightTask.finish();
    await stopping;
    await starting;

    expect(stopStream).toHaveBeenCalledTimes(1);
    expect(stopHeartbeat).toHaveBeenCalledTimes(1);
    expect(unsubscribe).toHaveBeenCalledTimes(1);
  });

  it('still unregisters when an in-flight task is cancelled during shutdown', async () => {
    const { unsubscribe } = registerOpenActionStream();
    const inFlightTask = registerRunningTask('in-flight-task/0' as ActionKey);

    const starting = worker.start();
    await jest.advanceTimersByTimeAsync(0);

    const stopping = worker.stop();
    await jest.advanceTimersByTimeAsync(0);

    inFlightTask.future.cancel();
    await jest.advanceTimersByTimeAsync(60_000);
    expect(worker.futures).toEqual({});
    expect(unsubscribe).not.toHaveBeenCalled();

    inFlightTask.finish();
    await expect(stopping).resolves.toBeUndefined();
    await starting;

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
    const inFlightTask = registerRunningTask('in-flight-task/0' as ActionKey);

    const starting = worker.start();
    await jest.advanceTimersByTimeAsync(0);

    let stopped = false;
    const stopping = worker.stop().then(() => {
      stopped = true;
    });
    await jest.advanceTimersByTimeAsync(60_000);
    expect(stopped).toBe(false);

    inFlightTask.finish();
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

  it('lets an in-flight durable registration settle, then evicts it instead of failing the run', async () => {
    const stopListener = jest.spyOn(client.durableListener, 'stop');
    const { running, reportedEventTypes } = startDurableTask('registering-task', (ctx) =>
      ctx.sleepFor('5m')
    );
    await jest.advanceTimersByTimeAsync(0);
    expect(waitForRequests).toEqual(['registering-task']);

    // SIGTERM lands while the waitFor ack is still pending: nothing is evicted and the
    // listener stays up until the ack arrives.
    const stopping = worker.stop();
    await jest.advanceTimersByTimeAsync(1_000);
    expect(requests).toEqual([]);
    expect(stopListener).not.toHaveBeenCalled();

    waitForAcknowledgement.release();
    acknowledgement.release();
    await jest.advanceTimersByTimeAsync(1_000);
    await stopping;
    await running;

    expect(requests).toEqual(['registering-task']);
    expect(reportedEventTypes()).toEqual([StepActionEventType.STEP_EVENT_TYPE_STARTED]);
    expect(worker.futures).toEqual({});
  });

  it('leaves a registration whose ack never arrives for reassignment once the shutdown bound elapses', async () => {
    const { running, reportedEventTypes } = startDurableTask('stuck-registration', (ctx) =>
      ctx.sleepFor('5m')
    );
    await jest.advanceTimersByTimeAsync(0);
    expect(waitForRequests).toEqual(['stuck-registration']);

    let stopped = false;
    const stopping = worker.stop().then(() => {
      stopped = true;
    });
    await jest.advanceTimersByTimeAsync(9_999);
    expect(stopped).toBe(false);

    // Not evicted (the engine may not have registered the wait's wake-up yet) and
    // not reported: the run stays assigned and the engine reassigns it when the
    // worker's heartbeat stops.
    await jest.advanceTimersByTimeAsync(1);
    await stopping;
    await running;

    expect(requests).toEqual([]);
    expect(reportedEventTypes()).toEqual([StepActionEventType.STEP_EVENT_TYPE_STARTED]);
    expect(worker.futures).toEqual({});
  });

  it('leaves a run with an acknowledged wait and a pending registration for reassignment, not eviction', async () => {
    const { running, reportedEventTypes } = startDurableTask('mixed-task', (ctx) =>
      Promise.all([
        ctx.sleepFor('5m', { label: 'acked' }),
        ctx.sleepFor('5m', { label: 'never-acked' }),
      ])
    );
    waitForAcknowledgement.release();
    await jest.advanceTimersByTimeAsync(0);
    // The first sleep is acknowledged and waiting; the serialized second send is pending forever.
    expect(waitForRequests).toEqual(['mixed-task', 'mixed-task']);

    const stopping = worker.stop();
    await jest.advanceTimersByTimeAsync(10_000);
    await stopping;
    await running;

    // Evicting would rely on the acknowledged wait to resume the run, but that wait can
    // complete before the eviction commits; the run is left assigned for reassignment instead.
    expect(requests).toEqual([]);
    expect(reportedEventTypes()).toEqual([StepActionEventType.STEP_EVENT_TYPE_STARTED]);
    expect(worker.futures).toEqual({});
  });

  it('rechecks a run before evicting it: a registration started while an earlier eviction ack was pending leaves it for reassignment', async () => {
    const first = startDurableTask('first-task', (ctx) => ctx.sleepFor('5m'));
    const second = startDurableTask('next-task', async (ctx) => {
      await ctx.sleepFor('1s');
      await ctx.sleepFor('5m', { label: 'never-acked' });
    });
    waitForAcknowledgement.release();
    await jest.advanceTimersByTimeAsync(0);
    expect(waitForRequests).toEqual(['first-task', 'next-task']);

    const stopping = worker.stop();
    await jest.advanceTimersByTimeAsync(0);
    expect(requests).toEqual(['first-task']);

    // While the first eviction ack is pending, the second run's sleep completes and
    // its task registers another sleep that the engine never acks.
    (client.durableListener as any)._handleResponse(
      DurableTaskResponse.fromPartial({
        entryCompleted: {
          ref: { durableTaskExternalId: 'next-task', invocationCount: 1, branchId: 0, nodeId: 1 },
          payload: new TextEncoder().encode('{}'),
        },
      })
    );
    await jest.advanceTimersByTimeAsync(0);
    // The mock reads requests in order and is parked on the first eviction, so the
    // new registration shows up in the cache first and on the stream once released.
    expect(
      worker.evictionManager!.cache.getAllRegistering().map((rec) => rec.taskRunExternalId)
    ).toEqual(['next-task']);

    acknowledgement.release();
    await jest.advanceTimersByTimeAsync(1_000);
    await stopping;
    await first.running;
    await second.running;

    expect(waitForRequests).toEqual(['first-task', 'next-task', 'next-task']);
    expect(requests).toEqual(['first-task']);
    expect(first.reportedEventTypes()).toEqual([
      StepActionEventType.STEP_EVENT_TYPE_STARTED,
      StepActionEventType.STEP_EVENT_TYPE_STARTED,
    ]);
    expect(worker.futures).toEqual({});
  });

  it('a completed child batch does not hide a concurrently pending sleep registration', async () => {
    const { running, reportedEventTypes } = startDurableTask('batch-task', (ctx) =>
      Promise.all([
        ctx.spawnChildren([
          { workflow: 'child', input: {} },
          { workflow: 'child', input: {} },
        ]),
        ctx.sleepFor('5m'),
      ])
    );
    await jest.advanceTimersByTimeAsync(0);
    // Both children have completed; the sleep's ack is still pending.
    expect(waitForRequests).toEqual(['batch-task']);

    const stopping = worker.stop();
    await jest.advanceTimersByTimeAsync(1_000);
    expect(requests).toEqual([]);

    waitForAcknowledgement.release();
    acknowledgement.release();
    await jest.advanceTimersByTimeAsync(1_000);
    await stopping;
    await running;

    expect(requests).toEqual(['batch-task']);
    expect(reportedEventTypes()).toEqual([StepActionEventType.STEP_EVENT_TYPE_STARTED]);
    expect(worker.futures).toEqual({});
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
