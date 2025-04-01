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
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

export const parent = hatchet.task({
  name: 'parent',
  fn: async (input: ParentInput, ctx) => {
    const c = await ctx.runChild(child, {
      Message: input.Message,
    });

    return {
      TransformedMessage: c.TransformedMessage,
    };
  },
});

// !!

// see ./worker.ts and ./run.ts for how to run the workflow
