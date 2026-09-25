/**
 * A small, dependency-free shim for the Hatchet serverless endpoint contract on Cloudflare
 * Workers. Every wire detail mirrors the operator's Go side, which is authoritative:
 *
 *   api-contracts/v1/serverless.proto          every request, response and websocket frame
 *   pkg/serverlessoperator/contract/http.go    header names, upgrade signature payload
 *   internal/signature/sign.go                 hex(hmac_sha256(secret, data))
 *   pkg/serverlessoperator/durable/protocol.go websocket close codes
 *   pkg/serverlessoperator/durable/dial.go     upgrade headers
 *   api-contracts/v1/workflows.proto           CreateWorkflowVersionRequest
 *   api-contracts/v1/dispatcher.proto          DurableTaskRequest / DurableTaskResponse
 *
 * Every message travels as protojson: field names are lowerCamelCase, `bytes` fields are
 * standard base64 strings, int64 fields arrive as strings, enums are their names, and unknown
 * fields are ignored on both sides. The interfaces below are hand-written mirrors of the
 * protobuf messages; the @hatchet-dev/serverless package generates them instead.
 */

// ---------------------------------------------------------------------------------------
// Headers and constants (contract/http.go)
// ---------------------------------------------------------------------------------------

export const SIGNATURE_HEADER = "X-Hatchet-Signature";
export const ENDPOINT_ID_HEADER = "X-Hatchet-Endpoint-Id";
export const TIMESTAMP_HEADER = "X-Hatchet-Timestamp";

/** Only on the durable websocket upgrade, which has no body to sign. */
export const NONCE_HEADER = "X-Hatchet-Nonce";
export const TASK_ID_HEADER = "X-Hatchet-Task-Id";
export const INVOCATION_HEADER = "X-Hatchet-Invocation";

/**
 * contract.RequestMaxAge (5 minutes), in seconds: how far a signed timestamp may lie from
 * the endpoint's clock, in either direction, before the request is refused. It applies to
 * the `timestamp` field of every signed POST body and to X-Hatchet-Timestamp on the upgrade.
 */
export const REQUEST_MAX_AGE_SECONDS = 5 * 60;

/** contract.UpgradeMaxAge, the same window as REQUEST_MAX_AGE_SECONDS. */
export const UPGRADE_MAX_AGE_SECONDS = REQUEST_MAX_AGE_SECONDS;

/** contract.TriggerEnvelopeVersion. */
export const TRIGGER_ENVELOPE_VERSION = 1;

const encoder = new TextEncoder();
const decoder = new TextDecoder();

// ---------------------------------------------------------------------------------------
// Signatures
// ---------------------------------------------------------------------------------------

/** hex(hmac_sha256(secret, data)), the same digest internal/signature.Sign produces. */
export async function signHex(secret: string, data: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    encoder.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const mac = await crypto.subtle.sign("HMAC", key, encoder.encode(data));

  return toHex(new Uint8Array(mac));
}

function toHex(bytes: Uint8Array): string {
  let out = "";

  for (const b of bytes) {
    out += b.toString(16).padStart(2, "0");
  }

  return out;
}

/**
 * Constant-time string comparison. Workers expose the non-standard
 * crypto.subtle.timingSafeEqual; the fallback never short-circuits on the first mismatch.
 */
export function constantTimeEqual(a: string, b: string): boolean {
  const ab = encoder.encode(a);
  const bb = encoder.encode(b);

  if (ab.byteLength !== bb.byteLength) {
    return false;
  }

  const subtle = crypto.subtle as SubtleCrypto & {
    timingSafeEqual?: (x: ArrayBufferView, y: ArrayBufferView) => boolean;
  };

  if (typeof subtle.timingSafeEqual === "function") {
    return subtle.timingSafeEqual(ab, bb);
  }

  let diff = 0;

  for (let i = 0; i < ab.length; i++) {
    diff |= ab[i] ^ bb[i];
  }

  return diff === 0;
}

/**
 * Verifies X-Hatchet-Signature on a POST (healthcheck or non-durable trigger). The signature
 * covers the raw body bytes exactly as sent, so read the body as text before parsing it.
 * This is the HMAC check only; verifySignedBody adds the freshness and endpoint checks
 * every handler needs.
 */
export async function verifyBodySignature(
  body: string,
  header: string | null,
  secret: string,
): Promise<boolean> {
  if (!header || !secret) {
    return false;
  }

  const expected = await signHex(secret, body);

  return constantTimeEqual(expected, header);
}

