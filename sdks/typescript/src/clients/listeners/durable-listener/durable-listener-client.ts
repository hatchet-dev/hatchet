import { EventEmitter, on } from 'events';
import { Channel, ClientFactory } from 'nice-grpc';
import { isAbortError } from 'abort-controller-x';
import { getErrorMessage } from '@hatchet/util/errors/hatchet-error';

import { ClientConfig } from '@clients/hatchet-client/client-config';
import { Logger } from '@hatchet/util/logger';
import {
  V1DispatcherClient,
  V1DispatcherDefinition,
  DurableTaskRequest,
  DurableTaskResponse,
  DurableTaskEventLogEntryCompletedResponse,
  DurableTaskErrorType,
  DurableTaskRequestRegisterWorker,
  DurableTaskWorkerStatusRequest,
  DurableTaskAwaitedCompletedEntry,
  DurableTaskEvictInvocationRequest,
  DurableTaskCompleteMemoRequest,
  DurableEvent,
  RegisterDurableEventResponse,
  ListenForDurableEventRequest,
  DurableTaskMemoRequest,
  DurableTaskTriggerRunsRequest,
  DurableTaskWaitForRequest,
  DurableEventLogEntryRef,
} from '@hatchet/protoc/v1/dispatcher';
import {
  DurableEventListenerConditions,
  SleepMatchCondition,
  UserEventMatchCondition,
} from '@hatchet/protoc/v1/shared/condition';
import { TriggerWorkflowRequest } from '@hatchet/protoc/v1/shared/trigger';
import { NonDeterminismError } from '@hatchet/util/errors/non-determinism-error';
import { createAbortError, bindAbortSignalHandler } from '@hatchet/util/abort-error';
import sleep from '@hatchet/util/sleep';

class TTLMap<K, V> {
  private cache = new Map<K, { value: V; expiresAt: number }>();
  private timer: ReturnType<typeof setInterval>;

  constructor(private ttlMs: number) {
    this.timer = setInterval(() => this.evict(), ttlMs);
  }

  set(key: K, value: V): void {
    this.cache.set(key, { value, expiresAt: Date.now() + this.ttlMs });
  }

  get(key: K): V | undefined {
    return this.cache.get(key)?.value;
  }

  get size(): number {
    return this.cache.size;
  }

  has(key: K): boolean {
    return this.cache.has(key);
  }

  delete(key: K): boolean {
    return this.cache.delete(key);
  }

  keys(): IterableIterator<K> {
    return this.cache.keys();
  }

  pop(key: K): V | undefined {
    const entry = this.cache.get(key);
    if (entry) {
      this.cache.delete(key);
      return entry.value;
    }
    return undefined;
  }

  destroy(): void {
    clearInterval(this.timer);
    this.cache.clear();
  }

  private evict(): void {
    const now = Date.now();
    for (const [key, entry] of this.cache) {
      if (entry.expiresAt <= now) {
        this.cache.delete(key);
      }
    }
  }
}

const DEFAULT_RECONNECT_INTERVAL = 3000;
const EVICTION_ACK_TIMEOUT_MS = 30_000;
const WORKER_STATUS_POLL_INTERVAL_MS = 1000;

export interface DurableTaskRunAckEntryResult {
  nodeId: number;
  branchId: number;
}

export interface DurableTaskEventRunAck {
  ackType: 'run';
  invocationCount: number;
  durableTaskExternalId: string;
  runEntries: DurableTaskRunAckEntryResult[];
}

export interface DurableTaskEventMemoAck {
  ackType: 'memo';
  invocationCount: number;
  durableTaskExternalId: string;
  branchId: number;
  nodeId: number;
  memoAlreadyExisted: boolean;
  memoResultPayload?: Uint8Array;
}

export interface DurableTaskEventWaitForAck {
  ackType: 'waitFor';
  invocationCount: number;
  durableTaskExternalId: string;
  branchId: number;
  nodeId: number;
}

export type DurableTaskEventAck =
  | DurableTaskEventRunAck
  | DurableTaskEventMemoAck
  | DurableTaskEventWaitForAck;

