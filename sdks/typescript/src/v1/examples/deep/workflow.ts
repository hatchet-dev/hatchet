import sleep from '@hatchet/util/sleep';
import { hatchet } from '../client';

type SimpleInput = {
  Message: string;
  N: number;
};

type Output = {
  transformer: {
    Sum: number;
  };
};

export const child1 = hatchet.workflow<SimpleInput, Output>({
  name: 'child1',
});

child1.task({
  name: 'transformer',
  fn: () => {
    sleep(15);
    return {
      Sum: 1,
    };
  },
});

export const child2 = hatchet.workflow<SimpleInput, Output>({
  name: 'child2',
});

child2.task({
  name: 'transformer',
  fn: async (input, ctx) => {
    const count = input.N;
    const promises = Array(count)
      .fill(null)
      .map(() => ({ workflow: child1, input }));

    const results = await ctx.bulkRunChildren(promises);

    sleep(15);
    return {
      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),
    };
  },
});

export const child3 = hatchet.workflow<SimpleInput, Output>({
  name: 'child3',
});

child3.task({
  name: 'transformer',
  fn: async (input, ctx) => {
    const count = input.N;
    const promises = Array(count)
      .fill(null)
      .map(() => ({ workflow: child2, input }));

    const results = await ctx.bulkRunChildren(promises);

    return {
      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),
    };
  },
});

export const child4 = hatchet.workflow<SimpleInput, Output>({
  name: 'child4',
});

child4.task({
  name: 'transformer',
  fn: async (input, ctx) => {
    const count = input.N;
    const promises = Array(count)
      .fill(null)
      .map(() => ({ workflow: child3, input }));

    const results = await ctx.bulkRunChildren(promises);

    return {
      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),
    };
  },
});

export const child5 = hatchet.workflow<SimpleInput, Output>({
  name: 'child5',
});

child5.task({
  name: 'transformer',
  fn: async (input, ctx) => {
    const count = input.N;
    const promises = Array(count)
      .fill(null)
      .map(() => ({ workflow: child4, input }));

    const results = await ctx.bulkRunChildren(promises);

    return {
      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),
    };
  },
});

export const parent = hatchet.workflow<SimpleInput, { parent: Output['transformer'] }>({
  name: 'parent',
});

parent.task({
  name: 'parent',
  fn: async (input, ctx) => {
    const count = input.N; // Random number between 2-4
    const promises = Array(count)
      .fill(null)
      .map(() => ({ workflow: child5, input }));

    const results = await ctx.bulkRunChildren(promises);

    return {
      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),
    };
  },
});
