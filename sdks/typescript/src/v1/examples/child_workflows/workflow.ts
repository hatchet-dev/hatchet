/* eslint-disable no-plusplus */
// ❓ Declaring a Child
import { hatchet } from '../hatchet-client';

type ChildInput = {
  N: number;
};

export const child = hatchet.task({
  name: 'child',
  fn: (input: ChildInput) => {
    return {
      Value: input.N,
    };
  },
});
// !!

// ❓ Declaring a Parent

type ParentInput = {
  N: number;
};

export const parent = hatchet.task({
  name: 'parent',
  retries: 3,
  fn: async (input: ParentInput, ctx) => {
    const n = input.N;
    const requests = [];

    for (let i = 0; i < n; i++) {
      requests.push({
        workflow: child,
        input: { N: i },
      });
    }

    const result = await ctx.bulkRunNoWaitChildren(requests);

    if (ctx.retryCount() === 0) {
      throw new Error('expected');
    }

    const childRes = await Promise.all(result.map((r) => r.output));
    const sum = childRes.reduce((acc, curr) => acc + curr.Value, 0);

    return {
      Result: sum,
    };
  },
});
// !!
