/* eslint-disable no-console */
import { hatchet } from '../client';

export const alwaysFail = hatchet.workflow({
  name: 'always-fail',
  onFailure: (input, ctx) => {
    console.log('onFailure for run:', ctx.workflowRunId());
    return {
      'on-failure': 'success',
    };
  },
});

alwaysFail.task({
  name: 'always-fail',
  fn: async () => {
    throw new Error('intentional failure');
  },
});