/** Reports whether a signed timestamp (Unix seconds) is within the freshness window of now. */
export function isFresh(timestampSeconds: number, nowSeconds: number): boolean {
  return (
    Number.isFinite(timestampSeconds) &&
    Math.abs(nowSeconds - timestampSeconds) <= REQUEST_MAX_AGE_SECONDS
  );
}

export type BodyVerification =
  | { ok: true; endpointId: string; timestamp: number }
  | { ok: false; status: 401 | 403; reason: string };

/**
 * Verifies a signed POST body end to end: the HMAC over the raw bytes, then the freshness
 * of the `timestamp` field the signature covers (a captured request is only good for
 * REQUEST_MAX_AGE_SECONDS on either side of the endpoint's clock), then the endpoint id
 * when the caller knows its own. A body that verifies but does not parse is refused.
 *
 * Within the window a request can still be repeated. The operator delivers every task at
 * most once per attempt, so treat (endpointId, taskRunExternalId, retryCount) as the
 * idempotency key of anything with side effects.
 */
export async function verifySignedBody(
  body: string,
  header: string | null,
  secret: string,
  opts: { endpointId?: string; nowSeconds?: number } = {},
): Promise<BodyVerification> {
  if (!(await verifyBodySignature(body, header, secret))) {
    return { ok: false, status: 401, reason: "bad signature" };
  }

  let parsed: { endpointId?: unknown; timestamp?: unknown };

  try {
    parsed = JSON.parse(body) as { endpointId?: unknown; timestamp?: unknown };
  } catch {
    return { ok: false, status: 401, reason: "malformed body" };
  }

  // An int64 on the wire, so a decimal string.
  const timestamp = Number.parseInt(String(parsed.timestamp ?? ""), 10);
  const now = opts.nowSeconds ?? Math.floor(Date.now() / 1000);

  if (!isFresh(timestamp, now)) {
    return { ok: false, status: 401, reason: "stale or missing timestamp" };
  }

  const endpointId = typeof parsed.endpointId === "string" ? parsed.endpointId : "";

  if (opts.endpointId && endpointId !== opts.endpointId) {
    return { ok: false, status: 403, reason: "unknown endpoint id" };
  }

  return { ok: true, endpointId, timestamp };
}

/**
 * The set of upgrade nonces accepted within the freshness window. Every accepted nonce is
 * kept until its window has passed (a nonce older than the window is refused by the timestamp
 * check anyway), so a captured upgrade cannot be replayed within it whatever the request rate;
 * once `capacity` live nonces are held, further upgrades are refused rather than a live nonce
 * forgotten. Memory is bounded by the capacity either way.
 *
 * It lives in one isolate. Workers run many isolates, so a replay that lands in another
 * isolate is not caught by this set; production endpoints should back it with a Durable
 * Object (one per endpoint id) or KV with an expiring key, consumed atomically after the
 * signature verifies.
 */
export class NonceSet {
  private readonly seen = new Map<string, number>();

  constructor(private readonly capacity = 4096) {}

  /**
   * Records nonce with the timestamp it was signed for. Returns "replayed" when it was
   * already present, "full" when the set holds `capacity` nonces still within their window
   * and cannot admit another, and "accepted" otherwise. Call it only after the signature
   * verified, so unsigned traffic cannot fill it.
   */
  consume(nonce: string, timestampSeconds: number, nowSeconds: number): "accepted" | "replayed" | "full" {
    this.expire(nowSeconds);

    if (this.seen.has(nonce)) {
      return "replayed";
    }

    if (this.seen.size >= this.capacity) {
      return "full";
    }

    this.seen.set(nonce, timestampSeconds);

    return "accepted";
  }

  get size(): number {
    return this.seen.size;
  }

  private expire(nowSeconds: number): void {
    for (const [nonce, ts] of this.seen) {
      if (nowSeconds - ts > REQUEST_MAX_AGE_SECONDS) {
        this.seen.delete(nonce);
      }
    }
  }
}

/** The isolate's nonce set for durable upgrades. */
export const upgradeNonces = new NonceSet();

export type UpgradeVerification =
  | { ok: true; endpointId: string; taskId: string; invocation: number; nonce: string }
  | { ok: false; status: 401 | 403 | 503; reason: string };

