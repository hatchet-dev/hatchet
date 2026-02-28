import { ConcurrencyLimitStrategy } from '@hatchet-dev/typescript-sdk/v1';
import { hatchet } from '../hatchet-client';
import type { EmptyTaskOutput } from '../concurrency-types';

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

export const SLEEP_TIME_MS = 2000;
export const DIGIT_MAX_RUNS = 8;
export const NAME_MAX_RUNS = 3;

export type WorkflowInput = {
  name: string;
  digit: string;
};

export type WorkflowOutput = {
  concurrency_task: EmptyTaskOutput;
};

export const concurrencyMultipleKeysWorkflow = hatchet.workflow<WorkflowInput, WorkflowOutput>({
  name: 'ConcurrencyWorkflowManyKeys',
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

concurrencyMultipleKeysWorkflow.task({
  name: 'concurrency_task',
  fn: async (): Promise<EmptyTaskOutput> => {
    await sleep(SLEEP_TIME_MS);
    return {};
  },
});
