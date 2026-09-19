/**
 * The SDK's `DurableTransport` over websocket frames. A worker implements the same seam
 * with `DurableListenerClient` over gRPC; here every durable event becomes a
 * `{ request }` frame with a sequence id, acks come back as `{ response }` frames, and
 * `entryCompleted` responses resolve the callback registered for their `(branchId, nodeId)`
 * in delivery order, the way the SDK's ordered-completion queue does.
 *
 * The operator allows one ack-bearing request in flight; requests queue here so a task
 * that fans out never trips that rule. The inline wait budget is measured from the ack
 * that created an entry to the moment its completion is awaited past the budget; when it
 * elapses the transport tells the invocation, which evicts.
 */
import {
  NonDeterminismError,
  TaskRunTerminatedError,
  createAbortError,
} from '@hatchet-dev/typescript-sdk/edge/index.js';
import type {
  DurableTaskEventAck,
  DurableTaskEventLogEntryResult,
  DurableTaskEventMemoAck,
  DurableTaskEventRunAck,
  DurableTaskEventWaitForAck,
  DurableTaskSendEvent,
  DurableTransport,
  MemoEvent,
  RunChildrenEvent,
  WaitForEvent,
} from '@hatchet-dev/typescript-sdk/edge/index.js';
import {
  DurableTaskErrorType,
  type DurableEventLogEntryRef,
  type DurableTaskEventLogEntryCompletedResponse,
  type DurableTaskRequest,
  type DurableTaskResponse,
} from '../../generated/proto/v1/dispatcher';
import type { ServerlessDurableFrame } from '../../generated/proto/v1/serverless';
import { ERROR_CODE_NON_DETERMINISM } from '../contract';
import { encodeFrame } from './frames';
import type { DurableSocket } from './socket';

interface Deferred<T> {
  promise: Promise<T>;
  resolve: (value: T) => void;
  reject: (reason: unknown) => void;
}

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  // A deferred can outlive its waiters (eviction aborts the caller while the entry stays
  // registered); a later rejection must not surface as an unhandled rejection.
  promise.catch(() => {});
  return { promise, resolve, reject };
}

export interface FrameTransportOptions {
  socket: DurableSocket;
  durableTaskExternalId: string;
  invocationCount: number;
  inlineWaitBudgetMs: number;
  /** Called once when an awaited completion outlives the inline budget. */
  onBudgetElapsed: (reason: string) => void;
  /** Called when the operator reports an engine-side error for the invocation. */
  onEngineError: (error: Error) => void;
  /** Called when the engine superseded the invocation. */
  onServerEvict: (reason: string) => void;
}

function callbackKey(branchId: number, nodeId: number): string {
  return `${branchId}:${nodeId}`;
}

function entryResult(
  completed: DurableTaskEventLogEntryCompletedResponse
): DurableTaskEventLogEntryResult {
  let payload: Record<string, unknown> | undefined;

  if (!completed.isFailure && completed.payload && completed.payload.length > 0) {
    payload = JSON.parse(new TextDecoder().decode(completed.payload));
  }

  return {
    durableTaskExternalId: completed.ref?.durableTaskExternalId ?? '',
    nodeId: completed.ref?.nodeId ?? 0,
    payload,
    isFailure: completed.isFailure,
    errorMessage: completed.errorMessage,
  };
}

export class FrameTransport implements DurableTransport {
  private seq = 0;
  private inFlight: Deferred<DurableTaskResponse> | undefined;
  private queue: Array<{ request: DurableTaskRequest; deferred: Deferred<DurableTaskResponse> }> =
    [];

  private callbacks = new Map<string, Deferred<DurableTaskEventLogEntryResult>>();
  private completions: Array<{ key: string; result: DurableTaskEventLogEntryResult }> = [];
  private delivered = new Set<string>();
  private draining = false;

  private ackedAt = new Map<string, number>();
  private budgetTimers = new Map<string, ReturnType<typeof setTimeout>>();
  private budgetElapsed = false;

  private sealed = false;
  private failure: Error | undefined;

  constructor(private readonly options: FrameTransportOptions) {}

  /** Nothing may be sent once the terminal frame went out or the socket closed. */
  seal(): void {
    this.sealed = true;
  }

  get isSealed(): boolean {
    return this.sealed;
  }

