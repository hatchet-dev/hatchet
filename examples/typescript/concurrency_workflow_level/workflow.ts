import { ConcurrencyLimitStrategy } from '@hatchet-dev/typescript-sdk/v1';
import { hatchet } from '../hatchet-client';
import type { EmptyTaskOutput } from '../concurrency-types';

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

export const SLEEP_TIME_MS = 500;
export const DIGIT_MAX_RUNS = 8;
export const NAME_MAX_RUNS = 3;

export type WorkflowInput = {
  name: string;
  digit: string;
};

export type WorkflowOutput = {
  task_1: EmptyTaskOutput;
  task_2: EmptyTaskOutput;
};

export const concurrencyWorkflowLevelWorkflow = hatchet.workflow<WorkflowInput, WorkflowOutput>({
  name: 'ConcurrencyWorkflowLevel',
  concurrency: [
    {
      expression: 'input.digit',
      maxRuns: DIGIT_MAX_RUNS,
      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    },
    {
      expression: 'input.name',
      maxRuns: NAME_MAX_RUNS,
      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    },
  ],
});

concurrencyWorkflowLevelWorkflow.task({
  name: 'task_1',
  fn: async (): Promise<EmptyTaskOutput> => {
    await sleep(SLEEP_TIME_MS);
    return {};
  },
});

concurrencyWorkflowLevelWorkflow.task({
  name: 'task_2',
  fn: async (): Promise<EmptyTaskOutput> => {
    await sleep(SLEEP_TIME_MS);
    return {};
  },
});
