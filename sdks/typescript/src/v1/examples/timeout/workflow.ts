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
  fn: async (_: SimpleInput, { cancelled }) => {
    await sleep(10 * 1000);

    if (cancelled) {
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
  executionTimeout: '10s',
  scheduleTimeout: '10s',
  fn: async (input: SimpleInput, ctx) => {
    ctx.refreshTimeout('15s');
    await sleep(15000);

    if (ctx.abortController.signal.aborted) {
      throw new Error('cancelled');
    }

    return {
      status: 'success',
      message: input.Message.toLowerCase(),
    };
  },
});
// !!