/**
 * Verifies the bodyless websocket upgrade (durable/dial.go signedUpgradeHeaders). The
 * signature covers contract.UpgradeSigningPayload:
 *
 *   endpoint_id + "." + timestamp + "." + nonce + "." + task_id + "." + invocation
 *
 * each as it appears in its header, so a signature made for one endpoint cannot be presented
 * to another that shares the secret. The timestamp must be within UPGRADE_MAX_AGE_SECONDS of
 * now in either direction, and the nonce is consumed from `nonces` (default: the isolate's
 * upgradeNonces) after the signature verified, so a captured upgrade cannot be replayed
 * within the window; a set that cannot admit another live nonce refuses the upgrade with 503
 * rather than forget one. The verified task id and invocation must then match the first
 * frame the operator sends: DurableClient.assertMatches does that.
 */
export async function verifyUpgradeSignature(
  headers: Headers,
  secret: string,
  opts: { endpointId?: string; nowSeconds?: number; nonces?: NonceSet | null } = {},
): Promise<UpgradeVerification> {
  const endpointId = headers.get(ENDPOINT_ID_HEADER) ?? "";

  if (opts.endpointId && endpointId !== opts.endpointId) {
    return { ok: false, status: 403, reason: "unknown endpoint id" };
  }

  const timestamp = headers.get(TIMESTAMP_HEADER) ?? "";
  const ts = Number.parseInt(timestamp, 10);
  const now = opts.nowSeconds ?? Math.floor(Date.now() / 1000);

  if (!isFresh(ts, now)) {
    return { ok: false, status: 401, reason: "stale or missing timestamp" };
  }

  const nonce = headers.get(NONCE_HEADER) ?? "";

  if (nonce === "") {
    return { ok: false, status: 401, reason: "missing nonce" };
  }

  const taskId = headers.get(TASK_ID_HEADER) ?? "";
  const invocation = headers.get(INVOCATION_HEADER) ?? "";
  const payload = `${endpointId}.${timestamp}.${nonce}.${taskId}.${invocation}`;
  const expected = await signHex(secret, payload);

  if (!constantTimeEqual(expected, headers.get(SIGNATURE_HEADER) ?? "")) {
    return { ok: false, status: 401, reason: "bad signature" };
  }

  // Only a verified nonce enters the set, so unsigned traffic cannot fill it.
  const nonces = opts.nonces === undefined ? upgradeNonces : opts.nonces;

  if (nonces) {
    switch (nonces.consume(nonce, ts, now)) {
      case "replayed":
        return { ok: false, status: 401, reason: "replayed nonce" };
      case "full":
        return { ok: false, status: 503, reason: "nonce storage full; retry later" };
      case "accepted":
        break;
    }
  }

  return { ok: true, endpointId, taskId, invocation: Number.parseInt(invocation, 10), nonce };
}

// ---------------------------------------------------------------------------------------
// Namespaces (routing.go)
// ---------------------------------------------------------------------------------------

/**
 * The operator prefixes everything an endpoint registers with `<namespace>_` (workflow names,
 * the service part of action ids, event keys). Actions arrive prefixed; strip the prefix
 * before dispatching to user code. Applying it twice is harmless, as on the Go side.
 */
export function stripNamespace(name: string, namespace: string): string {
  const prefix = `${namespace}_`;

  return name.startsWith(prefix) ? name.slice(prefix.length) : name;
}

/** The inverse, for anything the endpoint pushes back to Hatchet (events, run triggers). */
export function applyNamespace(name: string, namespace: string): string {
  const prefix = `${namespace}_`;

  return name.startsWith(prefix) ? name : prefix + name;
}

// ---------------------------------------------------------------------------------------
// Healthcheck (serverless.proto ServerlessHealthcheckRequest / ServerlessHealthcheckResponse)
// ---------------------------------------------------------------------------------------

export interface HealthcheckRequest {
  endpointId: string;
  namespace: string;
  /** Unix seconds; an int64, so a decimal string on the wire. */
  timestamp: string;
}

/** protojson subset of v1.CreateTaskOpts (workflows.proto). */
export interface TaskDefinition {
  /** (required) the task name */
  readableId: string;
  /** (required) `service:verb`, un-prefixed; the operator namespaces the service part */
  action: string;
  /** e.g. "60s" */
  timeout?: string;
  parents?: string[];
  retries?: number;
  backoffFactor?: number;
  backoffMaxSeconds?: number;
  scheduleTimeout?: string;
  /** true for tasks that run over the durable websocket protocol */
  isDurable?: boolean;
}