export interface DurableTaskEventLogEntryResult {
  durableTaskExternalId: string;
  nodeId: number;
  payload: Record<string, unknown> | undefined;
}

function eventLogEntryResultFromProto(
  proto: DurableTaskEventLogEntryCompletedResponse
): DurableTaskEventLogEntryResult {
  let payload: Record<string, unknown> | undefined;
  if (proto.payload && proto.payload.length > 0) {
    payload = JSON.parse(new TextDecoder().decode(proto.payload));
  }
  return {
    durableTaskExternalId: proto.ref?.durableTaskExternalId ?? '',
    nodeId: proto.ref?.nodeId ?? 0,
    payload,
  };
}

export interface WaitForEvent {
  kind: 'waitFor';
  waitForConditions: DurableEventListenerConditions;
}

export interface RunChildrenEvent {
  kind: 'runChildren';
  triggerOpts: TriggerWorkflowRequest[];
}

export interface MemoEvent {
  kind: 'memo';
  memoKey: Uint8Array;
  payload?: Uint8Array;
}

export type DurableTaskSendEvent = WaitForEvent | RunChildrenEvent | MemoEvent;

type TaskExternalId = string;
type InvocationCount = number;
type BranchId = number;
type NodeId = number;

type PendingEventAckKey = `${TaskExternalId}:${InvocationCount}`;
type PendingCallbackKey = `${TaskExternalId}:${InvocationCount}:${BranchId}:${NodeId}`;
type PendingEvictionAckKey = `${TaskExternalId}:${InvocationCount}`;

function ackKey(taskExtId: string, invocationCount: number): PendingEventAckKey {
  return `${taskExtId}:${invocationCount}`;
}
function callbackKey(
  taskExtId: string,
  invocationCount: number,
  branchId: number,
  nodeId: number
): PendingCallbackKey {
  return `${taskExtId}:${invocationCount}:${branchId}:${nodeId}`;
}
function evictionKey(taskExtId: string, invocationCount: number): PendingEvictionAckKey {
  return `${taskExtId}:${invocationCount}`;
}

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
  return { promise, resolve, reject };
}

export class DurableListenerClient {
  config: ClientConfig;
  client: V1DispatcherClient;
  logger: Logger;

  private _workerId: string | undefined;
  private _running = false;
  private _requestQueue: DurableTaskRequest[] = [];
  private _requestNotify: (() => void) | undefined;

  private _pendingEventAcks = new Map<PendingEventAckKey, Deferred<DurableTaskEventAck>>();
  private _pendingCallbacks = new Map<
    PendingCallbackKey,
    Deferred<DurableTaskEventLogEntryResult>
  >();
  // Completions that arrived before waitForCallback() registered a deferred
  // in _pendingCallbacks. This happens when the server delivers an
  // entryCompleted between the event ack and the waitForCallback call
  // (e.g. an already-satisfied sleep delivered via polling).
  private _bufferedCompletions = new TTLMap<PendingCallbackKey, DurableTaskEventLogEntryResult>(
    10_000
  );
  private _pendingEvictionAcks = new Map<PendingEvictionAckKey, Deferred<void>>();

  private _receiveAbort: AbortController | undefined;
  private _statusInterval: ReturnType<typeof setInterval> | undefined;
  private _startLock: Promise<void> | undefined;

