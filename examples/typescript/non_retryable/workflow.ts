import { NonRetryableError } from '@hatchet-dev/typescript-sdk/v1/task';
import { hatchet } from '../hatchet-client';

export const nonRetryableWorkflow = hatchet.workflow({
  name: 'no-retry-workflow',
});

// > Non-retrying task
const shouldNotRetry = nonRetryableWorkflow.task({
  name: 'should-not-retry',
  fn: () => {
    throw new NonRetryableError('This task should not retry');
  },
  retries: 1,
});

// Create a task that should retry
const shouldRetryWrongErrorType = nonRetryableWorkflow.task({
  name: 'should-retry-wrong-error-type',
  fn: () => {
    throw new Error('This task should not retry');
  },
  retries: 1,
});

const shouldNotRetrySuccessfulTask = nonRetryableWorkflow.task({
  name: 'should-not-retry-successful-task',
  fn: () => {},
});
