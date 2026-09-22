/**
 * The core entry point: the application client for runtimes without gRPC. It triggers runs,
 * pushes events, waits for results by polling and manages workflows over unary Connect calls
 * on a fetch transport, and it is configured from an explicit object only.
 *
 * Nothing exported here imports from Node. `scripts/check-edge-entry.mjs` bundles this module
 * for a browser-like target and fails the build if a Node builtin, `axios` or a gRPC package
 * sneaks in.
 *
 * ```typescript
 * import { Hatchet } from '@hatchet-dev/typescript-sdk/core';
 *
 * const hatchet = new Hatchet({ token: env.HATCHET_CLIENT_TOKEN });
 * const ref = await hatchet.runNoWait('greet', { name: 'world' });
 * const output = await ref.result({ timeoutMs: 30_000 });
 * ```
 * @module Core
 */

/** Bumped whenever the surface exported from this entry changes incompatibly. */
export const CORE_ENTRY_VERSION = 1;

export { HatchetCore, HatchetCore as Hatchet } from './client';
export type { ResolvedCoreConfig, WorkflowRef } from './client';
export { WorkflowRunRef, INITIAL_POLL_INTERVAL_MS, MAX_POLL_INTERVAL_MS } from './run-ref';
export type { RunRefClient } from './run-ref';
export { toRunDetail, runStatusToJSON } from './run-detail';
export { ConsoleLogger, consoleLogger } from './logger';
export type {
  CallOptions,
  CancelRunOpts,
  CoreClientConfig,
  CreateRateLimitOpts,
  DesiredWorkerLabelOpt,
  EventWithMetadata,
  LogConstructor,
  PushEventOptions,
  ReplayRunOpts,
  ResultOptions,
  RunDetail,
  RunFilter,
  RunFilterBase,
  TaskRunDetail,
  TriggerRun,
  TriggerRunOptions,
} from './types';

// Feature clients
export { EventsClient } from './features/events';
export { RunsClient } from './features/runs';
export { WorkflowsClient } from './features/workflows';
export { RateLimitsClient } from './features/rate-limits';
export { LogsClient } from './features/logs';
export type { PutLogOptions } from './features/logs';
export { StreamsClient } from './features/streams';

// Transport
export {
  createFetchTransport,
  resolveServerUrl,
  type FetchTlsConfig,
  type FetchTransportOptions,
} from '@clients/transport/fetch-transport';
export { createAuthInterceptor, type Transport } from '@clients/transport/transport';

// Wire enums and errors callers switch on
export { LogLevel } from '@hatchet/clients/event/rpc';
export { Priority } from '@hatchet/v1/declaration';
export type { RunManyOpt, RunOpts } from '@hatchet/v1/declaration';
export { RateLimitDuration } from '@hatchet/protoc/workflows/workflows';
export { WorkerLabelComparator } from '@hatchet/protoc/v1/shared/trigger';
export { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
export {
  default as HatchetError,
  getErrorMessage,
  toHatchetError,
} from '@hatchet/util/errors/hatchet-error';
export { IdempotencyCollisionError } from '@hatchet/util/errors/idempotency-collision-error';
export { BulkTriggerIdempotencyCollisionError } from '@hatchet/util/errors/bulk-trigger-idempotency-collision-error';
export { AbortError, isAbortError } from '@hatchet/util/abort-error';
