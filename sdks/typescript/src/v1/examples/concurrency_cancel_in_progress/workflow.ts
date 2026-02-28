import { ConcurrencyLimitStrategy } from '@hatchet/v1';
import { hatchet } from '../hatchet-client';
import type { EmptyTaskOutput } from '../concurrency-types';

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

export type WorkflowInput = {
  group: string;
};

export type WorkflowOutput = {
  step1: EmptyTaskOutput;
  step2: EmptyTaskOutput;
};

export const concurrencyCancelInProgressWorkflow = hatchet.workflow<WorkflowInput, WorkflowOutput>({
  name: 'ConcurrencyCancelInProgress',
  concurrency: {
    expression: 'input.group',
    maxRuns: 1,
    limitStrategy: ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
  },
});

const step1 = concurrencyCancelInProgressWorkflow.task({
  name: 'step1',
  fn: async (): Promise<EmptyTaskOutput> => {
    for (let i = 0; i < 50; i += 1) {
      await sleep(100);
    }
    return {};
  },
});

concurrencyCancelInProgressWorkflow.task({
  name: 'step2',
  parents: [step1],
  fn: async (): Promise<EmptyTaskOutput> => {
    for (let i = 0; i < 50; i += 1) {
      await sleep(100);
    }
    return {};
  },
});
