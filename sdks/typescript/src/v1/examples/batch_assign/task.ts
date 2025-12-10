// > Declaring a Task
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
  batchId: string;
};

const sleep = (ms: number) =>
  new Promise<void>((resolve) => {
    setTimeout(resolve, ms);
  });

export const batch = hatchet.batchTask({
  name: 'simple',
  // retries are on the individual invocation, not the batch
  retries: 0,
  executionTimeout: '60s',
  batchSize: 100,
  flushInterval: 5000,
  // group inputs by a computed key (e.g. tenant id or message)
  batchKey: 'input.batchId',
  // allow at most two concurrent batches per key
  maxRuns: 1,
  scheduleTimeout: '5m',
  fn: async (inputs: SimpleInput[], ctx) =>
    Promise.all(
      inputs.map(async (input, index) => {
        await sleep(10000);

        if (ctx.some((ctx) => ctx.cancelled)) {
          throw new Error('cancelled');
        }
        console.log(`${input.Message.toLowerCase()}index:${index}`);
        return {
          TransformedMessage: `${input.Message.toLowerCase()}index:${index}`,
        };
      })
    ),
});

// !!

// see ./worker.ts and ./run.ts for how to run the workflow
