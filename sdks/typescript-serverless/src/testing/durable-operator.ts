/**
 * An in-memory serverless operator for durable invocations. It dials the handler through
 * the `DurableHooks.upgrade` seam with a signed upgrade, speaks the frame protocol of
 * pkg/serverlessoperator/durable over an in-memory socket pair, keeps a durable event log
 * per task run (memo entries replay, wait entries complete from a virtual clock or from
 * emitted events, child runs execute the registered task function) and enforces the
 * operator's protocol rules so a misbehaving endpoint fails the test.
 */
import { durationToMs, type Duration } from '@hatchet-dev/typescript-sdk/edge/index.js';
import { ActionType, AssignedAction } from '../generated/proto/dispatcher';
import type {
  DurableEventLogEntryRef,
  DurableTaskRequest,
  DurableTaskResponse,
} from '../generated/proto/v1/dispatcher';
import type {
  ServerlessDurableFrame,
  ServerlessFirstFrame,
} from '../generated/proto/v1/serverless';
import {
  DONE_STATUS_EVICTED,
  ENDPOINT_ID_HEADER,
  INVOCATION_HEADER,
  NONCE_HEADER,
  SIGNATURE_HEADER,
  TASK_ID_HEADER,
  TIMESTAMP_HEADER,
  stripNamespace,
  upgradeSigningPayload,
} from '../handler/contract';
import { decodeFrame, encodeFrame, frameKind } from '../handler/durable/frames';
import type { DurableSocket } from '../handler/durable/socket';
import type { ServerlessHandler } from '../handler';
import { signHex } from '../handler/signature';
import { createSocketPair } from './socket-pair';

export interface DurableInvokeOptions {
  /** How long the endpoint may wait inline before it evicts. Defaults to 5000. */
  inlineWaitBudgetMs?: number;
  /** The invocation count; `resume` passes the previous one plus one. Defaults to 1. */
  invocationCount?: number;
  taskRunExternalId?: string;
  workflowRunId?: string;
  retryCount?: number;
  additionalMetadata?: Record<string, string>;
  parents?: Record<string, unknown>;
  /** Rewrites the first frame before it is sent, to test the endpoint's identity checks. */
  firstFrame?: (frame: ServerlessFirstFrame) => ServerlessFirstFrame;
}

export interface RecordedFrame {
  from: 'endpoint' | 'operator';
  frame: ServerlessDurableFrame;
}

export interface DurableResult {
  status: 'completed' | 'evicted' | 'failed';
  output?: unknown;
  error?: string;
  retry?: boolean;
  /** The status of the upgrade response; 101 when the socket was accepted. */
  httpStatus: number;
  closeCode: number;
  closeReason: string;
  taskRunExternalId: string;
  invocationCount: number;
  /** Every frame in the order it crossed the socket. */
  frames: RecordedFrame[];
  /** The kinds of the frames the endpoint sent, in order (`done` carries its shape). */
  endpointFrames: string[];
  /** The kinds of the frames the operator sent, in order. */
  operatorFrames: string[];
}

/** A durable invocation in progress, for tests that act on the operator side mid-flight. */
export interface DurableRun {
  result: Promise<DurableResult>;
  taskRunExternalId: string;
  invocationCount: number;
  /** Resolves when the endpoint has sent its n-th frame of the kind (1-based). */
  frame(kind: string, occurrence?: number): Promise<ServerlessDurableFrame>;
  /** The engine superseded the invocation: serverEvict, then close 4001. */
  serverEvict(reason?: string): void;
  /** An engine-side error for the invocation. */
  sendError(code: 'nondeterminism' | 'unspecified', message: string): void;
  /** The operator closes the socket. */
  close(code: number, reason?: string): void;
}

export interface VirtualClock {
  now(): number;
  /** Moves the clock forward and completes the sleeps that came due. */
  advance(duration: Duration | number): void;
}

interface LogEntry {
  branchId: number;
  nodeId: number;
  kind: 'memo' | 'waitFor' | 'child';
  completed: boolean;
  payload?: Uint8Array;
  isFailure: boolean;
  errorMessage?: string;
  memoKey?: string;
  sleepDueAt?: number;
  eventKey?: string;
  readableDataKey?: string;
  childRunId?: string;
}

