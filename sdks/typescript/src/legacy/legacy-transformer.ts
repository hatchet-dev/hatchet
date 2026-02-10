/* eslint-disable no-console */
import type { Workflow } from '@hatchet/legacy/workflow';
import { V0Context } from '@hatchet/legacy/step';
import type { CreateStep } from '@hatchet/legacy/step';
import { BaseWorkflowDeclaration, WorkflowDeclaration } from '../v1/declaration';
import type { CreateWorkflowOpts } from '../v1/declaration';
import type { CreateWorkflowTaskOpts, Concurrency } from '../v1/task';
import type { Duration } from '../v1/client/duration';

export type { Workflow as LegacyWorkflow } from '@hatchet/legacy/workflow';

const LEGACY_WORKFLOW_WARNING = [
  '',
  '\x1b[33m╔══════════════════════════════════════════════════════════════════════════════╗',
  '║                                                                              ║',
  '║   ⚠  DEPRECATION WARNING: Legacy workflow format detected.                   ║',
  '║                                                                              ║',
  '║   Please migrate to the v1 SDK:                                              ║',
  '║   https://docs.hatchet.run/home/migration-guide-typescript                   ║',
  '║                                                                              ║',
  '╚══════════════════════════════════════════════════════════════════════════════╝\x1b[0m',
  '',
].join('\n');

/**
 * Type guard: returns true if the workflow is a legacy v0 Workflow (not a BaseWorkflowDeclaration).
 */
export function isLegacyWorkflow(workflow: unknown): workflow is Workflow {
  return (
    workflow != null &&
    typeof workflow === 'object' &&
    !(workflow instanceof BaseWorkflowDeclaration) &&
    'id' in workflow &&
    'steps' in workflow &&
    Array.isArray((workflow as any).steps)
  );
}

/**
 * Emits a deprecation warning for legacy workflow usage.
 */
export function warnLegacyWorkflow(): void {
  console.warn(LEGACY_WORKFLOW_WARNING);
}

/**
 * Transforms a legacy v0 Workflow into a v1 WorkflowDeclaration.
 *
 * The transformed declaration can be registered with a worker and executed
 * by the v1 runtime. Legacy step `run` functions are wrapped to receive
 * a V0Context for backwards compatibility.
 */
export function transformLegacyWorkflow(workflow: Workflow): WorkflowDeclaration<any, any> {
  // Map concurrency
  let concurrency: Concurrency | undefined;
  if (workflow.concurrency) {
    if (workflow.concurrency.key) {
      console.warn(
        '[hatchet] Legacy concurrency key functions are not supported in v1. ' +
          'Use CEL expressions instead: https://docs.hatchet.run/home/v1-sdk-improvements'
      );
    }

    concurrency = {
      expression: workflow.concurrency.expression || workflow.concurrency.name,
      maxRuns: workflow.concurrency.maxRuns,
      limitStrategy: workflow.concurrency.limitStrategy as any,
    };
  }

  const opts: CreateWorkflowOpts = {
    name: workflow.id,
    description: workflow.description,
    version: workflow.version,
    sticky: workflow.sticky != null ? (workflow.sticky as any) : undefined,
    on: workflow.on
      ? {
          cron: workflow.on.cron,
          event: workflow.on.event,
        }
      : undefined,
    concurrency,
    taskDefaults: {
      executionTimeout: workflow.timeout as Duration | undefined,
      scheduleTimeout: workflow.scheduleTimeout as Duration | undefined,
    },
  };

  const declaration = new WorkflowDeclaration(opts);

  // Build task lookup for parent resolution (parents are strings in legacy, objects in v1)
  const taskMap: Record<string, CreateWorkflowTaskOpts<any, any>> = {};

  for (const step of workflow.steps) {
    const taskOpts = legacyStepToTaskOpts(step, taskMap, workflow.timeout);
    taskMap[step.name] = taskOpts;
    // eslint-disable-next-line no-underscore-dangle
    declaration.definition._tasks.push(taskOpts);
  }

  // Handle onFailure
  if (workflow.onFailure) {
    declaration.definition.onFailure = {
      fn: wrapLegacyStepRun(workflow.onFailure),
      executionTimeout: (workflow.onFailure.timeout || workflow.timeout) as Duration | undefined,
      retries: workflow.onFailure.retries,
      rateLimits: mapLegacyRateLimits(workflow.onFailure.rate_limits),
      desiredWorkerLabels: workflow.onFailure.worker_labels as any,
      backoff: workflow.onFailure.backoff,
    };
  }

  return declaration;
}

