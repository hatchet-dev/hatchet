/**
 * Turns a workflow declaration into the `CreateWorkflowVersionRequest` the engine
 * registers it under. Pure: nothing here talks to the engine or imports from Node, so
 * a serverless runtime can register the same declarations a worker does.
 * @module WorkflowProto
 */
import HatchetError from '@util/errors/hatchet-error';
import { CreateStepRateLimit, StickyStrategy } from '@hatchet/protoc/workflows';
import {
  CreateTaskOpts,
  CreateWorkflowVersionRequest,
  IdempotencyMethod,
  TaskBatchConfig,
} from '@hatchet/protoc/v1/workflows';
import type { DesiredWorkerLabels } from '@hatchet/protoc/v1/shared/trigger';
import type { BaseWorkflowDeclaration, WorkflowDefinition } from '@hatchet/v1/declaration';
import type {
  Concurrency,
  CreateOnFailureTaskOpts,
  CreateOnSuccessTaskOpts,
  CreateWorkflowDurableTaskOpts,
  CreateWorkflowTaskOpts,
} from '@hatchet/v1/task';
import { taskConditionsToPb } from '@hatchet/v1/conditions/transformer';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import { createActionId } from '@hatchet/clients/dispatcher/action';
import * as z from 'zod/v4';
import { Duration, durationToMs, durationToString } from '../../client/duration';

export const ON_FAILURE_TASK_NAME = 'on-failure-task';
export const ON_SUCCESS_TASK_NAME = 'on-success-task';

export interface WorkflowProtoOptions {
  /** The namespace prefixed onto the workflow name and its event triggers. */
  namespace?: string;
  /**
   * When true the on-success task is left out, matching how a durable-only registration
   * behaves on the worker. Defaults to false.
   */
  durable?: boolean;
}

function definitionOf(
  definition: WorkflowDefinition | BaseWorkflowDeclaration<any, any>
): WorkflowDefinition {
  return 'definition' in definition ? definition.definition : definition;
}

/**
 * Applies the namespace to a workflow definition and appends the on-success task, if the
 * workflow declares one, as a regular task whose parents are the leaves of the DAG.
 * Returns a new definition; the declaration is not mutated. Idempotent, so a definition
 * that was already normalized comes back unchanged.
 */
export function normalizeWorkflowDefinition(
  definition: WorkflowDefinition | BaseWorkflowDeclaration<any, any>,
  opts: WorkflowProtoOptions = {}
): WorkflowDefinition {
  const source = definitionOf(definition);
  // The client lowercases its namespace; doing the same here keeps this idempotent when
  // a caller passes a mixed-case namespace directly.
  const workflow: WorkflowDefinition = {
    ...source,
    name: applyNamespace(source.name, opts.namespace?.toLowerCase()).toLowerCase(),
    _tasks: [...source._tasks],
    _durableTasks: [...source._durableTasks],
  };

  const alreadyHasOnSuccess = workflow._tasks.some((task) => task.name === ON_SUCCESS_TASK_NAME);
  const onSuccessTask = alreadyHasOnSuccess || opts.durable ? undefined : onSuccessTaskOf(workflow);

  if (onSuccessTask) {
    workflow._tasks.push(onSuccessTask);
  }

  return workflow;
}

function onSuccessTaskOf(
  workflow: WorkflowDefinition
): CreateWorkflowTaskOpts<any, any> | undefined {
  if (!workflow.onSuccess) {
    return undefined;
  }

  const parents = getLeaves([...workflow._tasks, ...workflow._durableTasks]);

  if (typeof workflow.onSuccess === 'function') {
    return {
      name: ON_SUCCESS_TASK_NAME,
      fn: workflow.onSuccess,
      executionTimeout: '60s',
      parents,
      retries: 0,
      rateLimits: [],
      desiredWorkerLabels: undefined,
      concurrency: [],
    };
  }

  const onSuccess = workflow.onSuccess as CreateOnSuccessTaskOpts<any, any>;

  return {
    name: ON_SUCCESS_TASK_NAME,
    fn: onSuccess.fn,
    executionTimeout:
      onSuccess.executionTimeout || workflow.taskDefaults?.executionTimeout || '60s',
    scheduleTimeout: onSuccess.scheduleTimeout || workflow.taskDefaults?.scheduleTimeout,
    parents,
    retries: onSuccess.retries || workflow.taskDefaults?.retries || 0,
    rateLimits: onSuccess.rateLimits || workflow.taskDefaults?.rateLimits,
    desiredWorkerLabels: onSuccess.desiredWorkerLabels || workflow.taskDefaults?.workerLabels,
    concurrency: onSuccess.concurrency || workflow.taskDefaults?.concurrency,
    backoff: onSuccess.backoff || workflow.taskDefaults?.backoff,
  };
}

