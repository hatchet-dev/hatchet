/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';

// ❓ On Success Task
export const onSuccess = hatchet.task({
  name: 'run-on-success',
  // 👀 onSuccess handler will run if the fn task succeeds
  onSuccess: (_, ctx) => {
    console.log('onSuccess for run:', ctx.workflowRunId());
    return {
      'on-success': 'success',
    };
  },
  fn: async () => {
    return {
      'always-succeed': 'success',
    };
  },
});
// !!

// ❓ On Success DAG
export const onSuccessDag = hatchet.workflow({
  name: 'on-success-dag',
  // 👀 onSuccess handler will run if all tasks in the workflow succeed
  onSuccess: (_, ctx) => {
    console.log('onSuccess for run:', ctx.workflowRunId());
    return {
      'on-success': 'success',
    };
  },
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
// !!
