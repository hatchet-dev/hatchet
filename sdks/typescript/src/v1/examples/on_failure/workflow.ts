/* eslint-disable no-console */
import { hatchet } from '../client';

export const alwaysFail = hatchet.workflow({
  name: 'always-fail',
  onFailure: (ctx) => {
    console.log('onFailure for run:', ctx.workflowRunId());
    return {
      'on-failure': 'success',
    };
  },
});

alwaysFail.task({
  name: 'always-fail',
  fn: () => {
    throw new Error('intentional failure');
  },
});
