import { RetryAfterError } from '@hatchet-dev/typescript-sdk/v1/task';
import { hatchet } from '../hatchet-client';

export type UpstreamInput = {
  failingAttempts: number;
};

// > Retry after an upstream-provided delay
export const retryAfterUpstreamDelay = hatchet.task({
  name: 'retry-after-upstream-delay',
  retries: 5,
  fn: async (input: UpstreamInput, ctx) => {
    if (ctx.retryCount() < input.failingAttempts) {
      throw new RetryAfterError('upstream rate limited', { after: '2s' });
    }

    return { attempt: ctx.retryCount() };
  },
});

// > Exponential backoff with full jitter
export const retryAfterExponentialBackoff = hatchet.task({
  name: 'retry-after-exponential-backoff',
  retries: 3,
  fn: async (input: UpstreamInput, ctx) => {
    if (ctx.retryCount() < input.failingAttempts) {
      throw new RetryAfterError('upstream unavailable', {
        after: Math.random() * Math.min(2 ** ctx.retryCount(), 60) * 1_000,
      });
    }

    return { attempt: ctx.retryCount() };
  },
});
