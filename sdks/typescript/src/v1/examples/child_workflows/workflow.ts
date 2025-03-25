// ❓ Declaring a Child

import { hatchet } from '../hatchet-client';

type ChildInput = {
  N: number;
};

type ChildOutput = {
  value: {
    Value: number;
  };
};

export const child = hatchet.workflow<ChildInput, ChildOutput>({
  name: 'child',
});

child.task({
  name: 'value',
  fn: (input) => {
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

type ParentOutput = {
  sum: {
    Result: number;
  };
};

export const parent = hatchet.workflow<ParentInput, ParentOutput>({
  name: 'parent',
});

parent.task({
  name: 'sum',
  fn: async (input, ctx) => {
    const n = input.N;
    const promises = [];
    // eslint-disable-next-line no-plusplus
    for (let i = 0; i < n; i++) {
      promises.push(ctx.runChild(child, { N: i }));
    }

    const childRes = await Promise.all(promises);
    const sum = childRes.reduce((acc, curr) => acc + curr.value.Value, 0);

    return {
      Result: sum,
    };
  },
});

// !!
