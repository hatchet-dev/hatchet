// > Declaring a Task
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
    ctx.logger.info('hello from the child1', { hello: 'moon' });
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

export const child2 = child.task({
  name: 'child2',
  fn: (input: ChildInput, ctx) => {
    ctx.logger.info('hello from the child2');
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

export const child3 = child.task({
  name: 'child3',
  parents: [child1, child2],
  fn: (input: ChildInput, ctx) => {
    ctx.logger.info('hello from the child3');
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

export const parent = hatchet.task({
  name: 'parent',
  fn: async (input: ParentInput, ctx) => {
    const c = await child.run({
      Message: input.Message,
    });

    return {
      TransformedMessage: 'not implemented',
    };
  },
});


// see ./worker.ts and ./run.ts for how to run the workflow