interface TaskLog {
  entries: LogEntry[];
}

export interface DurableOperatorOptions {
  handler: ServerlessHandler;
  secret: string;
  namespace: string;
  endpointId: string;
  /** Runs a non-durable child through the trigger route; resolves with its output. */
  runChild: (workflowName: string, input: unknown) => Promise<unknown>;
}

interface Waiter {
  kind: string;
  occurrence: number;
  resolve: (frame: ServerlessDurableFrame) => void;
}

const encoder = new TextEncoder();
const ORIGIN = 'https://endpoint.test';

function doneKind(frame: ServerlessDurableFrame): string {
  const { done } = frame;

  if (!done) return 'done';
  if (done.status === DONE_STATUS_EVICTED) return 'done:evicted';
  if (done.error !== undefined) return 'done:error';
  return 'done:output';
}

export class DurableOperator {
  readonly clock: VirtualClock;
  private now = 0;
  private logs = new Map<string, TaskLog>();
  private live = new Map<string, ActiveInvocation>();

  constructor(private readonly options: DurableOperatorOptions) {
    this.clock = {
      now: () => this.now,
      advance: (duration) => {
        this.now += typeof duration === 'number' ? duration : durationToMs(duration);
        this.settleSleeps();
      },
    };
  }

  /** Completes user-event waits for the key with the payload. */
  emit(eventKey: string, payload: Record<string, unknown> = {}): void {
    for (const [taskId, log] of this.logs) {
      for (const entry of log.entries) {
        if (entry.kind !== 'waitFor' || entry.completed || entry.eventKey !== eventKey) {
          continue;
        }

        this.completeEntry(taskId, entry, {
          CREATE: { [entry.readableDataKey ?? eventKey]: [payload] },
        });
      }
    }
  }

  private settleSleeps(): void {
    for (const [taskId, log] of this.logs) {
      for (const entry of log.entries) {
        if (
          entry.kind === 'waitFor' &&
          !entry.completed &&
          entry.sleepDueAt !== undefined &&
          entry.sleepDueAt <= this.now
        ) {
          this.completeEntry(taskId, entry, {
            CREATE: { [entry.readableDataKey ?? 'sleep']: [{}] },
          });
        }
      }
    }
  }

  private completeEntry(taskId: string, entry: LogEntry, payload: unknown, failure?: string): void {
    entry.completed = true;
    entry.isFailure = failure !== undefined;
    entry.errorMessage = failure;
    entry.payload = failure === undefined ? encoder.encode(JSON.stringify(payload)) : undefined;
    this.live.get(taskId)?.deliverCompletion(entry);
  }

  /** Signed upgrade headers for the task and invocation, with optional overrides. */
  async upgradeHeaders(
    taskRunExternalId: string,
    invocationCount: number,
    overrides: Partial<
      Record<'timestamp' | 'nonce' | 'endpointId' | 'signature' | 'secret', string>
    > = {}
  ): Promise<Headers> {
    const timestamp = overrides.timestamp ?? String(Math.floor(Date.now() / 1000));
    const nonce = overrides.nonce ?? crypto.randomUUID();
    const invocation = String(invocationCount);
    const endpointId = overrides.endpointId ?? this.options.endpointId;
    const signature =
      overrides.signature ??
      (await signHex(
        overrides.secret ?? this.options.secret,
        upgradeSigningPayload(endpointId, timestamp, nonce, taskRunExternalId, invocation)
      ));

    return new Headers({
      upgrade: 'websocket',
      connection: 'upgrade',
      [SIGNATURE_HEADER]: signature,
      [TIMESTAMP_HEADER]: timestamp,
      [NONCE_HEADER]: nonce,
      [TASK_ID_HEADER]: taskRunExternalId,
      [INVOCATION_HEADER]: invocation,
      [ENDPOINT_ID_HEADER]: endpointId,
    });
  }

