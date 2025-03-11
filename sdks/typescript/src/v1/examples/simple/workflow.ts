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

export const simple = hatchet.workflow<SimpleInput, SimpleOutput>({
  name: 'simple',
});

const step1 = simple.task({
  name: 'step1',
  fn: (input) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

simple.task({
  name: 'step2',
  parents: [step1],
  timeout: '15s',
  fn: async (input, ctx) => {
    const step1Res = await ctx.parentData(step1);
    return {
      Original: input.Message,
      Transformed: step1Res.TransformedMessage,
    };
  },
});