  /** Routes a frame from the operator, anything but the first one. */
  handleFrame(frame: ServerlessDurableFrame): void {
    if (frame.error) {
      const { code, message } = frame.error;
      const error =
        code === ERROR_CODE_NON_DETERMINISM
          ? new NonDeterminismError(
              this.options.durableTaskExternalId,
              this.options.invocationCount,
              0,
              message
            )
          : new Error(
              `durable task error from the engine: ${message} (code ${code || 'unspecified'})`
            );

      this.fail(error);
      this.options.onEngineError(error);
      return;
    }

    if (frame.response) {
      this.handleResponse(frame.response);
    }
  }

  private handleResponse(response: DurableTaskResponse): void {
    if (response.error) {
      const { error } = response;
      const { ref } = error;
      const exc =
        error.errorType === DurableTaskErrorType.DURABLE_TASK_ERROR_TYPE_NONDETERMINISM
          ? new NonDeterminismError(
              ref?.durableTaskExternalId ?? this.options.durableTaskExternalId,
              ref?.invocationCount ?? this.options.invocationCount,
              ref?.nodeId ?? 0,
              error.errorMessage
            )
          : new Error(`durable task error from the engine: ${error.errorMessage}`);

      this.fail(exc);
      this.options.onEngineError(exc);
      return;
    }

    if (response.serverEvict) {
      this.options.onServerEvict(response.serverEvict.reason);
      return;
    }

    if (response.entryCompleted) {
      this.handleCompletion(response.entryCompleted);
      return;
    }

    if (
      response.memoAck ||
      response.waitForAck ||
      response.triggerRunsAck ||
      response.evictionAck
    ) {
      const pending = this.inFlight;

      if (!pending) {
        return;
      }

      this.inFlight = undefined;
      pending.resolve(response);
      this.pump();
    }
  }

  private handleCompletion(completed: DurableTaskEventLogEntryCompletedResponse): void {
    const key = callbackKey(completed.ref?.branchId ?? 0, completed.ref?.nodeId ?? 0);
    const result = entryResult(completed);

    if (!this.delivered.has(key)) {
      this.delivered.add(key);
      this.completions.push({ key, result });
    } else if (this.completions.every((entry) => entry.key !== key)) {
      // A re-delivered completion that already drained; hand it straight to a waiter.
      const waiter = this.callbacks.get(key);

      if (waiter) {
        this.settleCallback(key, waiter, result);
      }
    }

    this.drain();
  }

  /**
   * Hands queued completions to their waiters strictly in delivery order, stopping at the
   * first one whose waiter has not registered yet, with a microtask between releases so a
   * resumed waiter reaches its next durable event before the following completion lands.
   */
  private drain(): void {
    if (this.draining) {
      return;
    }

    this.draining = true;

    const step = (): void => {
      const [head] = this.completions;
      const waiter = head ? this.callbacks.get(head.key) : undefined;

      if (!head || !waiter) {
        this.draining = false;
        return;
      }

      this.completions.shift();
      this.settleCallback(head.key, waiter, head.result);
      queueMicrotask(step);
    };

    step();
  }

  private settleCallback(
    key: string,
    waiter: Deferred<DurableTaskEventLogEntryResult>,
    result: DurableTaskEventLogEntryResult
  ): void {
    this.callbacks.delete(key);
    this.clearBudget(key);
    waiter.resolve(result);
  }

  private send(frame: ServerlessDurableFrame): void {
    if (this.sealed) {
      throw new Error('the durable socket is closed; no more frames can be sent');
    }

    this.options.socket.send(encodeFrame(frame));
  }

  private sendRequest(request: DurableTaskRequest): Promise<DurableTaskResponse> {
    const d = deferred<DurableTaskResponse>();

    if (this.failure) {
      d.reject(this.failure);
      return d.promise;
    }

    this.queue.push({ request, deferred: d });
    this.pump();

    return d.promise;
  }

  private pump(): void {
    while (!this.inFlight) {
      const next = this.queue.shift();

      if (!next) {
        return;
      }

      this.inFlight = next.deferred;
      this.seq += 1;

      try {
        this.send({ request: next.request, id: this.seq });
      } catch (err) {
        this.inFlight = undefined;
        next.deferred.reject(err);
      }
    }
  }

