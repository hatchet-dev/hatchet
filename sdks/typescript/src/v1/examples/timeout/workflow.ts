import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';

export type SimpleInput = {
  Message: string;
};

// > Execution Timeout
// Mirrors Python `examples/timeout/test_timeout.py::test_execution_timeout`
export const timeoutTask = hatchet.task({
  name: 'timeout',
  executionTimeout: '3s',
  fn: async (_: SimpleInput, ctx) => {
    try {
      await sleep(10 * 1000, ctx.abortController.signal);
    } catch {
      throw new Error('Task was cancelled');
    }
    return {
      status: 'success',
    };
  },
});
// !!

// > Refresh Timeout
// Mirrors Python `examples/timeout/test_timeout.py::test_run_refresh_timeout`
export const refreshTimeoutTask = hatchet.task({
  name: 'refresh-timeout',
  executionTimeout: '3s',
  scheduleTimeout: '5s',
  fn: async (input: SimpleInput, ctx) => {
    ctx.refreshTimeout('5s');
    try {
      await sleep(4000, ctx.abortController.signal);
    } catch {
      throw new Error('cancelled');
    }
    return {
      status: 'success',
      message: input.Message.toLowerCase(),
    };
  },
});
// !!