/** protojson subset of v1.CreateWorkflowVersionRequest (workflows.proto), un-prefixed. */
export interface WorkflowDefinition {
  name: string;
  description?: string;
  version?: string;
  eventTriggers?: string[];
  cronTriggers?: string[];
  cronInput?: string;
  tasks: TaskDefinition[];
  onFailureTask?: TaskDefinition;
  defaultPriority?: number;
}

export interface HealthcheckResponse {
  workflows: WorkflowDefinition[];
  /** action ids served in addition to the ones derived from workflows */
  actions?: string[];
  durable: { supported: boolean };
  runtime: { name: string; sdkVersion: string };
}

export const SDK_VERSION = "0.1.0";

/**
 * Builds the healthcheck body. `workflows` map 1:1 onto CreateWorkflowVersionRequest and are
 * parsed with protojson (unknown fields ignored); the operator applies the namespace, derives
 * the action set and registers the workflows when the canonical response changes.
 */
export function healthcheckResponse(
  workflows: WorkflowDefinition[],
  opts: { durable?: boolean; actions?: string[] } = {},
): HealthcheckResponse {
  return {
    workflows,
    ...(opts.actions ? { actions: opts.actions } : {}),
    durable: { supported: opts.durable ?? true },
    runtime: { name: "cloudflare-workers", sdkVersion: SDK_VERSION },
  };
}

// ---------------------------------------------------------------------------------------
// Assigned actions (dispatcher.proto AssignedAction, protojson)
// ---------------------------------------------------------------------------------------

export type ActionType = "START_STEP_RUN" | "CANCEL_STEP_RUN" | "START_GET_GROUP_KEY" | "START_BATCH";

export interface AssignedAction {
  tenantId?: string;
  workflowRunId?: string;
  jobId?: string;
  /** the workflow name, namespaced as registered */
  jobName?: string;
  jobRunId?: string;
  taskId?: string;
  /** the task run external id; the durable task id for durable tasks */
  taskRunExternalId: string;
  /** namespaced `<ns>_service:verb` */
  actionId: string;
  /** protojson omits the zero value START_STEP_RUN */
  actionType?: ActionType;
  /** a JSON string: {"input": ..., "parents": {...}, "triggered_by": ...} */
  actionPayload?: string;
  taskName?: string;
  retryCount?: number;
  additionalMetadata?: string;
  priority?: number;
  workflowId?: string;
  workflowVersionId?: string;
  durableTaskInvocationCount?: number;
  triggeringEventExternalId?: string;
  triggeringEventKey?: string;
}

/** The non-durable trigger body (serverless.proto ServerlessTriggerRequest). */
export interface TriggerRequest {
  version: number;
  endpointId: string;
  namespace: string;
  /** Unix seconds; an int64, so a decimal string on the wire. */
  timestamp: string;
  action: AssignedAction;
}

/** The optional non-2xx body that overrides the status mapping (ServerlessTriggerError). */
export interface TriggerError {
  error: string;
  retry?: boolean;
}

export interface ActionInput<T = unknown> {
  input: T;
  parents?: Record<string, unknown>;
  triggered_by?: string;
  additional_metadata?: Record<string, string>;
}

/** Parses AssignedAction.actionPayload. An empty payload means an empty input. */
export function parseActionInput<T = unknown>(action: AssignedAction): ActionInput<T> {
  if (!action.actionPayload) {
    return { input: {} as T };
  }

  return JSON.parse(action.actionPayload) as ActionInput<T>;
}

// ---------------------------------------------------------------------------------------
// Durable protocol (serverless.proto ServerlessDurableFrame and its payloads)
// ---------------------------------------------------------------------------------------

/**
 * ServerlessFirstFrame: the payload of the first frame the operator sends after the 101,
 * as {"first": FirstFrame}.
 */
export interface FirstFrame {
  action: AssignedAction;
  namespace: string;
  invocationCount: number;
  inlineWaitBudgetMs: number;
}

/** dispatcher.proto DurableEventLogEntryRef; branchId and nodeId are int64, so strings. */
export interface LogEntryRef {
  durableTaskExternalId?: string;
  invocationCount?: number;
  branchId?: string | number;
  nodeId?: string | number;
}

export interface BaseMatchCondition {
  readableDataKey: string;
  action: "CREATE" | "QUEUE" | "CANCEL" | "SKIP";
  orGroupId: string;
  expression?: string;
}