function onFailureTaskOf(workflow: WorkflowDefinition): CreateTaskOpts | undefined {
  if (!workflow.onFailure) {
    return undefined;
  }

  if (typeof workflow.onFailure === 'function') {
    return {
      readableId: ON_FAILURE_TASK_NAME,
      action: onFailureTaskName(workflow),
      timeout: '60s',
      inputs: '{}',
      parents: [],
      retries: 0,
      rateLimits: [],
      workerLabels: {},
      concurrency: [],
      isDurable: false,
      slotRequests: { default: 1 },
    };
  }

  const onFailure = workflow.onFailure as CreateOnFailureTaskOpts<any, any>;
  const scheduleTimeout = onFailure.scheduleTimeout ?? workflow.taskDefaults?.scheduleTimeout;

  return {
    readableId: ON_FAILURE_TASK_NAME,
    action: onFailureTaskName(workflow),
    timeout: durationToString(
      onFailure.executionTimeout || workflow.taskDefaults?.executionTimeout || '60s'
    ),
    scheduleTimeout: scheduleTimeout ? durationToString(scheduleTimeout) : undefined,
    inputs: '{}',
    parents: [],
    retries: onFailure.retries || workflow.taskDefaults?.retries || 0,
    rateLimits: mapRateLimitPb(onFailure.rateLimits || workflow.taskDefaults?.rateLimits),
    workerLabels: mapWorkerLabelPb(
      onFailure.desiredWorkerLabels || workflow.taskDefaults?.workerLabels
    ),
    concurrency: [],
    backoffFactor: onFailure.backoff?.factor || workflow.taskDefaults?.backoff?.factor,
    backoffMaxSeconds: onFailure.backoff?.maxSeconds || workflow.taskDefaults?.backoff?.maxSeconds,
    isDurable: false,
    slotRequests: mapSlotRequestsPb(onFailure, false),
  };
}

export function mapStickyStrategyPb(
  sticky: WorkflowDefinition['sticky']
): StickyStrategy | undefined {
  // `workflow.sticky` is optional. When omitted, we don't set any sticky strategy.
  //
  // When provided, `workflow.sticky` is a v1 (non-protobuf) config which may also include
  // legacy protobuf enum values for backwards compatibility.
  if (sticky == null) {
    return undefined;
  }

  switch (sticky) {
    case 'soft':
    case 'SOFT':
    case 0:
      return StickyStrategy.SOFT;
    case 'hard':
    case 'HARD':
    case 1:
      return StickyStrategy.HARD;
    default:
      throw new HatchetError(`Invalid sticky strategy: ${sticky}`);
  }
}

/**
 * Builds the registration request for a workflow. Accepts a declaration or its
 * definition; the namespace is applied and the on-success task appended the same way
 * `normalizeWorkflowDefinition` does, so callers may pass either the raw declaration or
 * an already normalized definition.
 *
 * Action ids are `<workflow>:<task>`, lowercased (see `createActionId`).
 */
