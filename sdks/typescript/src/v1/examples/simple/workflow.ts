import { hatchet } from '../client';

type SimpleInput = {
  Message: string;
};

type SimpleOutput = {
  step2: {
    Original: string;
    Transformed: string;
  };
};

export const simple = hatchet.createWorkflow<SimpleInput, SimpleOutput>({
  name: 'simple',
});

const step1 = simple.addTask({
  name: 'step1',
  fn: (input) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

simple.addTask({
  name: 'step2',
  parents: [step1],
  fn: async (input, ctx) => {
    const step1Res = await ctx.parentData(step1);

    if (step1Res.TransformedMessage) {
      return {
        Original: input.Message,
        Transformed: step1Res.TransformedMessage,
      };
    }
    throw new Error('Function not implemented.');
  },
});