  start(
    workflowName: string,
    taskName: string,
    input: unknown,
    options: DurableInvokeOptions = {}
  ): DurableRun {
    const taskRunExternalId = options.taskRunExternalId ?? crypto.randomUUID();
    const invocationCount = options.invocationCount ?? 1;
    const log = this.logs.get(taskRunExternalId) ?? { entries: [] };
    this.logs.set(taskRunExternalId, log);

    const active = new ActiveInvocation(this, this.options, log, {
      taskRunExternalId,
      invocationCount,
      inlineWaitBudgetMs: options.inlineWaitBudgetMs ?? 5000,
      workflowName,
      taskName,
      input,
      options,
    });

    this.live.set(taskRunExternalId, active);
    active.result.finally(() => {
      if (this.live.get(taskRunExternalId) === active) {
        this.live.delete(taskRunExternalId);
      }
    });

    void active.dial();

    return active;
  }
}

interface ActiveInvocationParams {
  taskRunExternalId: string;
  invocationCount: number;
  inlineWaitBudgetMs: number;
  workflowName: string;
  taskName: string;
  input: unknown;
  options: DurableInvokeOptions;
}

class ActiveInvocation implements DurableRun {
  readonly result: Promise<DurableResult>;
  readonly taskRunExternalId: string;
  readonly invocationCount: number;

  private resolveResult!: (result: DurableResult) => void;
  private frames: RecordedFrame[] = [];
  private waiters: Waiter[] = [];
  private socket: DurableSocket | undefined;
  private pair = createSocketPair();
  private cursor = 0;
  private inFlight = false;
  private evictionAcked = false;
  private errored = false;
  private done: ServerlessDurableFrame['done'] | undefined;
  private violation: string | undefined;
  private settled = false;
  private httpStatus = 0;

  constructor(
    private readonly operator: DurableOperator,
    private readonly options: DurableOperatorOptions,
    private readonly log: TaskLog,
    private readonly params: ActiveInvocationParams
  ) {
    this.taskRunExternalId = params.taskRunExternalId;
    this.invocationCount = params.invocationCount;
    this.result = new Promise((resolve) => {
      this.resolveResult = resolve;
    });
  }

  frame(kind: string, occurrence = 1): Promise<ServerlessDurableFrame> {
    const seen = this.frames.filter((f) => f.from === 'endpoint' && this.kindOf(f.frame) === kind);

    if (seen.length >= occurrence) {
      return Promise.resolve(seen[occurrence - 1].frame);
    }

    return new Promise((resolve) => {
      this.waiters.push({ kind, occurrence, resolve });
    });
  }

  private kindOf(frame: ServerlessDurableFrame): string {
    return frame.done ? doneKind(frame) : frameKind(frame);
  }

  async dial(): Promise<void> {
    const { handler } = this.options;
    const headers = await this.operator.upgradeHeaders(
      this.taskRunExternalId,
      this.invocationCount
    );
    const request = new Request(`${ORIGIN}${handler.basePath}/trigger`, { headers });

    let response: Response;

    try {
      response = await handler.fetch(request, undefined, {
        upgrade: (_request, run) => {
          this.socket = this.pair.operator;
          this.socket.onMessage((text) => this.onEndpointFrame(text));
          this.socket.onClose((code, reason) => this.onClose(code, reason));
          void run(this.pair.endpoint);
          return new Response(null, { status: 200, headers: { 'x-upgraded': '101' } });
        },
      });
    } catch (err) {
      this.settle('failed', { error: `upgrade failed: ${String(err)}`, retry: true }, 0, 1006, '');
      return;
    }

    if (response.headers.get('x-upgraded') !== '101') {
      const body = await response.text();
      this.settle(
        'failed',
        { error: `upgrade refused: ${response.status} ${body}`, retry: false },
        response.status,
        0,
        ''
      );
      return;
    }

    this.httpStatus = 101;
    const first = this.firstFrame();
    this.send({ first: this.params.options.firstFrame?.(first) ?? first });
  }

