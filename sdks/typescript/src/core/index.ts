/**
 * The core entry point: the application client for runtimes without gRPC. It triggers runs,
 * pushes events, waits for results by polling, publishes to durable streams and manages
 * workflows over unary Connect calls on a fetch transport, reaches the REST API through a
 * client generated for `fetch`, and it is configured from an explicit object only.
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
export { toRunDetail, parseTaskOutput, runStatusToJSON } from './run-detail';
export { ConsoleLogger, consoleLogger } from './logger';
export type {
  CallOptions,
  CancelRunOpts,
  CoreClientConfig,
  CreateRateLimitOpts,
  PauseRateLimitOpts,
  DesiredWorkerLabelOpt,
  EventWithMetadata,
  LogConstructor,
  PushEventOptions,
  ReplayRunOpts,
  ResultOptions,
  RunDetail,
  RunFilter,
  RunFilterBase,
  StreamCallOptions,
  StreamTopicMetadata,
  TaskRunDetail,
  TriggerRun,
  TriggerRunOptions,
} from './types';

// Feature clients
export { EventsClient } from './features/events';
export { RunsClient } from './features/runs';
export { WorkflowsClient } from './features/workflows';
export { RateLimitsClient } from './features/rate-limits';
export { LogsClient, MAX_LOG_LINE_CHARS } from './features/logs';
export type { PutLogOptions } from './features/logs';
export { StreamsClient, MAX_STREAM_MESSAGE_BYTES } from './features/streams';
export type { StreamsClientDeps } from './features/streams';
export { TasksClient } from './features/tasks';
export type { RunsClientDeps } from './features/runs';
export type { WorkflowsClientDeps } from './features/workflows';
export type { Duration, DurationObject } from '@hatchet/v1/client/duration';

// REST API and the feature clients built on it, shared with the Node client
export type { Api as RestApiClient } from '@hatchet/clients/rest/generated-fetch/Api';
export type {
  HttpResponse,
  RequestParams,
} from '@hatchet/clients/rest/generated-fetch/http-client';
export type { RestApi, RestClientDeps, RestRequestParams } from '@hatchet/clients/rest/rest-api';
export { restErrors } from '@hatchet/clients/rest/errors';
export { RestRunsClient } from '@hatchet/clients/rest/runs';
export type { ListRunsOpts, RunFilter as RestRunFilter } from '@hatchet/clients/rest/runs';
export { CELClient } from '@hatchet/v1/client/features/cel';
export type { CELEvaluationResult, DebugCELInput } from '@hatchet/v1/client/features/cel';
export { CronClient } from '@hatchet/v1/client/features/crons';
export type { CreateCronInput } from '@hatchet/v1/client/features/crons';
export { FiltersClient } from '@hatchet/v1/client/features/filters';
export { MetricsClient } from '@hatchet/v1/client/features/metrics';
export type { TaskStatusMetrics } from '@hatchet/v1/client/features/metrics';
export { ScheduleClient } from '@hatchet/v1/client/features/schedules';
export type {
  CreateScheduledRunInput,
  UpdateScheduledRunInput,
} from '@hatchet/v1/client/features/schedules';
export { TenantClient } from '@hatchet/v1/client/features/tenant';
export { WebhooksClient } from '@hatchet/v1/client/features/webhooks';
export type { CreateWebhookOptions } from '@hatchet/v1/client/features/webhooks';
export { WorkersClient } from '@hatchet/v1/client/features/workers';
export {
  WorkflowPauseScheduledCronRunQueueBehavior,
  workflowNameString,
} from '@hatchet/v1/client/features/workflows';
export type { PauseWorkflowOpts } from '@hatchet/v1/client/features/workflows';

// Transport
export {
  createFetchTransport,
  limitMessageSizes,
  resolveServerUrl,
  type FetchTlsConfig,
  type FetchTransportOptions,
  type MessageSizeLimits,
} from '@clients/transport/fetch-transport';
export {
  createAuthInterceptor,
  DEFAULT_MAX_MESSAGE_BYTES,
  type Transport,
} from '@clients/transport/transport';

// Wire enums and errors callers switch on
export { LogLevel } from '@hatchet/clients/event/rpc';
export { Priority } from '@hatchet/v1/declaration';
export type { RunManyOpt, RunOpts } from '@hatchet/v1/declaration';
export { RateLimitDuration } from '@hatchet/protoc/workflows/workflows';
export { WorkerLabelComparator } from '@hatchet/protoc/v1/shared/trigger';
export { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
// HatchetError is imported and re-exported by name rather than as `default as`: tsc emits a
// getter for the latter that Node's CommonJS export detection cannot see, which hides the name
// from ESM importers of this entry.
import HatchetError from '@hatchet/util/errors/hatchet-error';

export { HatchetError };
export { getErrorMessage, toHatchetError } from '@hatchet/util/errors/hatchet-error';
export { IdempotencyCollisionError } from '@hatchet/util/errors/idempotency-collision-error';
export { BulkTriggerIdempotencyCollisionError } from '@hatchet/util/errors/bulk-trigger-idempotency-collision-error';
export { BulkTriggerPartialError } from '@hatchet/util/errors/bulk-trigger-partial-error';
export { AbortError, isAbortError } from '@hatchet/util/abort-error';
