import { hatchet } from '../hatchet-client';

type DagInput = {
  Message: string;
};

type DagOutput = {
  reverse: {
    Original: string;
    Transformed: string;
  };
};

// > Declaring a DAG Workflow
// First, we declare the workflow
export const dag = hatchet.workflow<DagInput, DagOutput>({
  name: 'simple',
});

// Next, we declare the tasks bound to the workflow
const toLower = dag.task({
  name: 'to-lower',
  fn: (input) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

// Next, we declare the tasks bound to the workflow
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

