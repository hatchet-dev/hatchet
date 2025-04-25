
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

// ❓ Declaring a Parent

type ParentInput = {
  N: number;
};

export const parent = hatchet.task({
  name: 'parent',
  fn: async (input: ParentInput, ctx) => {
    const n = input.N;
    const promises = [];

    for (let i = 0; i < n; i++) {
      promises.push(ctx.runChild(child, { N: i }));
    }

    const childRes = await Promise.all(promises);
    const sum = childRes.reduce((acc, curr) => acc + curr.Value, 0);

    return {
      Result: sum,
    };
  },
});