export function workflowToProto(
  definition: WorkflowDefinition | BaseWorkflowDeclaration<any, any>,
  opts: WorkflowProtoOptions = {}
): CreateWorkflowVersionRequest {
  const workflow = normalizeWorkflowDefinition(definition, opts);
  const { namespace } = opts;
  const { concurrency } = workflow;

  const eventTriggers = [
    ...(workflow.onEvents || []).map((event) => applyNamespace(event, namespace)),
    ...(workflow.on && 'event' in workflow.on && workflow.on.event
      ? Array.isArray(workflow.on.event)
        ? workflow.on.event.map((event) => applyNamespace(event, namespace))
        : [applyNamespace(workflow.on.event, namespace)]
      : []),
  ];
  const cronTriggers: string[] = [
    ...(workflow.onCrons || []),
    ...(workflow.on && 'cron' in workflow.on && workflow.on.cron
      ? Array.isArray(workflow.on.cron)
        ? workflow.on.cron
        : [workflow.on.cron]
      : []),
  ];

  const concurrencyArr = Array.isArray(concurrency) ? concurrency : [];
  const concurrencySolo = !Array.isArray(concurrency) ? concurrency : undefined;

  assertValidConcurrencyArr(concurrencyArr);
  assertValidConcurrencyArr(concurrencySolo ? [concurrencySolo] : undefined);

  // Convert Zod schema to JSON Schema if provided
  let inputJsonSchema: Uint8Array | undefined;
  if (workflow.inputValidator) {
    const jsonSchema = z.toJSONSchema(workflow.inputValidator as any);
    inputJsonSchema = new TextEncoder().encode(JSON.stringify(jsonSchema));
  }

  const durableTaskSet = new Set(workflow._durableTasks);

  return {
    name: workflow.name,
    description: workflow.description || '',
    version: workflow.version || '',
    eventTriggers,
    cronTriggers,
    sticky: mapStickyStrategyPb(workflow.sticky),
    concurrencyArr: mapConcurrencyPb(concurrencyArr),
    onFailureTask: onFailureTaskOf(workflow),
    defaultPriority: workflow.defaultPriority,
    inputJsonSchema,
    tasks: [...workflow._tasks, ...workflow._durableTasks].map<CreateTaskOpts>((task) => ({
      readableId: task.name,
      action: createActionId(workflow.name, task.name),
      timeout: resolveExecutionTimeout(task, workflow.taskDefaults),
      scheduleTimeout: resolveScheduleTimeout(task, workflow.taskDefaults),
      inputs: '{}',
      parents: task.parents?.map((p) => p.name) ?? [],
      userData: '{}',
      // Batch tasks buffer many concurrent runs into a single execution; per-item retry
      // semantics don't apply, so retries is always forced to 0.
      retries: batchOf(task) ? 0 : task.retries || workflow.taskDefaults?.retries || 0,
      rateLimits: mapRateLimitPb(task.rateLimits || workflow.taskDefaults?.rateLimits),
      workerLabels: mapWorkerLabelPb(
        task.desiredWorkerLabels || workflow.taskDefaults?.workerLabels
      ),
      backoffFactor: task.backoff?.factor || workflow.taskDefaults?.backoff?.factor,
      backoffMaxSeconds: task.backoff?.maxSeconds || workflow.taskDefaults?.backoff?.maxSeconds,
      conditions: taskConditionsToPb(task, namespace),
      isDurable: durableTaskSet.has(task),
      slotRequests: mapSlotRequestsPb(task, durableTaskSet.has(task)),
      batch: mapBatchConfigPb(batchOf(task)),
      concurrency: (() => {
        const taskConcurrency = taskConcurrencyArr(task, workflow);
        assertValidConcurrencyArr(taskConcurrency);
        return mapConcurrencyPb(taskConcurrency);
      })(),
    })),
    concurrency: concurrencySolo ? mapConcurrencyPb([concurrencySolo])[0] : undefined,
    defaultFilters:
      workflow.defaultFilters?.map((f) => ({
        scope: f.scope,
        expression: f.expression,
        payload: f.payload ? new TextEncoder().encode(JSON.stringify(f.payload)) : undefined,
      })) ?? [],
    idempotency: workflow.idempotency
      ? {
          expression: workflow.idempotency.expression,
          ttlMs:
            workflow.idempotency.strategy === 'status'
              ? workflow.idempotency.fallbackTtlMs
              : workflow.idempotency.ttlMs,
          method:
            workflow.idempotency.strategy === 'status'
              ? IdempotencyMethod.STATUS
              : IdempotencyMethod.TTL,
        }
      : undefined,
  };
}

