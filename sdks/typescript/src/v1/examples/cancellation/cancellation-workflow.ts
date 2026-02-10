import axios from 'axios';
import { hatchet } from '../hatchet-client';

function sleepWithAbort(signal: AbortSignal, ms: number) {
  return new Promise<void>((resolve, reject) => {
    const timer = setTimeout(() => {
      signal.removeEventListener('abort', onAbort);
      resolve();
    }, ms);

    const onAbort = () => {
      clearTimeout(timer);
      reject(new Error('Cancelled'));
    };

    if (signal.aborted) {
      clearTimeout(timer);
      reject(new Error('Cancelled'));
      return;
    }

    signal.addEventListener('abort', onAbort, { once: true });
  });
}

// > Self-cancelling workflow (mirrors Python example)
export const cancellationWorkflow = hatchet.workflow({
  name: 'CancelWorkflow',
});

cancellationWorkflow.task({
  name: 'self-cancel',
  fn: async (_, ctx) => {
    await sleepWithAbort(ctx.abortController.signal, 2000);

    // Cancel the current task run (server-side) and optimistically abort local execution.
    await ctx.cancel();

    // If cancellation didn't stop execution yet, keep waiting but cooperatively.
    await sleepWithAbort(ctx.abortController.signal, 10_000);

    return { error: 'Task should have been cancelled' };
  },
});

cancellationWorkflow.task({
  name: 'check-flag',
  fn: async (_, ctx) => {
    for (let i = 0; i < 3; i += 1) {
      await sleepWithAbort(ctx.abortController.signal, 1000);
      if (ctx.cancelled) {
        throw new Error('Cancelled');
      }
    }
    return { error: 'Task should have been cancelled' };
  },
});
// !!

// > Abort Signal
export const abortSignal = hatchet.task({
  name: 'abort-signal',
  fn: async (_, { abortController }) => {
    try {
      const response = await axios.get('https://api.example.com/data', {
        signal: abortController.signal,
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
// !!

// see ./worker.ts and ./run.ts for how to run the workflow
