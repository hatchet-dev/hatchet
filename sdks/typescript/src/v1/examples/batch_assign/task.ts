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
  batchMaxSize: 200,
  batchMaxInterval: '1s',
  // group inputs by a computed key (e.g. tenant id or message)
  batchGroupKey: 'input.batchId',
  // allow at most two concurrent batches per key
  batchGroupMaxRuns: 1,
  scheduleTimeout: '5m',
  fn: async (inputs: SimpleInput[], ctx) =>
    Promise.all(
      inputs.map(async (input, index) => {
        // sleep for a random amount of time between 0 and 10 seconds
        const sleepTime = Math.floor(Math.random() * 10000);
        console.log(`sleeping for ${sleepTime}ms`);
        await sleep(sleepTime);

        if (ctx.some((c) => c.cancelled)) {
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
