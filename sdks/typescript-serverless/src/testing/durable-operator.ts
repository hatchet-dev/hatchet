/**
 * An in-memory serverless operator for socket invocations. It dials the handler through
 * the `DurableHooks.upgrade` seam with a signed upgrade, speaks the frame protocol of
 * pkg/serverlessoperator/durable over an in-memory socket pair, keeps a durable event log
 * per task run (memo entries replay, wait entries complete from a virtual clock or from
 * emitted events, child runs execute the registered task function), answers the engine
 * streams a task opens over the socket from a scripted engine, and enforces the operator's
 * protocol rules so a misbehaving endpoint fails the test.
 */
import { durationToMs, type Duration } from '@hatchet-dev/typescript-sdk/edge/index.js';
import {
  ActionType,
  AssignedAction,
  SubscribeToWorkflowEventsRequest,
  SubscribeToWorkflowRunsRequest,
  WorkflowEvent,
  WorkflowRunEvent,
  WorkflowRunEventType,
} from '../generated/proto/dispatcher';
import { StreamMessage, SubscribeStreamRequest } from '../generated/proto/v1/streams';
import type {
  DurableEventLogEntryRef,
  DurableTaskRequest,
  DurableTaskResponse,
} from '../generated/proto/v1/dispatcher';
import type {
  ServerlessDurableFrame,
  ServerlessFirstFrame,
  ServerlessStreamClose,
  ServerlessStreamMessage,
  ServerlessStreamOpen,
} from '../generated/proto/v1/serverless';
import {
  MAX_SOCKET_STREAMS,
  SOCKET_STREAM_PROCEDURES,
  SUBSCRIBE_STREAM,
  SUBSCRIBE_TO_WORKFLOW_EVENTS,
  SUBSCRIBE_TO_WORKFLOW_RUNS,
  StreamCloseCode,
  streamCloseCodeName,
} from '../handler/durable/streams';
import {
  DONE_STATUS_EVICTED,
  ENDPOINT_ID_HEADER,
  INVOCATION_HEADER,
  NONCE_HEADER,
  SIGNATURE_HEADER,
  TASK_ID_HEADER,
  TIMESTAMP_HEADER,
  upgradeSigningPayload,
} from '../handler/contract';
import { OPERATOR_MAX_FRAME_BYTES } from '../handler/contract';
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

/** A socket invocation in progress, for tests that act on the operator side mid-flight. */
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
  /** The ids of the streams the endpoint holds open on the socket right now. */
  openStreams(): string[];
  /** The operator ends one stream with a connect code, as it does when the engine drops it. */
  closeStream(id: string, code: number, message?: string): void;
}

/** One task's result in a run's terminal event, as `completeRun` takes it. */
export interface RunTaskResult {
  taskName: string;
  /** The task's output, serialized as JSON in the event. */
  output?: unknown;
  error?: string;
}

/** One retained message of a scripted topic. */
export interface TopicStreamEntry {
  payload?: string | Uint8Array;
  /** Defaults to `c<n>`, n being the entry's 1-based position in the script. */
  cursor?: string;
  createdAt?: Date;
  /** An uploaded payload's ref, in place of `payload`. */
  payloadRef?: string;
}

/**
 * What the engine sends on a durable stream topic subscription: the retained entries after
 * the request's cursor, then a hang-up and a close as scripted. A request naming a cursor
 * that is not an entry's is closed with code 11 (out of range).
 */
export interface TopicStreamScript {
  /** The retained messages, oldest first; a string or bytes is an entry with that payload. */
  entries?: Array<string | Uint8Array | TopicStreamEntry>;
  /** How many entries one stream_message carries. Defaults to 1. */
  batch?: number;
  /** Sends a hang-up message after the entries, as the engine does with an idle subscription. */
  hangup?: boolean;
  /**
   * How the stream ends after the entries: a connect code (0 is the engine finishing it) or
   * `false`, the default, which keeps it open until the task closes it, as the engine does
   * while a topic is live.
   */
  close?: number | false;
  /** The stream_close message; defaults to the operator's fixed message for the code. */
  message?: string;
}

