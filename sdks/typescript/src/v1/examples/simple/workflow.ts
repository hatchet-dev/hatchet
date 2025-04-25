// ❓ Declaring a Task
import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};
// HH-retries 4
// HH-func 3 input,ctx
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

// !!

// see ./worker.ts and ./run.ts for how to run the workflow