/**
 * Converts a legacy CreateStep to a v1 CreateWorkflowTaskOpts.
 */
function legacyStepToTaskOpts(
  step: CreateStep<any, any>,
  taskMap: Record<string, CreateWorkflowTaskOpts<any, any>>,
  workflowTimeout?: string
): CreateWorkflowTaskOpts<any, any> {
  return {
    name: step.name,
    fn: wrapLegacyStepRun(step),
    executionTimeout: (step.timeout || workflowTimeout) as Duration | undefined,
    retries: step.retries,
    rateLimits: mapLegacyRateLimits(step.rate_limits),
    desiredWorkerLabels: step.worker_labels as any,
    backoff: step.backoff,
    parents: step.parents?.map((name) => taskMap[name]).filter(Boolean),
  };
}

/**
 * Wraps a legacy step's `run(ctx: V0Context)` function into a v1-compatible
 * `fn(input, ctx: Context)` function by constructing a V0Context at runtime.
 */
function wrapLegacyStepRun(step: CreateStep<any, any>) {
  return (input: any, ctx: any) => {
    // Access the V1Worker from the ContextWorker's private field.
    // This is intentionally accessing a private field for legacy compatibility.
    // eslint-disable-next-line no-underscore-dangle
    const v1Worker = (ctx.worker as any).worker;
    // eslint-disable-next-line no-underscore-dangle
    const v0ctx = new V0Context(ctx.action, ctx.v1._v0, v1Worker);
    // Share the abort controller so cancellation propagates
    v0ctx.controller = ctx.controller;
    return step.run(v0ctx);
  };
}

/**
 * Maps legacy rate limits to v1 format.
 */
function mapLegacyRateLimits(
  limits?: CreateStep<any, any>['rate_limits']
): CreateWorkflowTaskOpts<any, any>['rateLimits'] {
  if (!limits) return undefined;
  return limits.map((l) => ({
    staticKey: l.staticKey || l.key,
    dynamicKey: l.dynamicKey,
    units: l.units,
    limit: l.limit,
    duration: l.duration,
  }));
}

/**
 * Normalizes a workflow: if it is a legacy Workflow, emits a deprecation warning
 * and transforms it into a BaseWorkflowDeclaration. If already a v1 declaration,
 * returns it as-is.
 */
export function normalizeWorkflow(
  workflow: BaseWorkflowDeclaration<any, any> | Workflow
): BaseWorkflowDeclaration<any, any> {
  if (isLegacyWorkflow(workflow)) {
    warnLegacyWorkflow();
    return transformLegacyWorkflow(workflow);
  }
  return workflow;
}

/**
 * Normalizes an array of workflows, transforming any legacy workflows and
 * emitting deprecation warnings.
 */
export function normalizeWorkflows(
  workflows: Array<BaseWorkflowDeclaration<any, any> | Workflow>
): Array<BaseWorkflowDeclaration<any, any>> {
  return workflows.map(normalizeWorkflow);
}

/**
 * Extracts the workflow name from a workflow-like value.
 * Works for strings, BaseWorkflowDeclaration, WorkflowDefinition, and legacy Workflow.
 * Emits a deprecation warning if a legacy workflow is detected.
 */
export function getWorkflowName(
  workflow: string | BaseWorkflowDeclaration<any, any> | Workflow | { name: string }
): string {
  if (typeof workflow === 'string') {
    return workflow;
  }
  if (workflow instanceof BaseWorkflowDeclaration) {
    return workflow.name;
  }
  if (isLegacyWorkflow(workflow)) {
    warnLegacyWorkflow();
    return workflow.id;
  }
  if ('name' in workflow) {
    return workflow.name as string;
  }
  throw new Error(
    'Invalid workflow: must be a string, BaseWorkflowDeclaration, or legacy Workflow object'
  );
}