/**
 * The engine behind the streams a task opens over its socket: run subscriptions, run event
 * subscriptions and durable stream topics. `/v1.V1Dispatcher/ListenForDurableEvent` is
 * closed with code 12 (unimplemented), since the test operator has no engine for it.
 */
export interface RunStreamScript {
  /**
   * Answers a subscription to a run with its terminal event. By default the subscription
   * waits for `completeRun(workflowRunId, ...)`, or is answered at once when the run was
   * completed already. A rejection closes the stream with the error's `code` (a connect
   * code) or 2 (unknown) and its message.
   */
  workflowRun?: (workflowRunId: string) => WorkflowRunEvent | Promise<WorkflowRunEvent>;
  /**
   * Answers a subscription to a run's events (`client.runs.events(runId)`) with the events,
   * sent in order, then the engine finishing the stream (code 0). Without one the stream is
   * closed with code 12. A rejection closes it the way `workflowRun` does.
   */
  workflowEvents?: (
    request: SubscribeToWorkflowEventsRequest
  ) => WorkflowEvent[] | Promise<WorkflowEvent[]>;
  /**
   * Answers a subscription to a durable stream topic (`client.streams.events(topic)`) with
   * the script to play on it. Without one the subscription stays open with no messages until
   * the task closes it, as the engine does for a topic nothing was published to. A rejection
   * closes the stream the way `workflowRun` does: `{ code: 7 }` is the engine refusing a
   * tenant without the durable streams entitlement.
   */
  topic?: (request: SubscribeStreamRequest) => TopicStreamScript | Promise<TopicStreamScript>;
  /** The cap on open streams per socket; defaults to the operator's 16. */
  maxStreams?: number;
}

/**
 * The message a stream_close carries for a connect code, as the operator words it
 * (pkg/serverlessoperator/durable/streams.go publicStreamMessages): the code's name and a
 * phrase for the kind of failure, never the engine's own text.
 */
