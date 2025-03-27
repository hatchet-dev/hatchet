/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';

// ❓ On Failure Task
export const failureWorkflow = hatchet.workflow({
  name: 'always-fail',
});

failureWorkflow.task({
  name: 'always-fail',
  fn: async () => {
    throw new Error('intentional failure');
  },
});

failureWorkflow.onFailure({
  name: 'on-failure',
  fn: async (input, ctx) => {
    console.log('onFailure for run:', ctx.workflowRunId());
    return {
      'on-failure': 'success',
    };
  },
});

// !!
