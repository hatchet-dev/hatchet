/**
 * The option and result types shared by the core client and the Node client. Everything in
 * this module is edge-safe: it imports nothing from Node, so the Node client's types extend
 * these where they need more.
 */
import type { Transport } from '@connectrpc/connect';
import type { FetchTlsConfig } from '@clients/transport/fetch-transport';
import type { Priority } from '@hatchet/v1/declaration';
import type { WorkerLabelComparator } from '@hatchet/protoc/v1/shared/trigger';
import type { RateLimitDuration } from '@hatchet/protoc/workflows/workflows';
import type { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import type { RetrierConfig } from '@hatchet/util/retrier';
import type { Logger, LogLevel } from '@hatchet/util/logger/logger';
import type { UnaryCallOptions } from '@clients/transport/ts-proto-client';

/** Builds the logger a client component logs through. */
export type LogConstructor = (context: string, logLevel?: LogLevel) => Logger;

/**
 * Per-call options for a unary RPC: `signal` aborts the call, `deadline` (a `Date` or epoch
 * milliseconds) bounds it, `metadata` adds request headers. The same options the Node
 * client's RPC clients take.
 */
export type CallOptions = UnaryCallOptions;

/**
 * The core client's configuration. It is explicit by design: nothing is read from the
 * environment, a config file or the filesystem, so the same object works in a Worker, a
 * browser bundle and Node.
 */
export interface CoreClientConfig {
  /** The tenant API token. */
  token: string;
  /**
   * The engine's base URL, for example `https://engine.example.com:7070`: an absolute
   * `http://` or `https://` URL with no username, password, query string or fragment, whose
   * scheme agrees with `tls` (`http://` needs `{ strategy: 'none' }`). When neither this nor
   * `hostPort` is set, the address is read from the token's `grpc_broadcast_address` claim,
   * so `{ token }` alone reaches the engine the token was issued for.
   */
  serverUrl?: string;
  /** The engine's `host:port`, combined with `tls` into the base URL. */
  hostPort?: string;
  /**
   * Defaults to `{ strategy: 'tls' }`. The token travels on every call, so plain HTTP has to
   * be chosen explicitly with `{ strategy: 'none' }`.
   */
  tls?: FetchTlsConfig;
  /**
   * The prefix applied to workflow names and event keys, the same way the Node client applies
   * `HATCHET_CLIENT_NAMESPACE`: it is lowercased and gets a trailing underscore when missing.
   */
  namespace?: string;
  /** A ready transport, in place of the fetch transport the client builds from the fields above. */
  transport?: Transport;
  /** The `fetch` the transport sends with. Defaults to the runtime's global `fetch`. */
  fetch?: typeof globalThis.fetch;
  /** Retry settings for the unary calls, the same shape as the Node client's `retrier`. */
  retrier?: RetrierConfig;
  logger?: LogConstructor;
  logLevel?: LogLevel;
}

export type DesiredWorkerLabelOpt = {
  value: string | number;
  required?: boolean;
  weight?: number;
  comparator?: WorkerLabelComparator;
};

/**
 * The options a single `TriggerWorkflow` call accepts. The Node client's `admin.runWorkflow`
 * takes exactly this type.
 */
export type TriggerRunOptions = {
  parentId?: string | undefined;
  /**
   * (optional) the parent task run external id.
   *
   * This is the field understood by the workflows gRPC API (`parent_task_run_external_id`).
   */
  parentTaskRunExternalId?: string | undefined;
  /**
   * @deprecated Use `parentTaskRunExternalId` instead.
   * Kept for backward compatibility; will be mapped to `parentTaskRunExternalId`.
   */
  parentStepRunId?: string | undefined;
  childIndex?: number | undefined;
  childKey?: string | undefined;
  additionalMetadata?: Record<string, string> | undefined;
  desiredWorkerId?: string | undefined;
  priority?: Priority;
  desiredWorkerLabels?: Record<string, DesiredWorkerLabelOpt>;
  _standaloneTaskName?: string | undefined;
};

/** One entry of a bulk trigger. */
export type TriggerRun<I = object> = {
  workflowName: string;
  input: I;
  options?: TriggerRunOptions;
};

/** Options for pushing an event. */
export interface PushEventOptions {
  additionalMetadata?: Record<string, string>;
  priority?: number;
  scope?: string;
}

/** One event of a bulk push, with per-event overrides of the push options. */
export interface EventWithMetadata<T> {
  payload: T;
  additionalMetadata?: Record<string, unknown>;
  priority?: number;
  scope?: string;
}

export type CreateRateLimitOpts = {
  key: string;
  limit: number;
  duration?: RateLimitDuration;
};

/** The state of one task of a run, as `GetRunDetails` reports it. */
export type TaskRunDetail = {
  externalId: string;
  readableId: string;
  status: V1TaskStatus;
  /** The output decoded as JSON; `null` when the task stored nothing, `null` or text that is not JSON. */
  output: unknown;
  /** The output text as the engine stored it; absent when the task stored nothing. */
  rawOutput?: string;
  error?: string;
  isEvicted: boolean;
};

/** The state of a run, as `GetRunDetails` reports it. `taskRuns` is keyed by readable id. */
export type RunDetail = {
  status: V1TaskStatus;
  done: boolean;
  input: unknown;
  additionalMetadata: unknown;
  isEvicted: boolean;
  taskRuns: Record<string, TaskRunDetail>;
};

/** The filter fields shared by the core and Node run filters. */
export type RunFilterBase = {
  /** Defaults to one hour ago. */
  since?: Date;
  until?: Date;
  statuses?: V1TaskStatus[];
  additionalMetadata?: Record<string, string>;
};

/**
 * The core client's run filter. It takes workflow ids rather than names because resolving a
 * name is a REST call, which the core client does not make.
 */
export type RunFilter = RunFilterBase & {
  workflowIds?: string[];
};

export type CancelRunOpts = {
  ids?: string[];
  filters?: RunFilter;
};

export type ReplayRunOpts = {
  ids?: string[];
  filters?: RunFilter;
};

/** Options for waiting on a run's result. */
export interface ResultOptions {
  /** Rejects with a `HatchetError` once this many milliseconds have passed. No limit by default. */
  timeoutMs?: number;
  /** Rejects with an `AbortError` when the signal fires. */
  signal?: AbortSignal;
}
