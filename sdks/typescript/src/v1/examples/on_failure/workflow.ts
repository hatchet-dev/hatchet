/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';

// ❓ On Failure Task
export const alwaysFail = hatchet.task({
  name: 'always-fail',
  // the onFailure function is called when the task fails
  // note: not guaranteed to be called on the same worker
  onFailure: (input, ctx) => {
    console.log('onFailure for run:', ctx.workflowRunId());
    return {
      'on-failure': 'success',
    };
  },
  fn: async () => {
    throw new Error('intentional failure');
  },
});
// !!
