// ❓ Declaring a Task
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type ChildInput = {
  Message: string;
};

export type ParentInput = {
  Message: string;
};

export const child = hatchet.task({
  name: 'child',
  fn: (input: ChildInput) => {
    const largePayload = new Array(1024 * 1024).fill('a').join('');

    return {
      TransformedMessage: largePayload,
    };
  },
});

export const parent = hatchet.task({
  name: 'parent',
  timeout: '10m',
  fn: async (input: ParentInput, ctx) => {
    // lets generate large payload 1 mb
    const largePayload = new Array(1024 * 1024).fill('a').join('');

    // Send the large payload 100 times
    const num = 1000;

    const children = [];
    for (let i = 0; i < num; i += 1) {
      children.push({
        workflow: child,
        input: {
          Message: `Iteration ${i + 1}: ${largePayload}`,
        },
      });
    }

    await ctx.bulkRunNoWaitChildren(children);

    return {
      TransformedMessage: 'done',
    };
  },
});

// !!

// see ./worker.ts and ./run.ts for how to run the workflow