  onServerEvict: ((durableTaskExternalId: string, invocationCount: number) => void) | undefined;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory) {
    this.config = config;
    this.client = factory.create(V1DispatcherDefinition, channel);
    this.logger = config.logger(`DurableListener`, config.log_level);
  }

  get workerId(): string | undefined {
    return this._workerId;
  }

  async start(workerId: string): Promise<void> {
    if (this._startLock) {
      await this._startLock;
      return;
    }
    this._startLock = this._doStart(workerId);
    await this._startLock;
  }

  private async _doStart(workerId: string): Promise<void> {
    if (this._running) return;
    this._workerId = workerId;
    this._running = true;
    await this._connect();
    this._startStatusPolling();
  }

  async ensureStarted(workerId: string): Promise<void> {
    if (!this._running) {
      await this.start(workerId);
    }
  }

  async stop(): Promise<void> {
    this._running = false;
    this._startLock = undefined;
    if (this._statusInterval) {
      clearInterval(this._statusInterval);
      this._statusInterval = undefined;
    }
    if (this._receiveAbort) {
      this._receiveAbort.abort();
    }
    this._failPendingAcks(new Error('DurableListener stopped'));
    this._bufferedCompletions.destroy();
  }

  private async _connect(): Promise<void> {
    this.logger.info('durable event listener connecting...');

    this._requestQueue = [];

    this._receiveAbort = new AbortController();

    this._enqueueRequest({
      registerWorker: { workerId: this._workerId! } as DurableTaskRequestRegisterWorker,
    });

    this._pollWorkerStatus();

    void this._streamLoop();

    this.logger.info('durable event listener connected');
  }

  private async _streamLoop(): Promise<void> {
    while (this._running) {
      try {
        const stream = this.client.durableTask(this._requestIterator(), {
          signal: this._receiveAbort?.signal,
        });

        for await (const response of stream) {
          this._handleResponse(response);
        }

        if (this._running) {
          this.logger.warn(
            `durable event listener disconnected (EOF), reconnecting in ${DEFAULT_RECONNECT_INTERVAL}ms...`
          );
          this._failPendingAcks(new Error('durable stream disconnected'));
          await sleep(DEFAULT_RECONNECT_INTERVAL);
          await this._connect();
          return;
        }
      } catch (e: unknown) {
        if (isAbortError(e)) {
          this.logger.debug('durable event listener aborted');
          return;
        }
        this.logger.error(`error in durable event listener: ${getErrorMessage(e)}`);
        if (this._running) {
          this._failPendingAcks(new Error(`durable stream error: ${getErrorMessage(e)}`));
          await sleep(DEFAULT_RECONNECT_INTERVAL);
          await this._connect();
          return;
        }
      }
    }
  }

  private async *_requestIterator(): AsyncIterable<DurableTaskRequest> {
    while (this._running) {
      while (this._requestQueue.length > 0) {
        yield this._requestQueue.shift()!;
      }

      await new Promise<void>((resolve) => {
        this._requestNotify = resolve;
      });
      this._requestNotify = undefined;
    }
  }

  private _enqueueRequest(request: DurableTaskRequest): void {
    this._requestQueue.push(request);
    if (this._requestNotify) {
      this._requestNotify();
    }
  }

  private _startStatusPolling(): void {
    if (this._statusInterval) {
      clearInterval(this._statusInterval);
    }
    this._statusInterval = setInterval(() => {
      this._pollWorkerStatus();
    }, WORKER_STATUS_POLL_INTERVAL_MS);
  }

  private _pollWorkerStatus(): void {
    if (!this._workerId || this._pendingCallbacks.size === 0) return;

    const waitingEntries: DurableTaskAwaitedCompletedEntry[] = [];
    for (const key of this._pendingCallbacks.keys()) {
      const parts = key.split(':');
      waitingEntries.push({
        durableTaskExternalId: parts[0],
        invocationCount: parseInt(parts[1], 10),
        branchId: parseInt(parts[2], 10),
        nodeId: parseInt(parts[3], 10),
      });
    }

    this._enqueueRequest({
      workerStatus: {
        workerId: this._workerId,
        waitingEntries,
      } as DurableTaskWorkerStatusRequest,
    });
  }

  private _failPendingAcks(exc: Error): void {
    for (const d of this._pendingEventAcks.values()) {
      d.reject(exc);
    }
    this._pendingEventAcks.clear();

    for (const d of this._pendingEvictionAcks.values()) {
      d.reject(exc);
    }
    this._pendingEvictionAcks.clear();
  }

  private _handleResponse(response: DurableTaskResponse): void {
    if (response.registerWorker) {
      // registration acknowledged
    } else if (response.triggerRunsAck) {
      const ack = response.triggerRunsAck;
      const key = ackKey(ack.durableTaskExternalId, ack.invocationCount);
      const pending = this._pendingEventAcks.get(key);
      if (pending) {
        pending.resolve({
          ackType: 'run',
          invocationCount: ack.invocationCount,
          durableTaskExternalId: ack.durableTaskExternalId,
          runEntries: (ack.runEntries || []).map((e) => ({
            nodeId: e.nodeId,
            branchId: e.branchId,
          })),
        });
        this._pendingEventAcks.delete(key);
      }
    } else if (response.memoAck) {
      const ack = response.memoAck;
      const { ref } = ack;
      const key = ackKey(ref?.durableTaskExternalId ?? '', ref?.invocationCount ?? 0);
      const pending = this._pendingEventAcks.get(key);
      if (pending) {
        pending.resolve({
          ackType: 'memo',
          invocationCount: ref?.invocationCount ?? 0,
          durableTaskExternalId: ref?.durableTaskExternalId ?? '',
          branchId: ref?.branchId ?? 0,
          nodeId: ref?.nodeId ?? 0,
          memoAlreadyExisted: ack.memoAlreadyExisted,
          memoResultPayload: ack.memoResultPayload,
        });
        this._pendingEventAcks.delete(key);
      }
    } else if (response.waitForAck) {
      const ack = response.waitForAck;
      const { ref } = ack;
      const key = ackKey(ref?.durableTaskExternalId ?? '', ref?.invocationCount ?? 0);
      const pending = this._pendingEventAcks.get(key);
      if (pending) {
        pending.resolve({
          ackType: 'waitFor',
          invocationCount: ref?.invocationCount ?? 0,
          durableTaskExternalId: ref?.durableTaskExternalId ?? '',
          branchId: ref?.branchId ?? 0,
          nodeId: ref?.nodeId ?? 0,
        });
        this._pendingEventAcks.delete(key);
      }
    } else if (response.entryCompleted) {
      const completed = response.entryCompleted;
      const { ref } = completed;
      const key = callbackKey(
        ref?.durableTaskExternalId ?? '',
        ref?.invocationCount ?? 0,
        ref?.branchId ?? 0,
        ref?.nodeId ?? 0
      );
      const result = eventLogEntryResultFromProto(completed);
      const pending = this._pendingCallbacks.get(key);
      if (pending) {
        pending.resolve(result);
        this._pendingCallbacks.delete(key);
      } else {
        this._bufferedCompletions.set(key, result);
      }
    } else if (response.evictionAck) {
      const ack = response.evictionAck;
      const key = evictionKey(ack.durableTaskExternalId, ack.invocationCount);
      const pending = this._pendingEvictionAcks.get(key);
      if (pending) {
        pending.resolve();
        this._pendingEvictionAcks.delete(key);
      }
    } else if (response.serverEvict) {
      const evict = response.serverEvict;
      this.logger.info(
        `received server eviction notification for task ${evict.durableTaskExternalId} ` +
          `invocation ${evict.invocationCount}: ${evict.reason}`
      );
      this.cleanupTaskState(evict.durableTaskExternalId, evict.invocationCount);
      if (this.onServerEvict) {
        this.onServerEvict(evict.durableTaskExternalId, evict.invocationCount);
      }
    } else if (response.error) {
      const { error } = response;
      const { ref } = error;
      let exc: Error;

      if (error.errorType === DurableTaskErrorType.DURABLE_TASK_ERROR_TYPE_NONDETERMINISM) {
        exc = new NonDeterminismError(
          ref?.durableTaskExternalId ?? '',
          ref?.invocationCount ?? 0,
          ref?.nodeId ?? 0,
          error.errorMessage
        );
      } else {
        exc = new Error(
          `Unspecified durable task error: ${error.errorMessage} (type: ${error.errorType})`
        );
      }

      const eAckKey = ackKey(ref?.durableTaskExternalId ?? '', ref?.invocationCount ?? 0);
      const pendingAck = this._pendingEventAcks.get(eAckKey);
      if (pendingAck) {
        pendingAck.reject(exc);
        this._pendingEventAcks.delete(eAckKey);
      }

      const eCbKey = callbackKey(
        ref?.durableTaskExternalId ?? '',
        ref?.invocationCount ?? 0,
        ref?.branchId ?? 0,
        ref?.nodeId ?? 0
      );
      const pendingCb = this._pendingCallbacks.get(eCbKey);
      if (pendingCb) {
        pendingCb.reject(exc);
        this._pendingCallbacks.delete(eCbKey);
      }

      const eEvKey = evictionKey(ref?.durableTaskExternalId ?? '', ref?.invocationCount ?? 0);
      const pendingEv = this._pendingEvictionAcks.get(eEvKey);
      if (pendingEv) {
        pendingEv.reject(exc);
        this._pendingEvictionAcks.delete(eEvKey);
      }
    }
  }

  async sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: RunChildrenEvent
  ): Promise<DurableTaskEventRunAck>;
  async sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: WaitForEvent
  ): Promise<DurableTaskEventWaitForAck>;
  async sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: MemoEvent
  ): Promise<DurableTaskEventMemoAck>;
  async sendEvent(
    durableTaskExternalId: string,
    invocationCount: number,
    event: DurableTaskSendEvent
  ): Promise<DurableTaskEventAck> {
    const key = ackKey(durableTaskExternalId, invocationCount);
    const d = deferred<DurableTaskEventAck>();
    this._pendingEventAcks.set(key, d);

    let request: DurableTaskRequest;

    switch (event.kind) {
      case 'runChildren': {
        const triggerRunsReq: DurableTaskTriggerRunsRequest = {
          invocationCount,
          durableTaskExternalId,
          triggerOpts: event.triggerOpts,
        };
        request = { triggerRuns: triggerRunsReq };
        break;
      }

      case 'waitFor': {
        const waitForReq: DurableTaskWaitForRequest = {
          invocationCount,
          durableTaskExternalId,
          waitForConditions: event.waitForConditions,
        };
        request = { waitFor: waitForReq };
        break;
      }

      case 'memo': {
        const memoReq: DurableTaskMemoRequest = {
          invocationCount,
          durableTaskExternalId,
          key: event.memoKey,
          payload: event.payload,
        };
        request = { memo: memoReq };
        break;
      }

      default: {
        const _: never = event;
        throw new Error(`Unknown durable task send event: ${_}`);
      }
    }

    this._enqueueRequest(request);
    return d.promise;
  }

  async waitForCallback(
    durableTaskExternalId: string,
    invocationCount: number,
    branchId: number,
    nodeId: number,
    opts?: { signal?: AbortSignal }
  ): Promise<DurableTaskEventLogEntryResult> {
    const key = callbackKey(durableTaskExternalId, invocationCount, branchId, nodeId);

    const early = this._bufferedCompletions.get(key);
    if (early) {
      this._bufferedCompletions.delete(key);
      return early;
    }

    if (!this._pendingCallbacks.has(key)) {
      this._pendingCallbacks.set(key, deferred<DurableTaskEventLogEntryResult>());
      this._pollWorkerStatus();
    }

    const d = this._pendingCallbacks.get(key)!;
    const signal = opts?.signal;

    if (!signal) {
      return d.promise;
    }

    if (signal.aborted) {
      return Promise.reject(createAbortError('Operation cancelled by AbortSignal'));
    }

    return new Promise<DurableTaskEventLogEntryResult>((resolve, reject) => {
      let settled = false;

      const onAbort = () => {
        if (settled) return;
        settled = true;
        reject(createAbortError('Operation cancelled by AbortSignal'));
      };

      bindAbortSignalHandler(signal, onAbort);

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

  cleanupTaskState(durableTaskExternalId: string, invocationCount: number): void {
    for (const [k, d] of this._pendingCallbacks) {
      const parts = k.split(':');
      if (parts[0] === durableTaskExternalId && parseInt(parts[1], 10) <= invocationCount) {
        d.reject(new Error('task state cleaned up'));
        this._pendingCallbacks.delete(k);
      }
    }

    for (const [k, d] of this._pendingEventAcks) {
      const parts = k.split(':');
      if (parts[0] === durableTaskExternalId && parseInt(parts[1], 10) <= invocationCount) {
        d.reject(new Error('task state cleaned up'));
        this._pendingEventAcks.delete(k);
      }
    }

    for (const k of this._bufferedCompletions.keys()) {
      const parts = k.split(':');
      if (parts[0] === durableTaskExternalId && parseInt(parts[1], 10) <= invocationCount) {
        this._bufferedCompletions.delete(k);
      }
    }
  }

  async sendEvictInvocation(
    durableTaskExternalId: string,
    invocationCount: number,
    reason?: string
  ): Promise<void> {
    const key = evictionKey(durableTaskExternalId, invocationCount);
    const d = deferred<void>();
    this._pendingEvictionAcks.set(key, d);

    const req: DurableTaskEvictInvocationRequest = {
      invocationCount,
      durableTaskExternalId,
      reason,
    };

    this._enqueueRequest({ evictInvocation: req });

    const timeout = sleep(EVICTION_ACK_TIMEOUT_MS).then(() => {
      throw new Error(
        `Eviction ack timed out after ${EVICTION_ACK_TIMEOUT_MS}ms for task ${durableTaskExternalId} invocation ${invocationCount}`
      );
    });

    try {
      await Promise.race([d.promise, timeout]);
    } catch (err) {
      this._pendingEvictionAcks.delete(key);
      throw err;
    }
  }

  async sendMemoCompletedNotification(
    durableTaskExternalId: string,
    nodeId: number,
    branchId: number,
    invocationCount: number,
    memoKey: Uint8Array,
    memoResultPayload?: Uint8Array
  ): Promise<void> {
    const ref: DurableEventLogEntryRef = {
      durableTaskExternalId,
      invocationCount,
      branchId,
      nodeId,
    };

    const req: DurableTaskCompleteMemoRequest = {
      ref,
      payload: memoResultPayload ?? new Uint8Array(),
      memoKey,
    };

    this._enqueueRequest({ completeMemo: req });
  }

  /**
   * @deprecated Legacy backward-compat: uses the old unary RegisterDurableEvent RPC.
   */
  async registerDurableEvent(request: {
    taskId: string;
    signalKey: string;
    sleepConditions: Array<SleepMatchCondition>;
    userEventConditions: Array<UserEventMatchCondition>;
  }): Promise<RegisterDurableEventResponse> {
    return this.client.registerDurableEvent({
      taskId: request.taskId,
      signalKey: request.signalKey,
      conditions: {
        sleepConditions: request.sleepConditions,
        userEventConditions: request.userEventConditions,
      },
    });
  }

  /**
   * @deprecated Legacy backward-compat: uses the old streaming ListenForDurableEvent RPC.
   */
  subscribe(request: { taskId: string; signalKey: string }): LegacyDurableEventStreamable {
    if (!this._legacyPooledListener) {
      this._legacyPooledListener = new LegacyPooledListener(this);
    }
    return this._legacyPooledListener.subscribe(request);
  }

  /**
   * @deprecated Legacy backward-compat: subscribes and waits for a single result.
   */
  async result(
    request: { taskId: string; signalKey: string },
    opts?: { signal?: AbortSignal }
  ): Promise<DurableEvent> {
    const subscriber = this.subscribe(request);
    return subscriber.get({ signal: opts?.signal });
  }

  private _legacyPooledListener: LegacyPooledListener | undefined;
}

/**
 * @deprecated Legacy support for the old streaming ListenForDurableEvent RPC.
 */
export class LegacyDurableEventStreamable {
  responseEmitter = new EventEmitter();
  private _onCleanup: () => void;

  constructor(onCleanup: () => void) {
    this._onCleanup = onCleanup;
  }

  async get(opts?: { signal?: AbortSignal }): Promise<DurableEvent> {
    const signal = opts?.signal;

    return new Promise((resolve, reject) => {
      let cleanedUp = false;

      const cleanup = () => {
        if (cleanedUp) return;
        cleanedUp = true;
        this.responseEmitter.removeListener('response', onResponse);
        if (signal) {
          signal.removeEventListener('abort', onAbort);
        }
        this._onCleanup();
      };

      const onResponse = (event: DurableEvent) => {
        cleanup();
        resolve(event);
      };

      const onAbort = () => {
        cleanup();
        reject(createAbortError('Operation cancelled by AbortSignal'));
      };

      if (signal?.aborted) {
        onAbort();
        return;
      }

      this.responseEmitter.once('response', onResponse);
      if (signal) {
        bindAbortSignalHandler(signal, onAbort);
      }
    });
  }
}

/**
 * @deprecated Legacy pooled listener for old ListenForDurableEvent streaming RPC.
 */
class LegacyPooledListener {
  private client: DurableListenerClient;
  private requestEmitter = new EventEmitter();
  private signal = new AbortController();
  private listener: AsyncIterable<DurableEvent> | undefined;
  private subscribers: Record<string, LegacyDurableEventStreamable> = {};
  private taskSignalKeyToSubscriptionIds: Record<string, string[]> = {};
  private subscriptionCounter = 0;
  private currRequester = 0;

  constructor(client: DurableListenerClient) {
    this.client = client;
    this.init();
  }

  private async init(retries = 0) {
    const MAX_RETRY_INTERVAL = 5000;
    const BASE_RETRY_INTERVAL = 100;
    const MAX_RETRY_COUNT = 5;

    if (retries > 0) {
      const backoffTime = Math.min(BASE_RETRY_INTERVAL * 2 ** (retries - 1), MAX_RETRY_INTERVAL);
      await sleep(backoffTime);
    }

    if (retries > MAX_RETRY_COUNT) return;

    try {
      this.signal = new AbortController();
      this.currRequester++;

      this.listener = this.client.client.listenForDurableEvent(this.request(), {
        signal: this.signal.signal,
      });

      for await (const event of this.listener) {
        const subscriptionKey = `${event.taskId}|${event.signalKey}`;
        const subscriptionIds = this.taskSignalKeyToSubscriptionIds[subscriptionKey] || [];
        for (const subId of subscriptionIds) {
          const emitter = this.subscribers[subId];
          if (emitter) {
            emitter.responseEmitter.emit('response', event);
            this.cleanupSubscription(subId);
          }
        }
      }
    } catch (e: unknown) {
      if (isAbortError(e)) return;
    } finally {
      if (Object.keys(this.subscribers).length > 0) {
        this.init(retries + 1);
      }
    }
  }

  private cleanupSubscription(subscriptionId: string) {
    const emitter = this.subscribers[subscriptionId];
    if (!emitter) return;
    const key = Object.entries(this.taskSignalKeyToSubscriptionIds).find(([, ids]) =>
      ids.includes(subscriptionId)
    )?.[0];
    delete this.subscribers[subscriptionId];
    if (key && this.taskSignalKeyToSubscriptionIds[key]) {
      this.taskSignalKeyToSubscriptionIds[key] = this.taskSignalKeyToSubscriptionIds[key].filter(
        (id) => id !== subscriptionId
      );
      if (this.taskSignalKeyToSubscriptionIds[key].length === 0) {
        delete this.taskSignalKeyToSubscriptionIds[key];
      }
    }
  }

  subscribe(request: { taskId: string; signalKey: string }): LegacyDurableEventStreamable {
    const subscriptionId = (this.subscriptionCounter++).toString();
    const subscriber = new LegacyDurableEventStreamable(() =>
      this.cleanupSubscription(subscriptionId)
    );
    this.subscribers[subscriptionId] = subscriber;

    const key = `${request.taskId}|${request.signalKey}`;
    if (!this.taskSignalKeyToSubscriptionIds[key]) {
      this.taskSignalKeyToSubscriptionIds[key] = [];
    }
    this.taskSignalKeyToSubscriptionIds[key].push(subscriptionId);
    this.requestEmitter.emit('subscribe', request);
    return subscriber;
  }

  private async *request(): AsyncIterable<ListenForDurableEventRequest> {
    const { currRequester } = this;
    const existing = new Set<string>();

    for (const key in this.taskSignalKeyToSubscriptionIds) {
      if (this.taskSignalKeyToSubscriptionIds[key].length > 0) {
        const [taskId, signalKey] = key.split('|');
        existing.add(key);
        yield { taskId, signalKey };
      }
    }

    for await (const e of on(this.requestEmitter, 'subscribe')) {
      if (currRequester !== this.currRequester) break;
      const request = e[0] as ListenForDurableEventRequest;
      const key = `${request.taskId}|${request.signalKey}`;
      if (!existing.has(key)) {
        existing.add(key);
        yield request;
      }
    }
  }
}
