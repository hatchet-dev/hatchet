
import { hatchet } from '../hatchet-client';

// ❓ On Success DAG
export const onSuccessDag = hatchet.workflow({
  name: 'on-success-dag',
});

onSuccessDag.task({
  name: 'always-succeed',
  fn: async () => {
    return {
      'always-succeed': 'success',
    };
  },
});
onSuccessDag.task({
  name: 'always-succeed2',
  fn: async () => {
    return {
      'always-succeed': 'success',
    };
  },
});

// 👀 onSuccess handler will run if all tasks in the workflow succeed
onSuccessDag.onSuccess({
  fn: (_, ctx) => {
    console.log('onSuccess for run:', ctx.workflowRunId());
    return {
      'on-success': 'success',
    };
  },
});
