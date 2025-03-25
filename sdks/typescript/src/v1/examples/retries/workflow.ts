/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';

// ❓ Simple Step Retries
export const retries = hatchet.task({
  name: 'retries',
  retries: 3,
  fn: async (_, ctx) => {
    throw new Error('intentional failure');
  },
});
// !!

// ❓ Step Retries with Count
export const retriesWithCount = hatchet.task({
  name: 'retriesWithCount',
  retries: 3,
  fn: async (_, ctx) => {
    // ❓ Get the current retry count
    const retryCount = ctx.retryCount();

    console.log(`Retry count: ${retryCount}`);

    if (retryCount < 2) {
      throw new Error('intentional failure');
    }

    return {
      message: 'success',
    };
  },
});
// !!

// ❓ Step Retries with Backoff
export const withBackoff = hatchet.task({
  name: 'withBackoff',
  retries: 3,
  backoff: {
    factor: 2,
    maxSeconds: 10,
  },
  fn: async () => {
    throw new Error('intentional failure');
  },
});
// !!
