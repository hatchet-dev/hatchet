// ❓ Declaring a Task
import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export const cancellation = hatchet.task({
  name: 'cancellation',
  fn: async (_, { cancelled }) => {
    await sleep(10 * 1000);

    if (cancelled) {
      throw new Error('Task was cancelled');
    }

    return {
      Completed: true,
    };
  },
});
// !!

// see ./worker.ts and ./run.ts for how to run the workflow
