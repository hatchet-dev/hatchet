import { hatchet } from '../hatchet-client';

type SimpleInput = {
  Message: string;
};

type SimpleOutput = {
  TransformedMessage: string;
};

export const declaredType = hatchet.task<SimpleInput, SimpleOutput>({
  name: 'declared-type',
  fn: (input) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

export const inferredType = hatchet.task({
  name: 'inferred-type',
  fn: (input: SimpleInput) => {
    return {
      TransformedMessage: input.Message.toUpperCase(),
    };
  },
});

export const inferredTypeDurable = hatchet.durableTask({
  name: 'inferred-type-durable',
  fn: async (input: SimpleInput, ctx) => {
    // await ctx.sleepFor('5s');

    return {
      TransformedMessage: input.Message.toUpperCase(),
    };
  },
});

export const crazyWorkflow = hatchet.workflow<any, any>({
  name: 'crazy-workflow',
});

const step1 = crazyWorkflow.task(declaredType);
// crazyWorkflow.task(inferredTypeDurable);

crazyWorkflow.task({
  parents: [step1],
  ...inferredType.taskDef,
});
