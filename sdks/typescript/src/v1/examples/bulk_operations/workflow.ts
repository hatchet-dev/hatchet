import { hatchet } from '../hatchet-client';

export const bulkReplayTest1 = hatchet.task({
  name: 'bulk-replay-test-1',
  retries: 1,
  fn: async (_input, ctx) => {
    ctx.logger.info('retrying bulk replay test task', { retryCount: ctx.retryCount() });
    if (ctx.retryCount() === 0) {
      throw new Error('This is a test error to trigger a retry.');
    }
  },
});

export const bulkReplayTest2 = hatchet.task({
  name: 'bulk-replay-test-2',
  retries: 1,
  fn: async (_input, ctx) => {
    ctx.logger.info('retrying bulk replay test task', { retryCount: ctx.retryCount() });
    if (ctx.retryCount() === 0) {
      throw new Error('This is a test error to trigger a retry.');
    }
  },
});

export const bulkReplayTest3 = hatchet.task({
  name: 'bulk-replay-test-3',
  retries: 1,
  fn: async (_input, ctx) => {
    ctx.logger.info('retrying bulk replay test task', { retryCount: ctx.retryCount() });
    if (ctx.retryCount() === 0) {
      throw new Error('This is a test error to trigger a retry.');
    }
  },
});
