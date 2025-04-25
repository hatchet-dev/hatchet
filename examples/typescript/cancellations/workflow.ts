import sleep from '@hatchet-dev/typescript-sdk/util/sleep';
import axios from 'axios';
import { hatchet } from '../hatchet-client';

// ❓ Declaring a Task
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

// ❓ Abort Signal
export const abortSignal = hatchet.task({
  name: 'abort-signal',
  fn: async (_, { controller }) => {
    try {
      const response = await axios.get('https://api.example.com/data', {
        signal: controller.signal,
      });
      // Handle the response
    } catch (error) {
      if (axios.isCancel(error)) {
        // Request was canceled
        console.log('Request canceled');
      } else {
        // Handle other errors
      }
    }
  },
});

// see ./worker.ts and ./run.ts for how to run the workflow
