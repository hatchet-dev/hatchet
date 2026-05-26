import { hatchet } from '../hatchet-client';

export const ERROR_TEXT = 'step1 failed';

// > On Failure Task
// This workflow will fail because `step1` throws. We define an `onFailure` handler to run cleanup.
export const failureWorkflow = hatchet.workflow({
  name: 'on-failure-workflow',
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
    ctx.logger.info(`onFailure for run: ${ctx.workflowRunId()}`);
    ctx.logger.info('upstream errors', { errors: ctx.errors() });

    return {
      status: 'success',
    };
  },
});
