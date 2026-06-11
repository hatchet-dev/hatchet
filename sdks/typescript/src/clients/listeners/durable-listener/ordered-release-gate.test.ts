import { NonDeterminismError } from '@hatchet/util/errors/non-determinism-error';
import { DurableListenerClient } from './durable-listener-client';

function noopLogger() {
  return { info: jest.fn(), warn: jest.fn(), error: jest.fn(), debug: jest.fn() };
}

function mockConfig(): any {
  return { logger: () => noopLogger(), log_level: 'OFF' };
}

function makeListener(): DurableListenerClient {
  const factory: any = { create: jest.fn(() => ({})) };
  return new DurableListenerClient(mockConfig(), {} as any, factory);
}

function entryCompleted(
  taskId: string,
  invocation: number,
  branchId: number,
  nodeId: number,
  payload: Record<string, unknown> = {},
  satisfiedOrder?: number
) {
  return {
    entryCompleted: {
      ref: {
        durableTaskExternalId: taskId,
        invocationCount: invocation,
        branchId,
        nodeId,
      },
      payload: new TextEncoder().encode(JSON.stringify(payload)),
      satisfiedOrder,
    },
  };
}

/** Registers a parked waiter the same way waitForCallback does. */
function register(
  listener: DurableListenerClient,
  taskId: string,
  invocation: number,
  branchId: number,
  nodeId: number
) {
  const l = listener as any;
  const key = `${taskId}:${invocation}:${branchId}:${nodeId}`;

  let resolved: any;
  let rejected: any;
  const promise = new Promise((resolve, reject) => {
    l._pendingCallbacks.set(key, {
      promise: undefined,
      resolve: (v: any) => {
        resolved = v;
        resolve(v);
      },
      reject: (e: any) => {
        rejected = e;
        reject(e);
      },
    });
  });
  promise.catch(() => undefined);
  l._notifyParked(`${taskId}:${invocation}`);

  return {
    promise,
    isResolved: () => resolved !== undefined,
    value: () => resolved,
    error: () => rejected,
  };
}

function handle(listener: DurableListenerClient, response: any) {
  (listener as any)._handleResponse(response);
}

