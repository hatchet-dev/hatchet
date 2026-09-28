/**
 * `@hatchet-dev/serverless`: serve Hatchet tasks from a serverless runtime without a worker
 * process. Declare tasks with `hatchet.task(...)`, hand them to an adapter
 * (`@hatchet-dev/serverless/cloudflare`) and register the endpoint with Hatchet; the
 * serverless operator polls the endpoint and delivers runs to it over signed requests.
 *
 * The handler takes an optional `client` (the SDK's core client, built from a token in the
 * platform's secret store) for the `ctx` methods that reach the engine; child results are
 * awaited over the invocation socket. See LIMITATIONS.md for what `ctx` can and cannot do.
 *
 * Nothing exported here imports from Node or from the SDK outside its edge and core entries.
 * @module Serverless
 */
import { declarations } from '@hatchet-dev/typescript-sdk/edge/index.js';
import type { Declarations } from '@hatchet-dev/typescript-sdk/edge/index.js';

/**
 * The declaration factories (`task`, `durableTask`, `workflow`, `batchTask`) bound to no
 * client. `run`, `schedule` and `cron` on the declarations throw; trigger runs through a
 * `HatchetCore` (`hatchet.run(echo, input)` accepts the declaration) or the regular SDK.
 */
export const hatchet: Declarations = declarations();

export { createHandler, normalizeBasePath } from './handler';
export type {
  DurableHooks,
  DurableSocket,
  EnvResolver,
  HandlerOptions,
  ServerlessHandler,
  ServerlessRuntimeName,
} from './handler';
export { createClientResolver, isHatchetCore } from './handler/client';
export type { ClientOption, ClientSource } from './handler/client';
export { ServerlessLimitationError, isServerlessLimitationError } from './handler/errors';
export { ServerlessRuntime, ConsoleLogger } from './handler/context';
export type { ConsoleLike, ServerlessRuntimeOptions } from './handler/context';
export { RunStreamClosedError, RunWatcher, TaskRunRef, outputsOf } from './handler/runs';
export { FrameTransport } from './handler/durable/transport';
export type { FrameTransportOptions } from './handler/durable/transport';
export {
  LISTEN_FOR_DURABLE_EVENT,
  MAX_SOCKET_STREAMS,
  SOCKET_STREAM_PROCEDURES,
  SUBSCRIBE_TO_WORKFLOW_EVENTS,
  SUBSCRIBE_TO_WORKFLOW_RUNS,
  SocketStreams,
  StreamCloseCode,
  StreamOpenError,
} from './handler/durable/streams';
export type { SocketStream, SocketStreamsOptions, StreamListener } from './handler/durable/streams';
export { runDurableInvocation } from './handler/durable/invocation';
export type { DurableInvocationOptions } from './handler/durable/invocation';
export { decodeFrame, encodeFrame, frameKind } from './handler/durable/frames';
export type { ServeEntry } from './handler/registry';

// The core client, so a Worker can build one for its own handlers with one import.
export { HatchetCore, WorkflowRunRef } from '@hatchet-dev/typescript-sdk/core/index.js';
export type { CoreClientConfig, ResultOptions } from '@hatchet-dev/typescript-sdk/core/index.js';
export {
  constantTimeEqual,
  isFreshTimestamp,
  parseTimestamp,
  signHex,
  verifyBodySignature,
  verifySignedBody,
  verifyUpgradeSignature,
} from './handler/signature';
export type {
  BodyVerification,
  UpgradeVerification,
  VerifyBodyOptions,
  VerifyUpgradeOptions,
} from './handler/signature';
export { NonceSet } from './handler/nonce-set';
export type { NonceOutcome } from './handler/nonce-set';
export * from './handler/contract';

// The SDK surface a task author needs.
export {
  Context,
  DurableContext,
  NonRetryableError,
  TaskRunTerminatedError,
  BaseWorkflowDeclaration,
  TaskWorkflowDeclaration,
  WorkflowDeclaration,
  Priority,
} from '@hatchet-dev/typescript-sdk/edge/index.js';
export type {
  Declarations,
  CreateTaskWorkflowOpts,
  CreateDurableTaskWorkflowOpts,
  CreateWorkflowOpts,
  CreateBatchTaskWorkflowOpts,
  CreateWorkflowTaskOpts,
  CreateWorkflowDurableTaskOpts,
  WorkflowDefinition,
  TaskOutput,
  TaskOutputType,
  InputType,
  OutputType,
  JsonObject,
  Duration,
  ContextRuntime,
  DurableTransport,
} from '@hatchet-dev/typescript-sdk/edge/index.js';

// The wire contract, generated from api-contracts/v1/serverless.proto.
export type {
  ServerlessHealthcheckRequest,
  ServerlessHealthcheckResponse,
  ServerlessTriggerRequest,
  ServerlessTriggerError,
  ServerlessDurableFrame,
  ServerlessFirstFrame,
  ServerlessDoneFrame,
  ServerlessErrorFrame,
  ServerlessStreamOpen,
  ServerlessStreamMessage,
  ServerlessStreamClose,
  ServerlessTaskOptions,
} from './generated/proto/v1/serverless';
export type { WorkflowRunEvent, StepRunResult } from './generated/proto/dispatcher';
