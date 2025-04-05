// ❓ Declaring a Task
import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

export const simple = hatchet.task({
  name: 'simple',
  timeout: '1h',
  fn: async (input: SimpleInput) => {
    await sleep(100000);
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

// !!

// see ./worker.ts and ./run.ts for how to run the workflow