  private firstFrame(): ServerlessFirstFrame {
    const { namespace } = this.options;
    const { params } = this;
    const prefix = namespace ? `${namespace}_` : '';
    const action = AssignedAction.fromPartial({
      tenantId: crypto.randomUUID(),
      workflowRunId: params.options.workflowRunId ?? crypto.randomUUID(),
      jobId: crypto.randomUUID(),
      jobName: `${prefix}${params.workflowName}`,
      jobRunId: crypto.randomUUID(),
      taskId: crypto.randomUUID(),
      taskRunExternalId: this.taskRunExternalId,
      actionId: `${prefix}${params.workflowName}:${params.taskName}`.toLowerCase(),
      actionType: ActionType.START_STEP_RUN,
      actionPayload: JSON.stringify({
        input: params.input,
        parents: params.options.parents ?? {},
        triggered_by: 'manual',
      }),
      taskName: params.taskName,
      retryCount: params.options.retryCount ?? 0,
      additionalMetadata: params.options.additionalMetadata
        ? JSON.stringify(params.options.additionalMetadata)
        : undefined,
      priority: 1,
      durableTaskInvocationCount: this.invocationCount,
    });

    return {
      action,
      namespace,
      invocationCount: this.invocationCount,
      inlineWaitBudgetMs: this.params.inlineWaitBudgetMs,
    };
  }

  private send(frame: ServerlessDurableFrame): void {
    if (!this.socket || this.pair.closed) {
      return;
    }

    this.frames.push({ from: 'operator', frame });
    this.socket.send(encodeFrame(frame));
  }

  private respond(response: DurableTaskResponse): void {
    this.send({ response });
  }

  private onEndpointFrame(text: string): void {
    let frame: ServerlessDurableFrame;

    try {
      frame = decodeFrame(text);
    } catch (err) {
      this.violate(`malformed frame: ${String(err)}`);
      return;
    }

    this.frames.push({ from: 'endpoint', frame });

    // Requests are processed on a microtask (which is where their ack goes out and their
    // log entry appears); their waiters are notified from there so a test that acts on the
    // entry right after `frame()` finds it. Everything else is notified at once.
    if (!frame.request || frame.request.completeMemo || this.done) {
      this.notifyWaiters(frame);
    }

    if (this.done) {
      this.violate(`frame ${this.kindOf(frame)} after the done frame`);
      return;
    }

    if (frame.done) {
      this.onDone(frame);
      return;
    }

    if (!frame.request) {
      this.violate(`unexpected frame ${this.kindOf(frame)}`);
      return;
    }

    this.onRequest(frame.request);
  }

  private notifyWaiters(frame: ServerlessDurableFrame): void {
    const kind = this.kindOf(frame);
    const count = this.frames.filter(
      (f) => f.from === 'endpoint' && this.kindOf(f.frame) === kind
    ).length;

    this.waiters = this.waiters.filter((waiter) => {
      if (waiter.kind === kind && waiter.occurrence <= count) {
        waiter.resolve(frame);
        return false;
      }

      return true;
    });
  }

  private violate(reason: string): void {
    this.violation = reason;
    this.socket?.close(4005, reason);
  }

  private ref(entry: LogEntry): DurableEventLogEntryRef {
    return {
      durableTaskExternalId: this.taskRunExternalId,
      invocationCount: this.invocationCount,
      branchId: entry.branchId,
      nodeId: entry.nodeId,
    };
  }

  /** Matches the request against the log at the cursor, appending when the log ends. */
  private nextEntry(kind: LogEntry['kind']): LogEntry | undefined {
    const existing = this.log.entries[this.cursor];

    if (existing) {
      if (existing.kind !== kind) {
        this.errored = true;
        this.send({
          error: {
            code: 'nondeterminism',
            message: `replay diverged at node ${existing.nodeId}: recorded ${existing.kind}, got ${kind}`,
          },
        });
        return undefined;
      }

      this.cursor += 1;
      return existing;
    }

    const entry: LogEntry = {
      branchId: 0,
      nodeId: this.log.entries.length + 1,
      kind,
      completed: false,
      isFailure: false,
    };

    this.log.entries.push(entry);
    this.cursor += 1;

    return entry;
  }

