// > Declaring a Child
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

// > Declaring a Parent

type ParentInput = {
  N: number;
};

export const parent = hatchet.task({
  name: 'parent',
  fn: async (input: ParentInput, ctx) => {
    const n = input.N;
    const promises = [];

    for (let i = 0; i < n; i++) {
      promises.push(child.run({ N: i }));
    }

    const childRes = await Promise.all(promises);
    const sum = childRes.reduce((acc, curr) => acc + curr.Value, 0);

    return {
      Result: sum,
    };
  },
});

// > Parent with Single Child
export const parentSingleChild = hatchet.task({
  name: 'parent-single-child',
  fn: async () => {
    const childRes = await child.run({ N: 1 });

    return {
      Result: childRes.Value,
    };
  },
});

// > Parent with Error Handling
export const withErrorHandling = hatchet.task({
  name: 'parent-error-handling',
  fn: async () => {
    try {
      const childRes = await child.run({ N: 1 });

      return {
        Result: childRes.Value,
      };
    } catch (error) {
      // decide how to proceed here
      return {
        Result: -1,
      };
    }
  },
});
