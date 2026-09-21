/**
 * The edge entry point: everything needed to declare Hatchet tasks and run them from a
 * runtime that has no Hatchet client, such as a Cloudflare Worker or a Vercel Function.
 *
 * Nothing exported here imports from Node. `scripts/check-edge-entry.mjs` bundles this
 * module for a browser-like target and fails the build if a Node builtin sneaks in.
 *
 * ```typescript
 * import { declarations, workflowToProto } from '@hatchet-dev/typescript-sdk/edge';
 *
 * const { task } = declarations();
 * const greet = task({ name: 'greet', fn: (input: { name: string }) => ({ hi: input.name }) });
 * const request = workflowToProto(greet, { namespace: 'prod_' });
 * ```
 * @module Edge
 */

// Declarations
export {
  BaseWorkflowDeclaration,
  CreateBatchTaskWorkflow,
  CreateDurableTaskWorkflow,
  CreateTaskWorkflow,
  CreateWorkflow,
  Priority,
  StickyStrategy,
  TaskWorkflowDeclaration,
  WorkflowDeclaration,
} from '@hatchet/v1/declaration';
export type {
  CreateBaseWorkflowOpts,
  CreateBatchTaskWorkflowOpts,
  CreateDurableTaskWorkflowOpts,
  CreateTaskWorkflowOpts,
  CreateWorkflowOpts,
  RunManyOpt,
  RunOpts,
  StickyStrategyInput,
  TaskDefaults,
  TaskOutput,
  TaskOutputType,
  WorkflowDefinition,
} from '@hatchet/v1/declaration';

// Tasks, types, durations and conditions
export * from '@hatchet/v1/task';
export * from '@hatchet/v1/types';
export * from '@hatchet/v1/client/duration';
export * from '@hatchet/v1/conditions';
export { conditionsToPb, taskConditionsToPb } from '@hatchet/v1/conditions/transformer';

// Naming
export { applyNamespace } from '@hatchet/util/apply-namespace';
export {
  createAction,
  createActionId,
  workflowNameFromAction,
} from '@hatchet/clients/dispatcher/action';
export type { Action, ActionKey } from '@hatchet/clients/dispatcher/action';

// Contexts and the runtime seams they depend on
export {
  Context,
  ContextWorker,
  DurableContext,
  computeMemoKey,
} from '@hatchet/v1/client/worker/context';
export type { SleepForOptions, SleepResult } from '@hatchet/v1/client/worker/context';
export type {
  CancelBatchRequest,
  ContextRuntime,
  DesiredWorkerLabel,
  DurableContextOptions,
  DurableTransport,
  SpawnRunOptions,
  SpawnRunRequest,
  WorkerLabels,
} from '@hatchet/v1/client/worker/runtime';
export { isContextRuntime } from '@hatchet/v1/client/worker/runtime';
export type {
  DurableTaskEventAck,
  DurableTaskEventLogEntryResult,
  DurableTaskEventMemoAck,
  DurableTaskEventRunAck,
  DurableTaskEventWaitForAck,
  DurableTaskRunAckEntryResult,
  DurableTaskSendEvent,
  MemoEvent,
  RunChildrenEvent,
  WaitForEvent,
} from '@hatchet/clients/listeners/durable-listener/durable-events';
export { Logger, LogLevelEnum } from '@hatchet/util/logger/logger';
export type { LogExtra, LogLevel } from '@hatchet/util/logger/logger';
export {
  ParentRunContextManager,
  parentRunContextManager,
} from '@hatchet/v1/parent-run-context-vars';
export type {
  ParentRunContext,
  ParentRunContextStorage,
} from '@hatchet/v1/parent-run-context-vars';

// Errors
export {
  default as HatchetError,
  getErrorMessage,
  toHatchetError,
} from '@hatchet/util/errors/hatchet-error';
export { NonDeterminismError } from '@hatchet/util/errors/non-determinism-error';
export {
  TaskRunTerminatedError,
  isTaskRunTerminatedError,
} from '@hatchet/util/errors/task-run-terminated-error';
export type { TaskRunTerminationReason } from '@hatchet/util/errors/task-run-terminated-error';
export {
  AbortError,
  createAbortError,
  isAbortError,
  rethrowIfAborted,
  throwIfAborted,
} from '@hatchet/util/abort-error';

// Eviction policy
export {
  DEFAULT_DURABLE_TASK_EVICTION_POLICY,
  EvictionPolicy,
} from '@hatchet/v1/client/worker/eviction/eviction-policy';
export { MinEngineVersion, supportsEviction } from '@hatchet/v1/client/worker/engine-version';

// Registration
export {
  ON_FAILURE_TASK_NAME,
  ON_SUCCESS_TASK_NAME,
  normalizeWorkflowDefinition,
  onFailureTaskName,
  workflowToProto,
} from '@hatchet/v1/client/worker/workflow-proto';
export type { WorkflowProtoOptions } from '@hatchet/v1/client/worker/workflow-proto';

// Wire types
export { CreateWorkflowVersionRequest } from '@hatchet/protoc/v1/workflows';
export { AssignedAction } from '@hatchet/protoc/dispatcher';
export { DurableTaskRequest, DurableTaskResponse } from '@hatchet/protoc/v1/dispatcher';

export { declarations } from './declarations';
export type { Declarations } from './declarations';