  private onRequest(request: DurableTaskRequest): void {
    const ids = request.memo ?? request.waitFor ?? request.triggerRuns ?? request.evictInvocation;

    if (ids) {
      if (
        ids.durableTaskExternalId !== this.taskRunExternalId ||
        ids.invocationCount !== this.invocationCount
      ) {
        this.violation = 'invocation mismatch';
        this.socket?.close(4004, 'invocation mismatch');
        return;
      }

      if (this.inFlight) {
        this.violation = 'a second request while one awaits its ack';
        this.socket?.close(
          4006,
          'endpoint sent a durable request while another was awaiting its ack'
        );
        return;
      }
    }

    if (request.completeMemo) {
      const { ref, payload } = request.completeMemo;
      const entry = this.log.entries.find(
        (e) => e.nodeId === ref?.nodeId && e.branchId === ref?.branchId
      );

      if (entry) {
        entry.completed = true;
        entry.payload = payload;
        this.respond({ entryCompleted: { ref: this.ref(entry), payload, isFailure: false } });
      }

      return;
    }

    if (request.registerWorker || request.workerStatus) {
      this.violate('link-internal request');
      return;
    }

    this.inFlight = true;

    queueMicrotask(() => {
      this.inFlight = false;

      if (this.pair.closed) {
        return;
      }

      this.handleAckBearing(request);
      this.notifyWaiters({ request });
    });
  }

  private handleAckBearing(request: DurableTaskRequest): void {
    {
      if (request.memo) {
        const entry = this.nextEntry('memo');
        if (!entry) return;

        const memoKey = Array.from(request.memo.key).join(',');
        const replay = entry.completed && entry.memoKey === memoKey;
        entry.memoKey = memoKey;

        this.respond({
          memoAck: {
            ref: this.ref(entry),
            memoAlreadyExisted: replay,
            memoResultPayload: replay ? entry.payload : undefined,
          },
        });

        if (replay) {
          this.deliverCompletion(entry);
        }

        return;
      }

      if (request.waitFor) {
        const entry = this.nextEntry('waitFor');
        if (!entry) return;

        if (!entry.completed && entry.sleepDueAt === undefined && entry.eventKey === undefined) {
          const conditions = request.waitFor.waitForConditions;
          const [sleep] = conditions?.sleepConditions ?? [];
          const [event] = conditions?.userEventConditions ?? [];

          if (sleep) {
            entry.sleepDueAt = this.operator.clock.now() + durationToMs(sleep.sleepFor as Duration);
            entry.readableDataKey = sleep.base?.readableDataKey || undefined;
          } else if (event) {
            entry.eventKey = stripNamespace(event.userEventKey, this.options.namespace);
            entry.readableDataKey = event.base?.readableDataKey || entry.eventKey;
          }
        }

        this.respond({ waitForAck: { ref: this.ref(entry) } });

        if (entry.completed) {
          this.deliverCompletion(entry);
        } else if (
          entry.sleepDueAt !== undefined &&
          entry.sleepDueAt <= this.operator.clock.now()
        ) {
          entry.completed = true;
          entry.payload = encoder.encode(
            JSON.stringify({ CREATE: { [entry.readableDataKey ?? 'sleep']: [{}] } })
          );
          this.deliverCompletion(entry);
        }

        return;
      }

      if (request.triggerRuns) {
        const entries: LogEntry[] = [];

        for (const opt of request.triggerRuns.triggerOpts) {
          const entry = this.nextEntry('child');
          if (!entry) return;

          entries.push(entry);

          if (!entry.childRunId) {
            entry.childRunId = crypto.randomUUID();
            const name = stripNamespace(opt.name, this.options.namespace);
            let input: unknown = {};

            try {
              input = opt.input ? JSON.parse(opt.input) : {};
            } catch {
              // A malformed child input runs the child with an empty object.
            }

            this.options.runChild(name, input).then(
              (output) => this.completeChild(entry, output),
              (err) => this.completeChild(entry, undefined, String(err))
            );
          }
        }

        this.respond({
          triggerRunsAck: {
            durableTaskExternalId: this.taskRunExternalId,
            invocationCount: this.invocationCount,
            runEntries: entries.map((entry) => ({
              nodeId: entry.nodeId,
              branchId: entry.branchId,
              workflowRunExternalId: entry.childRunId ?? '',
            })),
          },
        });

        for (const entry of entries) {
          if (entry.completed) {
            this.deliverCompletion(entry);
          }
        }

        return;
      }

      if (request.evictInvocation) {
        this.evictionAcked = true;
        this.respond({
          evictionAck: {
            durableTaskExternalId: this.taskRunExternalId,
            invocationCount: this.invocationCount,
          },
        });
      }
    }
  }

