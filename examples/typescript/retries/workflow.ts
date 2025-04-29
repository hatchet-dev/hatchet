
import { hatchet } from '../hatchet-client';

// > Simple Step Retries
export const retries = hatchet.task({
  name: 'retries',
  retries: 3,
  fn: async (_, ctx) => {
    throw new Error('intentional failure');
  },
});


// > Retries with Count
export const retriesWithCount = hatchet.task({
  name: 'retriesWithCount',
  retries: 3,
  fn: async (_, ctx) => {
    // > Get the current retry count
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


// > Retries with Backoff
export const withBackoff = hatchet.task({
  name: 'withBackoff',
  retries: 10,
  backoff: {
    // 👀 Maximum number of seconds to wait between retries
    maxSeconds: 10,
    // 👀 Factor to increase the wait time between retries.
    // This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit
    factor: 2,
  },
  fn: async () => {
    throw new Error('intentional failure');
  },
});

