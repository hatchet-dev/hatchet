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

// > Cancel Queued Except Newest
export const concurrencyCancelQueuedExceptNewestWorkflow = hatchet.workflow<
  WorkflowInput,
  WorkflowOutput
>({
  name: 'concurrencycancelqueuedexceptnewest',
  concurrency: {
    expression: 'input.group',
    maxRuns: 1,
    limitStrategy: ConcurrencyLimitStrategy.CANCEL_QUEUED_EXCEPT_NEWEST,
  },
});
// !!

const step1 = concurrencyCancelQueuedExceptNewestWorkflow.task({
  name: 'step1',
  fn: async (_, ctx): Promise<EmptyTaskOutput> => {
    for (let i = 0; i < 50; i += 1) {
      await sleep(20, ctx.abortController.signal);
    }
    return {};
  },
});

concurrencyCancelQueuedExceptNewestWorkflow.task({
  name: 'step2',
  parents: [step1],
  fn: async (_, ctx): Promise<EmptyTaskOutput> => {
    for (let i = 0; i < 50; i += 1) {
      await sleep(20, ctx.abortController.signal);
    }
    return {};
  },
});
