import { ConcurrencyLimitStrategy, SharedConcurrency } from '@hatchet-dev/typescript-sdk/v1';
import { hatchet } from '../hatchet-client';

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

export const SLEEP_TIME_MS = 1500;

export type WorkflowInput = {
  group: string;
  inline?: string;
};

export type RunWindow = {
  startMs: number;
  endMs: number;
};

export type WorkflowOutput = {
  'shared-task': RunWindow;
};

// > Shared Concurrency Strategy
// A tenant-scoped strategy is shared across workflows: every task declaring the same name
// consumes the same concurrency limit. The definition rides on workflow registration and
// re-registering the name updates it in place.
export const sharedLimit: SharedConcurrency = {
  name: 'ts-example-shared-limit',
  tenantScoped: true,
  expression: 'input.group',
  maxRuns: 1,
  limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
};

const runWindowTask = async (): Promise<RunWindow> => {
  const startMs = Date.now();
  await sleep(SLEEP_TIME_MS);
  return { startMs, endMs: Date.now() };
};

export const concurrencySharedWorkflowA = hatchet.workflow<WorkflowInput, WorkflowOutput>({
  name: 'concurrency-shared-a',
});

concurrencySharedWorkflowA.task({
  name: 'shared-task',
  concurrency: [sharedLimit],
  fn: runWindowTask,
});

export const concurrencySharedWorkflowB = hatchet.workflow<WorkflowInput, WorkflowOutput>({
  name: 'concurrency-shared-b',
});

concurrencySharedWorkflowB.task({
  name: 'shared-task',
  concurrency: [sharedLimit],
  fn: runWindowTask,
});

// > Mixed Inline And Shared Concurrency
// A single task can combine a workflow-scoped inline strategy with a shared strategy;
// both limits apply at once.
export const concurrencySharedMixedWorkflow = hatchet.workflow<WorkflowInput, WorkflowOutput>({
  name: 'concurrency-shared-mixed',
});

concurrencySharedMixedWorkflow.task({
  name: 'shared-task',
  concurrency: [
    {
      expression: 'input.inline',
      maxRuns: 1,
      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    },
    sharedLimit,
  ],
  fn: runWindowTask,
});
