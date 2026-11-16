import { ConcurrencyLimitStrategy } from '@hatchet/v1';
import { hatchet } from '../hatchet-client';

export type WorkflowInput = {
  account: string;
  tier: string;
};

export type WorkflowOutput = {
  'dynamic-task': {
    account: string;
  };
};

// > Dynamic Max Runs
// maxRuns accepts a number or a CEL expression string. With an expression, each
// concurrency group's limit is computed from the task's input.
export const concurrencyDynamicWorkflow = hatchet.workflow<WorkflowInput, WorkflowOutput>({
  name: 'concurrency-dynamic',
});

concurrencyDynamicWorkflow.task({
  name: 'dynamic-task',
  concurrency: [
    {
      expression: 'input.account',
      maxRuns: "input.tier == 'premium' ? 10 : 1",
      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    },
  ],
  fn: async (input) => ({ account: input.account }),
});
// !!