export interface SleepMatchCondition {
  base: BaseMatchCondition;
  /** a duration string such as "3000ms" */
  sleepFor: string;
}

export interface UserEventMatchCondition {
  base: BaseMatchCondition;
  userEventKey: string;
  eventScope?: string;
}

export interface DurableEventListenerConditions {
  sleepConditions?: SleepMatchCondition[];
  userEventConditions?: UserEventMatchCondition[];
}

/** The oneof payloads of DurableTaskRequest an endpoint may send. */
export type DurableTaskRequest =
  | { memo: { invocationCount: number; durableTaskExternalId: string; key: string; payload?: string } }
  | { completeMemo: { ref: LogEntryRef; payload: string; memoKey: string } }
  | {
      waitFor: {
        invocationCount: number;
        durableTaskExternalId: string;
        waitForConditions: DurableEventListenerConditions;
        label?: string;
      };
    }
  | { evictInvocation: { invocationCount: number; durableTaskExternalId: string; reason?: string } };

export interface EntryCompleted {
  ref: LogEntryRef;
  /** base64 bytes */
  payload?: string;
  isFailure?: boolean;
  errorMessage?: string;
}

/** The oneof payloads of DurableTaskResponse the core forwards. */
export interface DurableTaskResponse {
  memoAck?: { ref: LogEntryRef; memoAlreadyExisted?: boolean; memoResultPayload?: string };
  waitForAck?: { ref: LogEntryRef };
  entryCompleted?: EntryCompleted;
  evictionAck?: { invocationCount?: number; durableTaskExternalId?: string };
  serverEvict?: { durableTaskExternalId?: string; invocationCount?: number; reason?: string };
  triggerRunsAck?: unknown;
}

/**
 * A frame the operator sends: exactly one of first (once, right after the upgrade), response
 * or error ({"code", "message"}; "nondeterminism" when a replay diverged from the log).
 */
export interface OutboundFrame {
  first?: FirstFrame;
  response?: DurableTaskResponse;
  error?: { code: string; message: string };
}

/**
 * ServerlessDoneFrame, sent as {"done": ...}. Checked by the relay in this order: status
 * "evicted" (no terminal event), error (FAILED; retry decides), otherwise output (COMPLETED,
 * {} when absent). The output is the task's result as JSON text, so any JSON value works and
 * the operator hands it to the engine byte for byte.
 */
export type DoneOutcome =
  | { output: string }
  | { error: string; retry: boolean }
  | { status: "evicted" };

/** Thrown inside a durable task once the invocation was evicted; let it propagate. */
export class Evicted extends Error {
  constructor(
    readonly source: "endpoint" | "server",
    reason?: string,
  ) {
    super(`invocation evicted by ${source}${reason ? `: ${reason}` : ""}`);
    this.name = "Evicted";
  }
}

/** An engine-side error for the invocation, e.g. a non-determinism error on replay. */
export class EngineError extends Error {
  constructor(
    readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "EngineError";
  }
}

function b64encode(text: string): string {
  const bytes = encoder.encode(text);
  let binary = "";

  for (const b of bytes) {
    binary += String.fromCharCode(b);
  }

  return btoa(binary);
}

function b64decode(b64: string): string {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);

  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }

  return decoder.decode(bytes);
}

function refKey(ref: LogEntryRef | undefined): string {
  return `${ref?.branchId ?? 0}:${ref?.nodeId ?? 0}`;
}

function formatDuration(ms: number): string {
  if (ms % 1000 === 0) {
    return `${ms / 1000}s`;
  }

  return `${ms}ms`;
}

interface Waiter {
  resolve: (entry: EntryCompleted) => void;
  reject: (err: Error) => void;
}

/**
 * The endpoint side of one durable invocation over the operator-dialed websocket.
 *
 * Wire rules (durable/relay.go):
 * - Endpoint frames are {"id": n, "request": <DurableTaskRequest>} and one final {"done": ...}.
 * - At most one ack-bearing request (memo, wait_for, trigger_runs) may be in flight; a second
 *   one closes the socket with 4006. complete_memo has no ack.
 * - The relay stamps durableTaskExternalId and invocationCount; values that are set must match
 *   the invocation's or the socket closes with 4004. This client always sets them.
 * - A socket that closes without a done frame is reported FAILED (retryable).
 *
 * Replay: every invocation re-executes the task from the top. memo answers from the log when
 * the key exists; wait_for on a satisfied entry gets its ack and entry_completed right away.
 */