  private assertInvocation(durableTaskExternalId: string, invocationCount: number): void {
    if (
      durableTaskExternalId !== this.options.durableTaskExternalId ||
      invocationCount !== this.options.invocationCount
    ) {
      throw new Error(
        `durable event for task ${durableTaskExternalId} invocation ${invocationCount} on a socket serving task ${this.options.durableTaskExternalId} invocation ${this.options.invocationCount}`
      );
    }
  }

  private markAcked(ref: DurableEventLogEntryRef | undefined): void {
    if (ref) {
      this.ackedAt.set(callbackKey(ref.branchId, ref.nodeId), Date.now());
    }
  }

  sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: RunChildrenEvent
  ): Promise<DurableTaskEventRunAck>;
  sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: WaitForEvent
  ): Promise<DurableTaskEventWaitForAck>;
  sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: MemoEvent
  ): Promise<DurableTaskEventMemoAck>;
  async sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: DurableTaskSendEvent
  ): Promise<DurableTaskEventAck> {
    this.assertInvocation(durableTaskExternalId, invocationCount);

    switch (event.kind) {
      case 'runChildren': {
        const response = await this.sendRequest({
          triggerRuns: { invocationCount, durableTaskExternalId, triggerOpts: event.triggerOpts },
        });
        const ack = response.triggerRunsAck;

        if (!ack) {
          throw new Error('the operator answered trigger_runs with something other than its ack');
        }

        const now = Date.now();

        for (const entry of ack.runEntries) {
          this.ackedAt.set(callbackKey(entry.branchId, entry.nodeId), now);
        }

        return {
          ackType: 'run',
          invocationCount: ack.invocationCount,
          durableTaskExternalId: ack.durableTaskExternalId,
          runEntries: ack.runEntries.map((entry) => ({
            nodeId: entry.nodeId,
            branchId: entry.branchId,
            workflowRunExternalId: entry.workflowRunExternalId,
          })),
        };
      }

      case 'waitFor': {
        const response = await this.sendRequest({
          waitFor: {
            invocationCount,
            durableTaskExternalId,
            waitForConditions: event.waitForConditions,
            label: event.label,
          },
        });
        const ref = response.waitForAck?.ref;

        if (!ref) {
          throw new Error('the operator answered wait_for with something other than its ack');
        }

        this.markAcked(ref);

        return {
          ackType: 'waitFor',
          invocationCount: ref.invocationCount,
          durableTaskExternalId: ref.durableTaskExternalId,
          branchId: ref.branchId,
          nodeId: ref.nodeId,
        };
      }

      case 'memo': {
        const response = await this.sendRequest({
          memo: {
            invocationCount,
            durableTaskExternalId,
            key: event.memoKey,
            payload: event.payload,
          },
        });
        const ack = response.memoAck;

        if (!ack?.ref) {
          throw new Error('the operator answered memo with something other than its ack');
        }

        this.markAcked(ack.ref);

        return {
          ackType: 'memo',
          invocationCount: ack.ref.invocationCount,
          durableTaskExternalId: ack.ref.durableTaskExternalId,
          branchId: ack.ref.branchId,
          nodeId: ack.ref.nodeId,
          memoAlreadyExisted: ack.memoAlreadyExisted,
          memoResultPayload: ack.memoResultPayload,
        };
      }

      default: {
        const unknown: never = event;
        throw new Error(`unknown durable event ${JSON.stringify(unknown)}`);
      }
    }
  }

  waitForCallback(
    durableTaskExternalId: string,
    invocationCount: number,
    branchId: number,
    nodeId: number,
    opts?: { signal?: AbortSignal }
  ): Promise<DurableTaskEventLogEntryResult> {
    const signal = opts?.signal;

    if (signal?.aborted) {
      return Promise.reject(createAbortError('Operation cancelled by AbortSignal'));
    }

    if (this.failure) {
      return Promise.reject(this.failure);
    }

    const key = callbackKey(branchId, nodeId);
    let d = this.callbacks.get(key);

    if (!d) {
      d = deferred<DurableTaskEventLogEntryResult>();
      this.callbacks.set(key, d);
      this.drain();
    }

    if (this.callbacks.has(key)) {
      this.armBudget(key);
    }

    if (!signal) {
      return d.promise;
    }

    return new Promise<DurableTaskEventLogEntryResult>((resolve, reject) => {
      let settled = false;

      const onAbort = () => {
        if (settled) return;
        settled = true;
        reject(createAbortError('Operation cancelled by AbortSignal'));
      };

      signal.addEventListener('abort', onAbort, { once: true });

      d.promise.then(
        (value) => {
          if (settled) return;
          settled = true;
          signal.removeEventListener('abort', onAbort);
          resolve(value);
        },
        (err) => {
          if (settled) return;
          settled = true;
          signal.removeEventListener('abort', onAbort);
          reject(err);
        }
      );
    });
  }

  /**
   * Starts the inline budget for an awaited entry, counted from its ack. A budget of zero
   * or less means no inline waiting at all: the invocation evicts as soon as it waits.
   */
  private armBudget(key: string): void {
    if (this.budgetTimers.has(key) || this.budgetElapsed || this.sealed) {
      return;
    }

    const budget = this.options.inlineWaitBudgetMs;
    const acked = this.ackedAt.get(key) ?? Date.now();
    const remaining = Math.max(0, budget - (Date.now() - acked));

    const timer = setTimeout(() => {
      this.budgetTimers.delete(key);

      if (!this.callbacks.has(key) || this.budgetElapsed || this.sealed) {
        return;
      }

      this.budgetElapsed = true;
      this.options.onBudgetElapsed(`inline wait budget of ${budget}ms elapsed`);
    }, remaining);

    this.budgetTimers.set(key, timer);
  }

  private clearBudget(key: string): void {
    const timer = this.budgetTimers.get(key);

    if (timer !== undefined) {
      clearTimeout(timer);
      this.budgetTimers.delete(key);
    }
  }

  consumeCallbackWithoutBlocking(
    durableTaskExternalId: string,
    invocationCount: number,
    branchId: number,
    nodeId: number
  ): void {
    const key = callbackKey(branchId, nodeId);

    if (this.callbacks.has(key)) {
      return;
    }

    this.callbacks.set(key, deferred<DurableTaskEventLogEntryResult>());
    this.drain();
  }

  async sendMemoCompletedNotification(
    durableTaskExternalId: string,
    nodeId: number,
    branchId: number,
    invocationCount: number,
    memoKey: Uint8Array,
    memoResultPayload?: Uint8Array
  ): Promise<void> {
    this.assertInvocation(durableTaskExternalId, invocationCount);

    if (this.failure) {
      throw this.failure;
    }

    // complete_memo carries no ack, so it bypasses the one-in-flight queue.
    this.seq += 1;
    this.send({
      id: this.seq,
      request: {
        completeMemo: {
          ref: { durableTaskExternalId, invocationCount, branchId, nodeId },
          memoKey,
          payload: memoResultPayload ?? new Uint8Array(),
        },
      },
    });
  }

  cleanupTaskState(_durableTaskExternalId: string, _invocationCount: number): void {
    this.rejectPending(() => new TaskRunTerminatedError('evicted', 'task state cleaned up'));
  }

  async sendEvictInvocation(
    durableTaskExternalId: string,
    invocationCount: number,
    reason?: string
  ): Promise<void> {
    this.assertInvocation(durableTaskExternalId, invocationCount);

    const response = await this.sendRequest({
      evictInvocation: { invocationCount, durableTaskExternalId, reason },
    });

    if (!response.evictionAck) {
      throw new Error('the operator answered evict_invocation with something other than its ack');
    }
  }

  /** Rejects everything pending with the error and refuses further requests. */
  fail(error: Error): void {
    this.failure = error;
    this.rejectPending(() => error);
  }

  private rejectPending(error: () => Error): void {
    for (const timer of this.budgetTimers.values()) {
      clearTimeout(timer);
    }

    this.budgetTimers.clear();

    const { inFlight } = this;
    this.inFlight = undefined;
    inFlight?.reject(error());

    for (const queued of this.queue.splice(0)) {
      queued.deferred.reject(error());
    }

    for (const waiter of this.callbacks.values()) {
      waiter.reject(error());
    }

    this.callbacks.clear();
    this.completions = [];
  }

  /** Clears timers and pending state; the invocation is over. */
  dispose(): void {
    this.sealed = true;
    this.rejectPending(() => new Error('the durable invocation is over'));
    this.ackedAt.clear();
    this.delivered.clear();
  }
}
