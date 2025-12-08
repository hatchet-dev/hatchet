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
  retries: 3,
  executionTimeout: '12s',
  batchSize: 2,
  flushInterval: 1000,
  // group inputs by a computed key (e.g. tenant id or message)
  batchKey: 'input.batchId',
  // allow at most two concurrent batches per key
  maxRuns: 2,
  fn: async (inputs: SimpleInput[]) =>
    Promise.all(
      inputs.map(async (input, index) => {
        await sleep(10000);
        console.log(`${input.Message.toLowerCase()}index:${index}`);
        return {
          TransformedMessage: `${input.Message.toLowerCase()}index:${index}`,
        };
      })
    ),
});

// !!

// see ./worker.ts and ./run.ts for how to run the workflow
