/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';

export const ERROR_TEXT = 'step1 failed';

// > On Failure Task
// This workflow will fail because `step1` throws. We define an `onFailure` handler to run cleanup.
export const failureWorkflow = hatchet.workflow({
  name: 'OnFailureWorkflow',
});

failureWorkflow.task({
  name: 'step1',
  executionTimeout: '1s',
  fn: async () => {
    throw new Error(ERROR_TEXT);
  },
});

// 👀 After the workflow fails, this special step will run
failureWorkflow.onFailure({
  name: 'on_failure',
  fn: async (_input, ctx) => {
    console.log('onFailure for run:', ctx.workflowRunId());
    console.log('upstream errors:', ctx.errors());

    return {
      status: 'success',
    };
  },
});
// !!
