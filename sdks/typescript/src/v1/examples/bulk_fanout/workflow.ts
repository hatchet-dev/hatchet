import { hatchet } from '../hatchet-client';

export type ParentInput = { n: number };

export const bulkChild = hatchet.task({
  name: 'bulk-child',
  fn: async (input: { i: number }) => {
    return { i: input.i };
  },
});

export const bulkParentWorkflow = hatchet.workflow({
  name: 'bulk-parent',
});

bulkParentWorkflow.task({
  name: 'spawn',
  fn: async (input, ctx) => {
    const typed = input as ParentInput;
    const children = Array.from({ length: typed.n }, (_, i) => ({
      workflow: bulkChild,
      input: { i },
    }));

    const results = await ctx.bulkRunChildren(children);

    return { results };
  },
});
