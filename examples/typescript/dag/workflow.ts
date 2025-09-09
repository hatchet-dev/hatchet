import { hatchet } from '../hatchet-client';

// > Declaring Types
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

// > Declaring Tasks
// Next, we declare the tasks bound to the workflow
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

// > Accessing Parent Outputs
dag.task({
  name: 'task-with-parent-output',
  parents: [toLower],
  fn: async (input, ctx) => {
    const lower = await ctx.parentOutput(toLower);
    return {
      Original: input.Message,
      Transformed: lower.TransformedMessage.split('').reverse().join(''),
    };
  },
});