  private completeChild(entry: LogEntry, output: unknown, failure?: string): void {
    entry.completed = true;
    entry.isFailure = failure !== undefined;
    entry.errorMessage = failure;
    entry.payload =
      failure === undefined ? encoder.encode(JSON.stringify(output ?? {})) : undefined;
    this.deliverCompletion(entry);
  }

  /** Sends entryCompleted for a completed entry while this invocation is connected. */
  deliverCompletion(entry: LogEntry): void {
    if (!entry.completed || this.pair.closed || this.done) {
      return;
    }

    this.respond({
      entryCompleted: {
        ref: this.ref(entry),
        payload: entry.payload ?? new Uint8Array(),
        isFailure: entry.isFailure,
        errorMessage: entry.errorMessage,
      },
    });
  }

  private onDone(frame: ServerlessDurableFrame): void {
    const done = frame.done!;
    this.done = done;

    if (done.status === DONE_STATUS_EVICTED && !this.evictionAcked) {
      this.violation = 'done evicted without an eviction ack';
      this.socket?.close(4005, this.violation);
      return;
    }

    if (done.output !== undefined) {
      try {
        JSON.parse(done.output);
      } catch {
        this.violation = 'done output is not JSON';
        this.socket?.close(4005, this.violation);
        return;
      }
    }

    this.socket?.close(1000, 'done');
  }

  private onClose(code: number, reason: string): void {
    if (this.settled) {
      return;
    }

    const { done } = this;

    if (this.violation && code !== 1000) {
      this.settle(
        'failed',
        { error: `protocol violation: ${this.violation}`, retry: true },
        this.httpStatus,
        code,
        reason
      );
      return;
    }

    if (done?.status === DONE_STATUS_EVICTED) {
      this.settle('evicted', {}, this.httpStatus, code, reason);
      return;
    }

    if (done?.error !== undefined) {
      this.settle(
        'failed',
        { error: done.error, retry: this.errored ? false : done.retry },
        this.httpStatus,
        code,
        reason
      );
      return;
    }

    if (this.errored) {
      this.settle(
        'failed',
        { error: 'engine error without a done frame', retry: false },
        this.httpStatus,
        code,
        reason
      );
      return;
    }

    if (done) {
      this.settle(
        'completed',
        { output: done.output !== undefined ? JSON.parse(done.output) : {} },
        this.httpStatus,
        code,
        reason
      );
      return;
    }

    if (this.evictionAcked) {
      this.settle('evicted', {}, this.httpStatus, code, reason);
      return;
    }

    this.settle(
      'failed',
      { error: `endpoint closed the websocket (code ${code}) without a done frame`, retry: true },
      this.httpStatus,
      code,
      reason
    );
  }

  private settle(
    status: DurableResult['status'],
    outcome: { output?: unknown; error?: string; retry?: boolean },
    httpStatus: number,
    closeCode: number,
    closeReason: string
  ): void {
    if (this.settled) {
      return;
    }

    this.settled = true;
    this.resolveResult({
      status,
      ...outcome,
      httpStatus,
      closeCode,
      closeReason,
      taskRunExternalId: this.taskRunExternalId,
      invocationCount: this.invocationCount,
      frames: this.frames,
      endpointFrames: this.frames
        .filter((f) => f.from === 'endpoint')
        .map((f) => this.kindOf(f.frame)),
      operatorFrames: this.frames
        .filter((f) => f.from === 'operator')
        .map((f) => this.kindOf(f.frame)),
    });
  }

  serverEvict(reason = 'superseded by the engine'): void {
    this.respond({
      serverEvict: {
        durableTaskExternalId: this.taskRunExternalId,
        invocationCount: this.invocationCount,
        reason,
      },
    });
    this.evictionAcked = true;
    queueMicrotask(() => this.socket?.close(4001, reason));
  }

  sendError(code: 'nondeterminism' | 'unspecified', message: string): void {
    this.errored = true;
    this.send({ error: { code, message } });
  }

  close(code: number, reason = ''): void {
    this.socket?.close(code, reason);
  }
}