export function mapWorkerLabelPb(
  in_: CreateWorkflowTaskOpts<any, any>['desiredWorkerLabels']
): Record<string, DesiredWorkerLabels> {
  if (!in_) {
    return {};
  }

  return Object.entries(in_).reduce<Record<string, DesiredWorkerLabels>>(
    (acc, [key, label]) => {
      if (!label) {
        return {
          ...acc,
          [key]: {
            strValue: undefined,
            intValue: undefined,
          },
        };
      }

      if (typeof label === 'string') {
        return {
          ...acc,
          [key]: {
            strValue: label,
            intValue: undefined,
          },
        };
      }

      if (typeof label === 'number') {
        return {
          ...acc,
          [key]: {
            strValue: undefined,
            intValue: label,
          },
        };
      }

      return {
        ...acc,
        [key]: {
          strValue: typeof label.value === 'string' ? label.value : undefined,
          intValue: typeof label.value === 'number' ? label.value : undefined,
          required: label.required,
          weight: label.weight,
          comparator: label.comparator,
        },
      };
    },
    {} as Record<string, DesiredWorkerLabels>
  );
}

/** The action id of a workflow's on-failure task. */
export function onFailureTaskName(workflow: Pick<WorkflowDefinition, 'name'>) {
  return createActionId(workflow.name, ON_FAILURE_TASK_NAME);
}

export type LeafableTask =
  CreateWorkflowTaskOpts<any, any> | CreateWorkflowDurableTaskOpts<any, any>;

export function getLeaves(tasks: LeafableTask[]): LeafableTask[] {
  return tasks.filter((task) => isLeafTask(task, tasks));
}

export function isLeafTask(task: LeafableTask, allTasks: LeafableTask[]): boolean {
  return !allTasks.some((t) => t.parents?.some((p) => p.name === task.name));
}

/** Durable tasks stay on the durable pool; slotCost applies only to the default pool. */
export function mapSlotRequestsPb(
  task: { slotRequests?: Record<string, number>; slotCost?: number },
  isDurable: boolean
): Record<string, number> {
  if (task.slotRequests) {
    return task.slotRequests;
  }

  if (isDurable) {
    return { durable: 1 };
  }

  if (task.slotCost !== undefined) {
    if (!Number.isInteger(task.slotCost) || task.slotCost <= 0) {
      throw new Error(`slotCost must be a positive integer, got: ${task.slotCost}`);
    }

    return { default: task.slotCost };
  }

  return { default: 1 };
}

export function mapRateLimitPb(
  limits: CreateWorkflowTaskOpts<any, any>['rateLimits']
): CreateStepRateLimit[] {
  if (!limits) {
    return [];
  }

  return limits.map((l) => {
    let key = l.staticKey;
    const keyExpression = l.dynamicKey;

    if (l.key !== undefined) {
      console.warn(
        'key is deprecated and will be removed in a future release, please use staticKey instead'
      );
      ({ key } = l);
    }

    if (keyExpression !== undefined) {
      if (key !== undefined) {
        throw new Error('Cannot have both static key and dynamic key set');
      }
      key = keyExpression;
      if (!validateCelExpression(keyExpression)) {
        throw new Error(`Invalid CEL expression: ${keyExpression}`);
      }
    }

    if (key === undefined) {
      throw new Error(`Invalid key`);
    }

    let units: number | undefined;
    let unitsExpression: string | undefined;
    if (typeof l.units === 'number') {
      ({ units } = l);
    } else {
      if (!validateCelExpression(l.units)) {
        throw new Error(`Invalid CEL expression: ${l.units}`);
      }
      unitsExpression = l.units;
    }

    let limitExpression: string | undefined;
    if (l.limit !== undefined) {
      if (typeof l.limit === 'number') {
        limitExpression = `${l.limit}`;
      } else {
        if (!validateCelExpression(l.limit)) {
          throw new Error(`Invalid CEL expression: ${l.limit}`);
        }

        limitExpression = l.limit;
      }
    }

    if (keyExpression !== undefined && limitExpression === undefined) {
      throw new Error('CEL based keys requires limit to be set');
    }

    if (limitExpression === undefined) {
      limitExpression = `-1`;
    }

    return {
      key,
      keyExpr: keyExpression,
      units,
      unitsExpr: unitsExpression,
      limitValuesExpr: limitExpression,
      duration: l.duration,
    };
  });
}

