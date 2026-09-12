/**
 * Turns the user's declarations into what the handler needs: one registration request per
 * workflow (the SDK's `CreateWorkflowVersionRequest`), the task functions keyed by action
 * id, and the subset of action ids this endpoint serves.
 */
import {
  createActionId,
  normalizeWorkflowDefinition,
  onFailureTaskName,
  workflowToProto,
} from '@hatchet-dev/typescript-sdk/edge/index.js';
import type {
  BaseWorkflowDeclaration,
  Context,
  CreateWorkflowVersionRequest,
  DurableContext,
  WorkflowDefinition,
} from '@hatchet-dev/typescript-sdk/edge/index.js';

export type TaskRunner = (ctx: Context<any, any>) => unknown;
export type DurableTaskRunner = (ctx: DurableContext<any, any>) => unknown;

export interface RegisteredWorkflow {
  /** The workflow name as registered, lowercased and without a namespace. */
  name: string;
  declaration: BaseWorkflowDeclaration<any, any>;
  /** The normalized definition the registration was built from, unsupported options removed. */
  definition: WorkflowDefinition;
  /** The SDK's registration request for the workflow. */
  proto: CreateWorkflowVersionRequest;
  /** Every action id the workflow derives, on-failure and on-success tasks included. */
  actions: string[];
}

export interface Registry {
  workflows: RegisteredWorkflow[];
  /** Task functions by action id, for every non-durable task. */
  runners: Map<string, TaskRunner>;
  /** Action ids of durable tasks; served over the relay, never over a POST. */
  durableActions: Set<string>;
  /** Task functions by action id, for every durable task. */
  durableRunners: Map<string, DurableTaskRunner>;
  /** Action ids this endpoint serves. */
  served: Set<string>;
  hasWorkflow(workflowName: string): boolean;
}

export type ServeEntry = BaseWorkflowDeclaration<any, any> | string;

/**
 * Declaration options that need a worker. They are removed from the registration and
 * warned about once each; the operator owns routing, slots and eviction.
 */
const IGNORED_TASK_OPTIONS: Record<string, string> = {
  slotCost: 'a serverless endpoint has no worker slots',
  slotRequests: 'a serverless endpoint has no worker slots',
  batch: 'batch tasks are not supported; the task is registered as a plain task',
  desiredWorkerLabels: 'worker labels do not apply; the operator routes by action id',
  evictionPolicy: "the endpoint's inline wait budget decides when a durable task evicts",
};

const IGNORED_WORKFLOW_OPTIONS: Record<string, string> = {
  sticky: 'sticky assignment needs a worker; the operator routes by action id',
};

type Warn = (message: string) => void;

export function buildRegistry(
  workflows: BaseWorkflowDeclaration<any, any>[],
  serve?: ServeEntry[],
  warn: Warn = (message) => console.warn(message)
): Registry {
  const ignored = new Map<string, string[]>();
  const noteIgnored = (option: string, where: string) => {
    const list = ignored.get(option) ?? [];
    list.push(where);
    ignored.set(option, list);
  };

  const registered: RegisteredWorkflow[] = [];
  const runners = new Map<string, TaskRunner>();
  const durableActions = new Set<string>();
  const durableRunners = new Map<string, DurableTaskRunner>();
  const seenNames = new Set<string>();

  for (const declaration of workflows) {
    const definition = normalizeWorkflowDefinition(sanitize(declaration.definition, noteIgnored));
    const proto = workflowToProto(definition);
    const { name } = definition;

    if (seenNames.has(name)) {
      throw new Error(`workflow "${name}" is declared twice`);
    }

    seenNames.add(name);

    const actions: string[] = [];

    for (const task of definition._tasks) {
      const actionId = createActionId(name, task.name);
      actions.push(actionId);

      if (task.fn) {
        const { fn } = task;
        runners.set(actionId, (ctx) => fn(ctx.input, ctx as any));
      }
    }

    for (const task of definition._durableTasks) {
      const actionId = createActionId(name, task.name);
      actions.push(actionId);
      durableActions.add(actionId);

      if (task.fn) {
        const { fn } = task;
        durableRunners.set(actionId, (ctx) => fn(ctx.input, ctx));
      }
    }

    const onFailureFn = definition.onFailure
      ? typeof definition.onFailure === 'function'
        ? definition.onFailure
        : definition.onFailure.fn
      : undefined;

    if (onFailureFn) {
      const actionId = onFailureTaskName(definition);
      actions.push(actionId);
      runners.set(actionId, (ctx) => onFailureFn(ctx.input, ctx as any));
    }

    registered.push({ name, declaration, definition, proto, actions });
  }

  for (const [option, where] of ignored) {
    const reason = IGNORED_TASK_OPTIONS[option] ?? IGNORED_WORKFLOW_OPTIONS[option];
    warn(`@hatchet-dev/serverless: ignoring "${option}" on ${where.join(', ')}: ${reason}`);
  }

  return {
    workflows: registered,
    runners,
    durableActions,
    durableRunners,
    served: servedActions(registered, serve),
    hasWorkflow: (workflowName) => seenNames.has(workflowName.toLowerCase()),
  };
}

function servedActions(workflows: RegisteredWorkflow[], serve?: ServeEntry[]): Set<string> {
  if (!serve) {
    return new Set(workflows.flatMap((workflow) => workflow.actions));
  }

  const served = new Set<string>();

  for (const entry of serve) {
    if (typeof entry === 'string') {
      served.add(entry.toLowerCase());
      continue;
    }

    const workflow = workflows.find((candidate) => candidate.declaration === entry);

    if (!workflow) {
      throw new Error(`serve lists workflow "${entry.definition.name}", which is not in workflows`);
    }

    for (const actionId of workflow.actions) {
      served.add(actionId);
    }
  }

  return served;
}

/**
 * Returns a copy of the definition with the options that need a worker removed, reporting
 * each removal. The declaration itself is never mutated: the same declarations may also be
 * registered by a worker elsewhere.
 */
function sanitize(
  definition: WorkflowDefinition,
  noteIgnored: (option: string, where: string) => void
): WorkflowDefinition {
  const copy: WorkflowDefinition = { ...definition };
  const workflowName = definition.name;

  for (const option of Object.keys(IGNORED_WORKFLOW_OPTIONS)) {
    if ((copy as Record<string, unknown>)[option] !== undefined) {
      noteIgnored(option, workflowName);
      delete (copy as Record<string, unknown>)[option];
    }
  }

  if (copy.taskDefaults?.workerLabels !== undefined) {
    noteIgnored('desiredWorkerLabels', `${workflowName} (taskDefaults.workerLabels)`);
    const { workerLabels: _workerLabels, ...taskDefaults } = copy.taskDefaults;
    copy.taskDefaults = taskDefaults;
  }

  const stripTask = <T extends { name: string }>(task: T): T => {
    const stripped = { ...task } as Record<string, unknown>;

    for (const option of Object.keys(IGNORED_TASK_OPTIONS)) {
      if (stripped[option] !== undefined) {
        noteIgnored(option, `${workflowName}:${task.name}`);
        delete stripped[option];
      }
    }

    return stripped as T;
  };

  copy._tasks = definition._tasks.map(stripTask);
  copy._durableTasks = definition._durableTasks.map(stripTask);

  if (copy.onFailure && typeof copy.onFailure !== 'function') {
    copy.onFailure = stripTask({ ...copy.onFailure, name: 'on-failure' });
    delete (copy.onFailure as Record<string, unknown>).name;
  }

  return copy;
}
