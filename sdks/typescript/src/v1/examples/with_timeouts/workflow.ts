// ❓ Declaring a Task
import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

// ❓ Execution Timeout
export const withTimeouts = hatchet.task({
  name: 'with-timeouts',
  // time the task can wait in the queue before it is cancelled
  scheduleTimeout: '10s',
  // time the task can run before it is cancelled
  executionTimeout: '10s',
  fn: async (input: SimpleInput, ctx) => {
    // wait 15 seconds
    await sleep(15000);

    // get the abort controller
    const { abortController } = ctx;

    // if the abort controller is aborted, throw an error
    if (abortController.signal.aborted) {
      throw new Error('cancelled');
    }

    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
// !!

// ❓ Refresh Timeout
export const refreshTimeout = hatchet.task({
  name: 'refresh-timeout',
  executionTimeout: '10s',
  scheduleTimeout: '10s',
  fn: async (input: SimpleInput, ctx) => {
    // adds 15 seconds to the execution timeout
    ctx.refreshTimeout('15s');
    await sleep(15000);

    // get the abort controller
    const { abortController } = ctx;

    // now this condition will not be met
    // if the abort controller is aborted, throw an error
    if (abortController.signal.aborted) {
      throw new Error('cancelled');
    }

    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
// !!