export function publicStreamMessage(code: number): string {
  const phrases: Record<number, string> = {
    [StreamCloseCode.CANCELED]: 'the engine stream was canceled',
    [StreamCloseCode.UNKNOWN]: 'the engine stream failed',
    [StreamCloseCode.INVALID_ARGUMENT]: 'the engine refused the request',
    [StreamCloseCode.DEADLINE_EXCEEDED]: 'the engine stream timed out',
    [StreamCloseCode.NOT_FOUND]: 'the engine found no such resource',
    [StreamCloseCode.ALREADY_EXISTS]: 'the resource already exists',
    [StreamCloseCode.PERMISSION_DENIED]: 'the engine denied the request',
    [StreamCloseCode.RESOURCE_EXHAUSTED]:
      'the engine or the operator is out of capacity for the request',
    [StreamCloseCode.FAILED_PRECONDITION]: 'the engine refused the request in its current state',
    [StreamCloseCode.ABORTED]: 'the engine aborted the stream',
    [StreamCloseCode.OUT_OF_RANGE]: 'the request was out of range',
    [StreamCloseCode.UNIMPLEMENTED]: 'the engine does not serve the request',
    [StreamCloseCode.INTERNAL]: 'the engine failed',
    [StreamCloseCode.UNAVAILABLE]: 'the engine stream was closed',
    [StreamCloseCode.DATA_LOSS]: 'the engine lost data for the stream',
    [StreamCloseCode.UNAUTHENTICATED]: "the engine did not accept the operator's credentials",
  };
  const known = code in phrases ? code : StreamCloseCode.UNKNOWN;

  return `${streamCloseCodeName(known)}: ${phrases[known]}`;
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
  endpointId: string;
  /** Runs a non-durable child through the trigger route; resolves with its output. */
  runChild: (workflowName: string, input: unknown) => Promise<unknown>;
  /** Whether the action is a durable task; a durable request on any other socket is a violation. */
  isDurable: (actionId: string) => boolean;
  /** The engine behind the streams a task opens; defaults to `completeRun`-driven runs. */
  runStream?: RunStreamScript;
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
  /** Terminal events of completed runs, delivered to later subscriptions at once. */
  private runEvents = new Map<string, WorkflowRunEvent>();
  private runWaiters = new Map<string, Array<(event: WorkflowRunEvent) => void>>();

  constructor(private readonly options: DurableOperatorOptions) {
    this.clock = {
      now: () => this.now,
      advance: (duration) => {
        this.now += typeof duration === 'number' ? duration : durationToMs(duration);
        this.settleSleeps();
      },
    };
  }

  /**
   * Finishes a run the engine is asked about over a socket: every subscription to it, now
   * or later, gets the terminal event carrying the tasks' outputs and errors.
   */
  completeRun(workflowRunId: string, results: RunTaskResult[]): void {
    const event = WorkflowRunEvent.create({
      workflowRunId,
      eventType: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FINISHED,
      eventTimestamp: new Date(),
      results: results.map((result) => ({
        taskRunExternalId: crypto.randomUUID(),
        taskName: result.taskName,
        jobRunId: crypto.randomUUID(),
        error: result.error,
        output: result.output === undefined ? undefined : JSON.stringify(result.output),
      })),
    });

    this.runEvents.set(workflowRunId, event);

    for (const waiter of this.runWaiters.get(workflowRunId) ?? []) {
      waiter(event);
    }

    this.runWaiters.delete(workflowRunId);
  }

  /** The engine's answer to a subscription: the scripted one, or the `completeRun` event. */
  awaitRun(workflowRunId: string): Promise<WorkflowRunEvent> {
    const scripted = this.options.runStream?.workflowRun;

    if (scripted) {
      return Promise.resolve().then(() => scripted(workflowRunId));
    }

    const event = this.runEvents.get(workflowRunId);

    if (event) {
      return Promise.resolve(event);
    }

    return new Promise((resolve) => {
      const waiters = this.runWaiters.get(workflowRunId) ?? [];
      waiters.push(resolve);
      this.runWaiters.set(workflowRunId, waiters);
    });
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
  /** The engine streams open on the socket, by the endpoint's id, with their procedure. */
  private streams = new Map<string, string>();

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
    const { params } = this;
    const action = AssignedAction.fromPartial({
      tenantId: crypto.randomUUID(),
      workflowRunId: params.options.workflowRunId ?? crypto.randomUUID(),
      jobId: crypto.randomUUID(),
      jobName: params.workflowName,
      jobRunId: crypto.randomUUID(),
      taskId: crypto.randomUUID(),
      taskRunExternalId: this.taskRunExternalId,
      actionId: `${params.workflowName}:${params.taskName}`.toLowerCase(),
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
      // the engine sets the durable invocation count on durable tasks only; a non-durable task
      // invoked over the socket for its streams carries none, like the real operator sends it
      durableTaskInvocationCount: this.options.isDurable(this.actionId)
        ? this.invocationCount
        : undefined,
    });

    return {
      action,
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

    // The relay reads at most this many bytes per frame and treats a larger one as a protocol
    // violation (close 1009), so a done frame carrying an output over the limit never
    // completes the task; the same here, so a test sees the boundary the operator enforces.
    if (new TextEncoder().encode(text).byteLength > OPERATOR_MAX_FRAME_BYTES) {
      // Settled here rather than on the close that follows: the endpoint may close with 1000
      // right behind its done frame, and that close would otherwise read as a clean end.
      const reason = `endpoint frame exceeded the ${OPERATOR_MAX_FRAME_BYTES} byte limit`;
      this.violation = reason;
      this.settle(
        'failed',
        { error: `protocol violation: ${reason}`, retry: true },
        this.httpStatus,
        1009,
        reason
      );
      this.socket?.close(1009, reason);
      return;
    }

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

    if (frame.streamOpen) {
      this.onStreamOpen(frame.streamOpen);
      return;
    }

    if (frame.streamMessage) {
      this.onStreamMessage(frame.streamMessage);
      return;
    }

    if (frame.streamClose) {
      this.onStreamClose(frame.streamClose);
      return;
    }

    if (!frame.request) {
      this.violate(`unexpected frame ${this.kindOf(frame)}`);
      return;
    }

    if (!this.options.isDurable(this.actionId)) {
      this.violate('endpoint sent a durable request on the socket of a non-durable task');
      return;
    }

    this.onRequest(frame.request);
  }

  private get actionId(): string {
    return `${this.params.workflowName}:${this.params.taskName}`.toLowerCase();
  }

  openStreams(): string[] {
    return [...this.streams.keys()];
  }

  closeStream(id: string, code: number, message = ''): void {
    if (this.streams.delete(id)) {
      this.sendStreamClose(id, code, message);
    }
  }

  private sendStreamClose(id: string, code: number, message: string): void {
    this.send({ streamClose: { id, code, message } });
  }

  /**
   * Mirrors pkg/serverlessoperator/durable/streams.go handleStreamOpen: a frame the contract
   * forbids ends the socket; anything the stream layer can answer is a stream_close with a
   * connect code and leaves the socket up.
   */
  private onStreamOpen(open: ServerlessStreamOpen): void {
    if (!open.id) {
      this.violate('endpoint sent stream_open without an id');
      return;
    }

    if (this.streams.has(open.id)) {
      this.violate(`endpoint sent stream_open for stream "${open.id}", which is already open`);
      return;
    }

    if (!SOCKET_STREAM_PROCEDURES.includes(open.procedure)) {
      this.sendStreamClose(
        open.id,
        StreamCloseCode.UNIMPLEMENTED,
        `procedure "${open.procedure}" is not served over the socket`
      );
      return;
    }

    const maxStreams = this.options.runStream?.maxStreams ?? MAX_SOCKET_STREAMS;

    if (this.streams.size >= maxStreams) {
      this.sendStreamClose(
        open.id,
        StreamCloseCode.RESOURCE_EXHAUSTED,
        `the socket already holds ${maxStreams} streams, the limit`
      );
      return;
    }

    switch (open.procedure) {
      case SUBSCRIBE_TO_WORKFLOW_RUNS:
        this.streams.set(open.id, open.procedure);
        this.subscribeRun(open.id, open.request);
        return;
      case SUBSCRIBE_TO_WORKFLOW_EVENTS:
        this.streams.set(open.id, open.procedure);
        this.subscribeRunEvents(open.id, open.request);
        return;
      case SUBSCRIBE_STREAM:
        this.streams.set(open.id, open.procedure);
        this.subscribeTopic(open.id, open.request);
        return;
      default:
        this.sendStreamClose(
          open.id,
          StreamCloseCode.UNIMPLEMENTED,
          `the test operator does not serve ${open.procedure}`
        );
    }
  }

  private onStreamMessage(message: ServerlessStreamMessage): void {
    const procedure = this.streams.get(message.id);

    if (procedure === undefined) {
      // A message crossing the operator's close; the relay drops it too.
      return;
    }

    if (procedure !== SUBSCRIBE_TO_WORKFLOW_RUNS) {
      this.streams.delete(message.id);
      this.sendStreamClose(
        message.id,
        StreamCloseCode.INVALID_ARGUMENT,
        `${procedure} takes no message after the request`
      );
      return;
    }

    this.subscribeRun(message.id, message.message);
  }

  private onStreamClose(close: ServerlessStreamClose): void {
    this.streams.delete(close.id);
  }

  /** Decodes a subscription and answers it with the run's terminal event when it comes. */
  private subscribeRun(streamId: string, raw: string): void {
    let workflowRunId: string;

    try {
      ({ workflowRunId } = SubscribeToWorkflowRunsRequest.fromJSON(JSON.parse(raw)));
    } catch (err) {
      this.streams.delete(streamId);
      this.sendStreamClose(
        streamId,
        StreamCloseCode.INVALID_ARGUMENT,
        `request is not a SubscribeToWorkflowRuns request: ${String(err)}`
      );
      return;
    }

    this.operator.awaitRun(workflowRunId).then(
      (event) => this.sendStreamMessage(streamId, JSON.stringify(WorkflowRunEvent.toJSON(event))),
      (err: unknown) => this.failStream(streamId, err)
    );
  }

  /** Decodes a run events subscription and plays the scripted events, then the engine's end. */
  private subscribeRunEvents(streamId: string, raw: string): void {
    const script = this.options.runStream?.workflowEvents;

    if (!script) {
      this.endStream(streamId, StreamCloseCode.UNIMPLEMENTED);
      return;
    }

    let request: SubscribeToWorkflowEventsRequest;

    try {
      request = SubscribeToWorkflowEventsRequest.fromJSON(JSON.parse(raw));
    } catch (err) {
      this.endStream(
        streamId,
        StreamCloseCode.INVALID_ARGUMENT,
        `request is not a SubscribeToWorkflowEvents request: ${String(err)}`
      );
      return;
    }

    Promise.resolve()
      .then(() => script(request))
      .then(
        (events) => {
          for (const event of events) {
            if (!this.sendStreamMessage(streamId, JSON.stringify(WorkflowEvent.toJSON(event)))) {
              return;
            }
          }

          this.endStream(streamId, StreamCloseCode.OK);
        },
        (err: unknown) => this.failStream(streamId, err)
      );
  }

  /** Decodes a topic subscription and plays its script: the entries after the cursor, then the end. */
  private subscribeTopic(streamId: string, raw: string): void {
    let request: SubscribeStreamRequest;

    try {
      request = SubscribeStreamRequest.fromJSON(JSON.parse(raw));
    } catch (err) {
      this.endStream(
        streamId,
        StreamCloseCode.INVALID_ARGUMENT,
        `request is not a SubscribeStream request: ${String(err)}`
      );
      return;
    }

    const script = this.options.runStream?.topic ?? (() => ({}));

    Promise.resolve()
      .then(() => script(request))
      .then(
        (topic) => this.playTopic(streamId, request, topic),
        (err: unknown) => this.failStream(streamId, err)
      );
  }

  private playTopic(streamId: string, request: SubscribeStreamRequest, script: TopicStreamScript) {
    const entries = (script.entries ?? []).map((entry, index) => {
      const given: TopicStreamEntry =
        typeof entry === 'string' || entry instanceof Uint8Array ? { payload: entry } : entry;
      const payload =
        typeof given.payload === 'string'
          ? encoder.encode(given.payload)
          : (given.payload ?? new Uint8Array(0));

      return {
        payload,
        cursor: given.cursor ?? `c${index + 1}`,
        createdAt: given.createdAt,
        payloadRef: given.payloadRef ?? '',
      };
    });

    let start = 0;

    if (request.cursor !== undefined && request.cursor !== '') {
      const at = entries.findIndex((entry) => entry.cursor === request.cursor);

      if (at < 0) {
        this.endStream(streamId, StreamCloseCode.OUT_OF_RANGE);
        return;
      }

      start = at + 1;
    }

    const batch = Math.max(script.batch ?? 1, 1);

    for (let from = start; from < entries.length; from += batch) {
      const slice = entries.slice(from, from + batch);
      const message = StreamMessage.create({
        entries: slice,
        hangup: false,
        cursor: slice[slice.length - 1].cursor,
      });

      if (!this.sendStreamMessage(streamId, JSON.stringify(StreamMessage.toJSON(message)))) {
        return;
      }
    }

    if (script.hangup) {
      const message = StreamMessage.create({ entries: [], hangup: true, cursor: '' });

      if (!this.sendStreamMessage(streamId, JSON.stringify(StreamMessage.toJSON(message)))) {
        return;
      }
    }

    if (script.close !== undefined && script.close !== false) {
      this.endStream(streamId, script.close, script.message);
    }
  }

  /** Relays one engine message on an open stream; false once the stream or socket is gone. */
  private sendStreamMessage(streamId: string, message: string): boolean {
    if (this.streams.get(streamId) === undefined || this.done || this.pair.closed) {
      return false;
    }

    this.send({ streamMessage: { id: streamId, message } });

    return true;
  }

  /** The engine ends a stream: code 0 when it finished, else the code with the operator's message. */
  private endStream(streamId: string, code: number, message?: string): void {
    if (!this.streams.delete(streamId) || this.done || this.pair.closed) {
      return;
    }

    this.sendStreamClose(
      streamId,
      code,
      message ?? (code === StreamCloseCode.OK ? '' : publicStreamMessage(code))
    );
  }

  /** A script's rejection ends the stream with the error's `code` (a connect code) or 2. */
  private failStream(streamId: string, err: unknown): void {
    const code =
      typeof (err as { code?: unknown })?.code === 'number'
        ? (err as { code: number }).code
        : StreamCloseCode.UNKNOWN;
    const message = err instanceof Error ? err.message : undefined;

    this.endStream(streamId, code, message || undefined);
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
            entry.eventKey = event.userEventKey;
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
            const { name } = opt;
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
