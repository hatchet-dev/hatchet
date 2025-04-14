// ❓ Declaring a Task
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type ChildInput = {
  Message: string;
};

export type ParentInput = {
  Message: string;
};

export const child = hatchet.workflow<ChildInput>({
  name: 'child',
});

export const child1 = child.task({
  name: 'child1',
  fn: (input: ChildInput, ctx) => {
    ctx.log('hello from the child1');
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

export const child2 = child.task({
  name: 'child2',
  fn: (input: ChildInput, ctx) => {
    ctx.log('hello from the child2');
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
      TransformedMessage: 'not implemented',
    };
  },
});

// !!

// see ./worker.ts and ./run.ts for how to run the workflow