describe('DurableListenerClient ordered-release gate', () => {
  let listener: DurableListenerClient;

  beforeEach(() => {
    listener = makeListener();
  });

  afterEach(async () => {
    await listener.stop();
  });

  it('holds out-of-order completions until the earlier order arrives (A->B / C->D)', () => {
    const waiterA = register(listener, 'task', 1, 1, 1);
    const waiterC = register(listener, 'task', 1, 1, 2);

    // A's completion was stamped second but arrives first: held.
    handle(listener, entryCompleted('task', 1, 1, 1, { r: 'a' }, 2));
    expect(waiterA.isResolved()).toBe(false);

    // C's completion (order 1) arrives: released, wakes C's continuation.
    handle(listener, entryCompleted('task', 1, 1, 2, { r: 'c' }, 1));
    expect(waiterC.isResolved()).toBe(true);
    expect(waiterC.value().payload).toEqual({ r: 'c' });

    // gate stays closed for order 2 until C's continuation parks.
    expect(waiterA.isResolved()).toBe(false);

    // C's continuation spawns D and parks on its result: order 2 released.
    const waiterD = register(listener, 'task', 1, 1, 3);
    expect(waiterA.isResolved()).toBe(true);
    expect(waiterA.value().payload).toEqual({ r: 'a' });
    expect(waiterD.isResolved()).toBe(false);
  });

  it('keeps pumping past releases with no waiter (sequential await deadlock avoidance)', () => {
    // the only parked waiter awaits the entry satisfied at order 2.
    const waiter2 = register(listener, 'task', 1, 1, 1);

    handle(listener, entryCompleted('task', 1, 1, 2, { r: 'c' }, 1));
    handle(listener, entryCompleted('task', 1, 1, 1, { r: 'a' }, 2));

    // order 1 had no waiter -> buffered; order 2 released to the waiter.
    expect(waiter2.isResolved()).toBe(true);
    expect(waiter2.value().payload).toEqual({ r: 'a' });
    expect((listener as any)._bufferedCompletions.has('task:1:1:2')).toBe(true);
  });

  it('bypasses the gate for re-delivered completions', () => {
    const waiter1 = register(listener, 'task', 1, 1, 1);
    handle(listener, entryCompleted('task', 1, 1, 1, { r: 'first' }, 1));
    expect(waiter1.isResolved()).toBe(true);

    // gate is closed, but a re-delivery of order 1 is delivered immediately.
    const waiterRetry = register(listener, 'task', 1, 1, 1);
    handle(listener, entryCompleted('task', 1, 1, 1, { r: 'first' }, 1));
    expect(waiterRetry.isResolved()).toBe(true);
  });

  it('releases legacy completions with no satisfied order immediately', () => {
    const waiterOrdered = register(listener, 'task', 1, 1, 1);
    const waiterLegacy = register(listener, 'task', 1, 1, 9);

    // ordered completion with a gap (order 2, order 1 missing): held.
    handle(listener, entryCompleted('task', 1, 1, 1, { r: 'ordered' }, 2));
    expect(waiterOrdered.isResolved()).toBe(false);

    // legacy completion: delivered immediately.
    handle(listener, entryCompleted('task', 1, 1, 9, { r: 'legacy' }));
    expect(waiterLegacy.isResolved()).toBe(true);
  });

  it('scopes gates per invocation', () => {
    const waiterInv1 = register(listener, 'task', 1, 1, 1);
    const waiterInv2 = register(listener, 'task', 2, 1, 1);

    // invocation 1 is blocked on a gap.
    handle(listener, entryCompleted('task', 1, 1, 1, { r: 'inv1' }, 2));
    expect(waiterInv1.isResolved()).toBe(false);

    // invocation 2's order 1 releases independently.
    handle(listener, entryCompleted('task', 2, 1, 1, { r: 'inv2' }, 1));
    expect(waiterInv2.isResolved()).toBe(true);
  });

  it('fails the invocation waiters with a NonDeterminismError on gap timeout', () => {
    (listener as any)._gapTimeoutMs = -1;

    const waiter = register(listener, 'task', 1, 1, 1);

    // order 2 arrives, order 1 never does (history diverged).
    handle(listener, entryCompleted('task', 1, 1, 1, { r: 'stranded' }, 2));
    expect(waiter.isResolved()).toBe(false);

    (listener as any)._sweepGates();

    expect(waiter.error()).toBeInstanceOf(NonDeterminismError);
    expect(waiter.error().taskExternalId).toBe('task');
    expect(waiter.error().invocationCount).toBe(1);
    expect(waiter.error().nodeId).toBe(1);
    expect((listener as any)._gates.size).toBe(0);
  });

  it('forces the gate open on park timeout', () => {
    (listener as any)._parkTimeoutMs = -1;

    const waiter1 = register(listener, 'task', 1, 1, 1);
    const waiter2 = register(listener, 'task', 1, 1, 2);

    handle(listener, entryCompleted('task', 1, 1, 1, { r: 'first' }, 1));
    expect(waiter1.isResolved()).toBe(true);

    handle(listener, entryCompleted('task', 1, 1, 2, { r: 'second' }, 2));
    expect(waiter2.isResolved()).toBe(false);

    // the woken continuation never parks: park timeout forces the gate open.
    (listener as any)._sweepGates();
    expect(waiter2.isResolved()).toBe(true);
  });

  it('drops gate state on cleanupTaskState', () => {
    handle(listener, entryCompleted('task', 1, 1, 1, { r: 'stranded' }, 2));
    expect((listener as any)._gates.size).toBe(1);

    listener.cleanupTaskState('task', 1);
    expect((listener as any)._gates.size).toBe(0);
  });
});
