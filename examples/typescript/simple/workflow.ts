// ❓ Declaring a Task
import sleep from '@hatchet-dev/typescript-sdk/util/sleep';
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow

export type SimpleInput = {
  Message: string;
};

export const simple = hatchet.task({
  name: 'simple',
  retries: 3,
  fn: async (input: SimpleInput, ctx) => {
    ctx.log('hello from the workflow');
    await sleep(100);
    ctx.log('goodbye from the workflow');
    await sleep(100);
    if (ctx.retryCount() < 2) {
      throw new Error('test error');
    }
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

// see ./worker.ts and ./run.ts for how to run the workflow