export class DurableClient {
  readonly action: AssignedAction;
  readonly namespace: string;
  readonly taskId: string;
  readonly invocation: number;
  readonly inlineWaitBudgetMs: number;

  private seq = 0;
  private pendingAck: { resolve: (r: DurableTaskResponse) => void; reject: (e: Error) => void } | null =
    null;
  private waiters = new Map<string, Waiter>();
  private earlyEntries = new Map<string, EntryCompleted>();
  private terminal: Error | null = null;
  private finished = false;

  constructor(
    private readonly socket: WebSocket,
    first: FirstFrame,
    private readonly log: (msg: string) => void = () => {},
  ) {
    this.action = first.action;
    this.namespace = first.namespace;
    this.taskId = first.action.taskRunExternalId;
    this.invocation = first.invocationCount;
    this.inlineWaitBudgetMs = first.inlineWaitBudgetMs;
  }

  /**
   * Checks the first frame against the upgrade that was verified: the signature covered the
   * task id and invocation in the headers, not the frame, so a frame naming another task
   * is not authenticated and must not run. Returns the mismatch, or null when they agree.
   */
  assertMatches(verified: { taskId: string; invocation: number }): string | null {
    if (this.taskId !== verified.taskId) {
      return `first frame task ${this.taskId} does not match the verified upgrade`;
    }

    if (this.invocation !== verified.invocation) {
      return `first frame invocation ${this.invocation} does not match the verified upgrade`;
    }

    return null;
  }

  /** Feed every text frame after the first one here. */
  onMessage(raw: string): void {
    let frame: OutboundFrame;

    try {
      frame = JSON.parse(raw) as OutboundFrame;
    } catch {
      this.failAll(new Error("malformed frame from the operator"));
      return;
    }

    if (frame.error) {
      // Non-determinism on replay and similar. The relay expects done {"error", "retry": false}
      // next; the task's catch block does that.
      this.failAll(new EngineError(frame.error.code, frame.error.message));
      return;
    }

    const resp = frame.response;

    if (!resp) {
      return;
    }

    if (resp.serverEvict) {
      // The engine superseded this invocation. The relay closes the socket with 4001 after
      // forwarding this; nothing more may be sent.
      this.terminal = new Evicted("server", resp.serverEvict.reason);
      this.finished = true;
      this.failAll(this.terminal);
      return;
    }

    if (resp.entryCompleted) {
      const key = refKey(resp.entryCompleted.ref);
      const waiter = this.waiters.get(key);

      if (waiter) {
        this.waiters.delete(key);
        waiter.resolve(resp.entryCompleted);
      } else {
        this.earlyEntries.set(key, resp.entryCompleted);
      }

      return;
    }

    // memoAck, waitForAck, evictionAck, triggerRunsAck: the ack of the single in-flight request.
    const pending = this.pendingAck;

    if (pending) {
      this.pendingAck = null;
      pending.resolve(resp);
    }
  }

  /** Call from the socket's close and error events. */
  onClose(reason: string): void {
    this.finished = true;
    this.failAll(this.terminal ?? new Error(`websocket closed: ${reason}`));
  }

  /**
   * Memoizes fn's result under key. The first invocation runs fn and records the JSON result
   * with complete_memo; replays get memo_already_existed with the recorded payload and skip fn.
   */
  async memo<T>(key: string, fn: () => Promise<T> | T): Promise<T> {
    const ack = await this.request({
      memo: { invocationCount: this.invocation, durableTaskExternalId: this.taskId, key: b64encode(key) },
    });

    const memoAck = ack.memoAck;

    if (!memoAck?.ref) {
      throw new Error("memo ack without a ref");
    }

    if (memoAck.memoAlreadyExisted && memoAck.memoResultPayload) {
      this.log(`memo ${key}: replayed from the log`);
      return JSON.parse(b64decode(memoAck.memoResultPayload)) as T;
    }

    const value = await fn();
    const payload = JSON.stringify(value === undefined ? null : value);

    // complete_memo carries no ack, so it is not subject to the one-in-flight rule.
    this.send({
      id: ++this.seq,
      request: { completeMemo: { ref: memoAck.ref, memoKey: b64encode(key), payload: b64encode(payload) } },
    });

    this.log(`memo ${key}: computed and recorded`);

    return value;
  }

