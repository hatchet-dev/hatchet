import sleep from '@hatchet/util/sleep';
import { ConcurrencyLimitStrategy } from '@hatchet/v1';
import { hatchet } from '../hatchet-client';
import type { EmptyTaskOutput } from '../concurrency-types';

export type WorkflowInput = {
  group: string;
};

export type WorkflowOutput = {
  step1: EmptyTaskOutput;
  step2: EmptyTaskOutput;
};

export const concurrencyCancelNewestWorkflow = hatchet.workflow<WorkflowInput, WorkflowOutput>({
  name: 'concurrency-cancel-newest',
  concurrency: {
    expression: 'input.group',
    maxRuns: 1,
    limitStrategy: ConcurrencyLimitStrategy.CANCEL_NEWEST,
  },
});

const step1 = concurrencyCancelNewestWorkflow.task({
  name: 'step1',
  fn: async (_, ctx): Promise<EmptyTaskOutput> => {
    for (let i = 0; i < 50; i += 1) {
      await sleep(20, ctx.abortController.signal);
    }
    return {};
  },
});

concurrencyCancelNewestWorkflow.task({
  name: 'step2',
  parents: [step1],
  fn: async (_, ctx): Promise<EmptyTaskOutput> => {
    for (let i = 0; i < 50; i += 1) {
      await sleep(20, ctx.abortController.signal);
    }
    return {};
  },
});