/** Batch tasks are only available on non-durable tasks; durable tasks never carry `batch`. */
export function batchOf(
  task: CreateWorkflowTaskOpts<any, any> | CreateWorkflowDurableTaskOpts<any, any>
): CreateWorkflowTaskOpts<any, any>['batch'] {
  return 'batch' in task ? task.batch : undefined;
}

// mapConcurrencyPb maps SDK concurrency entries onto the proto shape; entries keep their
// declared order, which is the chain order.
export function mapConcurrencyPb(entries: Concurrency[]) {
  return entries.map((c) => ({
    expression: c.expression,
    // a string maxRuns is a CEL expression; the static field then carries the default
    // of 1, which only governs slots created before the expression existed
    maxRuns: typeof c.maxRuns === 'string' ? 1 : c.maxRuns,
    limitStrategy: c.limitStrategy,
    name: c.name,
    isTenantScoped: c.isTenantScoped,
    maxRunsExpression: typeof c.maxRuns === 'string' ? c.maxRuns : undefined,
  }));
}

export function taskConcurrencyArr(
  task: { concurrency?: Concurrency | Concurrency[] },
  workflow: { taskDefaults?: { concurrency?: Concurrency | Concurrency[] } }
): Concurrency[] {
  if (task.concurrency) {
    return Array.isArray(task.concurrency) ? task.concurrency : [task.concurrency];
  }

  if (workflow.taskDefaults?.concurrency) {
    return Array.isArray(workflow.taskDefaults.concurrency)
      ? workflow.taskDefaults.concurrency
      : [workflow.taskDefaults.concurrency];
  }

  return [];
}

export function assertValidConcurrencyArr(concurrency: Concurrency[] | undefined): void {
  concurrency?.forEach((c) => {
    if (typeof c.maxRuns === 'string') {
      if (!c.maxRuns.trim()) {
        throw new Error('concurrency.maxRuns expression must be non-empty');
      }
      return;
    }

    if (c.maxRuns !== undefined && (!Number.isInteger(c.maxRuns) || c.maxRuns <= 0)) {
      throw new Error(
        `concurrency.maxRuns must be a positive integer or a CEL expression, got: ${c.maxRuns}`
      );
    }
  });
}

export function mapBatchConfigPb(
  batch: CreateWorkflowTaskOpts<any, any>['batch']
): TaskBatchConfig | undefined {
  if (!batch) {
    return undefined;
  }

  if (!Number.isInteger(batch.maxSize) || batch.maxSize <= 0) {
    throw new Error(`batch.maxSize must be a positive integer, got: ${batch.maxSize}`);
  }

  const batchMaxIntervalMs =
    batch.maxInterval !== undefined ? durationToMs(batch.maxInterval) : undefined;

  if (batchMaxIntervalMs !== undefined && batchMaxIntervalMs <= 0) {
    throw new Error('batch.maxInterval must be positive when provided');
  }

  if (
    batch.groupMaxRuns !== undefined &&
    (!Number.isInteger(batch.groupMaxRuns) || batch.groupMaxRuns <= 0)
  ) {
    throw new Error(
      `batch.groupMaxRuns must be a positive integer when provided, got: ${batch.groupMaxRuns}`
    );
  }

  return {
    batchMaxSize: batch.maxSize,
    batchMaxIntervalMs,
    batchGroupKey: batch.groupKey,
    batchGroupMaxRuns: batch.groupMaxRuns,
    broadcastOutput: batch.broadcastOutput,
  };
}

// Helper function to validate CEL expressions

function validateCelExpression(_expr: string): boolean {
  // FIXME: this is a placeholder. In a real implementation, you'd need to use a CEL parser or validator.
  // For now, we'll just return true to mimic the behavior.
  return true;
}

export function resolveExecutionTimeout(
  task: { executionTimeout?: Duration; timeout?: Duration },
  workflowDefaults?: { executionTimeout?: Duration }
): string {
  return durationToString(
    task.executionTimeout || task.timeout || workflowDefaults?.executionTimeout || '60s'
  );
}

export function resolveScheduleTimeout(
  task: { scheduleTimeout?: Duration },
  workflowDefaults?: { scheduleTimeout?: Duration }
): string | undefined {
  const value = task.scheduleTimeout || workflowDefaults?.scheduleTimeout;
  return value ? durationToString(value) : undefined;
}
