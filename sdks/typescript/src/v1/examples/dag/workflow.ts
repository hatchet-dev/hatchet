import { hatchet } from '../client';

type DagInput = {
  Message: string;
};

type DagOutput = {
  reverse: {
    Original: string;
    Transformed: string;
  };
};

export const dag = hatchet.workflow<DagInput, DagOutput>({
  name: 'simple',
});

const toLower = dag.task({
  name: 'to-lower',
  fn: (input) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

dag.task({
  name: 'reverse',
  parents: [toLower],
  fn: async (input, ctx) => {
    const lower = await ctx.parentOutput(toLower);
    return {
      Original: input.Message,
      Transformed: lower.TransformedMessage.split('').reverse().join(''),
    };
  },
});