  /**
   * Registers a wait_for and blocks up to inlineWaitBudgetMs for entry_completed. When the
   * budget elapses the invocation evicts itself (evict_invocation, eviction_ack, done evicted)
   * and this throws Evicted; the engine re-invokes the task with invocationCount + 1 once the
   * entry is satisfied, and the replayed wait_for resolves immediately.
   */
  async waitFor(conditions: DurableEventListenerConditions, label?: string): Promise<EntryCompleted> {
    const ack = await this.request({
      waitFor: {
        invocationCount: this.invocation,
        durableTaskExternalId: this.taskId,
        waitForConditions: conditions,
        ...(label ? { label } : {}),
      },
    });

    const ref = ack.waitForAck?.ref;

    if (!ref) {
      throw new Error("wait_for ack without a ref");
    }

    const completed = await this.awaitEntry(ref, this.inlineWaitBudgetMs);

    if (completed === null) {
      this.log(`wait_for ${label ?? ""}: inline budget of ${this.inlineWaitBudgetMs}ms elapsed, evicting`);
      return this.evict(`inline wait budget of ${this.inlineWaitBudgetMs}ms elapsed`);
    }

    if (completed.isFailure) {
      throw new Error(completed.errorMessage ?? "wait_for entry failed");
    }

    return completed;
  }

  /** A durable sleep: wait_for with a single sleep condition, the way the Go SDK's SleepFor does. */
  async sleep(ms: number): Promise<void> {
    const human = formatDuration(ms);

    await this.waitFor(
      {
        sleepConditions: [
          {
            base: {
              readableDataKey: `sleep:${human}`,
              action: "CREATE",
              orGroupId: crypto.randomUUID(),
              expression: "",
            },
            sleepFor: `${ms}ms`,
          },
        ],
      },
      `sleep ${human}`,
    );
  }

  /**
   * Evicts this invocation: evict_invocation, await eviction_ack, send done {"status": "evicted"}
   * (which must be the last frame) and throw Evicted so the task unwinds without touching the
   * socket again.
   */
  async evict(reason: string): Promise<never> {
    const ack = await this.request({
      evictInvocation: { invocationCount: this.invocation, durableTaskExternalId: this.taskId, reason },
    });

    if (!ack.evictionAck) {
      throw new Error("expected an eviction_ack");
    }

    this.terminal = new Evicted("endpoint", reason);
    this.done({ status: "evicted" });

    throw this.terminal;
  }

  /** Sends the terminal frame. Exactly one is sent; later calls are ignored. */
  done(outcome: DoneOutcome): void {
    if (this.finished) {
      return;
    }

    this.finished = true;
    this.send({ done: outcome });
  }

  get isFinished(): boolean {
    return this.finished;
  }

  private request(request: DurableTaskRequest): Promise<DurableTaskResponse> {
    if (this.terminal) {
      return Promise.reject(this.terminal);
    }

    if (this.finished) {
      return Promise.reject(new Error("durable invocation already finished"));
    }

    if (this.pendingAck) {
      // The relay would close 4006; fail locally instead.
      return Promise.reject(new Error("a durable request is already awaiting its ack"));
    }

    return new Promise((resolve, reject) => {
      this.pendingAck = { resolve, reject };
      this.send({ id: ++this.seq, request });
    });
  }

  private awaitEntry(ref: LogEntryRef, budgetMs: number): Promise<EntryCompleted | null> {
    const key = refKey(ref);
    const early = this.earlyEntries.get(key);

    if (early) {
      this.earlyEntries.delete(key);
      return Promise.resolve(early);
    }

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.waiters.delete(key);
        resolve(null);
      }, budgetMs);

      this.waiters.set(key, {
        resolve: (entry) => {
          clearTimeout(timer);
          resolve(entry);
        },
        reject: (err) => {
          clearTimeout(timer);
          reject(err);
        },
      });
    });
  }

  private failAll(err: Error): void {
    const pending = this.pendingAck;
    this.pendingAck = null;
    pending?.reject(err);

    for (const waiter of this.waiters.values()) {
      waiter.reject(err);
    }

    this.waiters.clear();
  }

  private send(frame: unknown): void {
    this.socket.send(JSON.stringify(frame));
  }
}

// ---------------------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------------------

export function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

/** A non-2xx trigger response with the {"error", "retry"} override body. */
export function triggerError(status: number, error: string, retry: boolean): Response {
  return json({ error, retry } satisfies TriggerError, status);
}
